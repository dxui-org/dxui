param(
    [switch]$Quick,
    [string]$OutputRoot = "",
    [int]$SampleSeconds = 1
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if (-not $OutputRoot) {
    $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
    $OutputRoot = Join-Path $repo "docs/performance/evidence/native-baseline/$stamp"
}
$OutputRoot = [IO.Path]::GetFullPath($OutputRoot)
New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null

Add-Type @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
public static class NativeBaselineProcess {
  [StructLayout(LayoutKind.Sequential)] public struct PMCEX {
    public uint cb, PageFaultCount; public UIntPtr PeakWorkingSetSize, WorkingSetSize;
    public UIntPtr QuotaPeakPagedPoolUsage, QuotaPagedPoolUsage, QuotaPeakNonPagedPoolUsage, QuotaNonPagedPoolUsage;
    public UIntPtr PagefileUsage, PeakPagefileUsage, PrivateUsage;
  }
  [DllImport("psapi.dll", SetLastError=true)] static extern bool GetProcessMemoryInfo(IntPtr h, out PMCEX p, uint cb);
  [DllImport("psapi.dll", SetLastError=true)] static extern bool QueryWorkingSet(IntPtr h, IntPtr p, int cb);
  public static ulong[] Memory(IntPtr h) {
    PMCEX p; p.cb=(uint)Marshal.SizeOf(typeof(PMCEX));
    if(!GetProcessMemoryInfo(h,out p,p.cb)) throw new Win32Exception();
    int bytes=1024*1024;
    for(int attempt=0;attempt<8;attempt++) {
      IntPtr b=Marshal.AllocHGlobal(bytes);
      try {
        if(QueryWorkingSet(h,b,bytes)) {
          long count = IntPtr.Size==8 ? Marshal.ReadInt64(b) : Marshal.ReadInt32(b);
          ulong privatePages=0;
          for(long i=0;i<count;i++) {
            long offset=IntPtr.Size+i*IntPtr.Size;
            ulong flags=IntPtr.Size==8 ? unchecked((ulong)Marshal.ReadInt64(b,(int)offset)) : unchecked((uint)Marshal.ReadInt32(b,(int)offset));
            if((flags & (1UL<<8))==0) privatePages++;
          }
          return new ulong[]{p.PrivateUsage.ToUInt64(),p.WorkingSetSize.ToUInt64(),privatePages*(ulong)Environment.SystemPageSize,p.PeakPagefileUsage.ToUInt64(),p.PeakWorkingSetSize.ToUInt64()};
        }
        int error=Marshal.GetLastWin32Error(); if(error!=24) throw new Win32Exception(error); bytes*=2;
      } finally { Marshal.FreeHGlobal(b); }
    }
    throw new Win32Exception(24,"QueryWorkingSet buffer did not converge");
  }
}
'@

function Save-Json($Value, $Path) { $Value | ConvertTo-Json -Depth 8 | Set-Content -Encoding utf8 -LiteralPath $Path }

$envInfo = [ordered]@{
    captured_at = (Get-Date).ToString('o')
    git_revision = (git -C $repo rev-parse HEAD)
    git_status = @(git -C $repo status --short)
    os = (Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,BuildNumber,OSArchitecture,TotalVisibleMemorySize)
    cpu = @(Get-CimInstance Win32_Processor | Select-Object Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed)
    gpu = @(Get-CimInstance Win32_VideoController | Select-Object Name,DriverVersion,AdapterRAM,VideoProcessor)
    go_version = (go version)
    go_env = (go env GOOS GOARCH CGO_ENABLED GOROOT GOPATH)
    sample_interval_seconds = $SampleSeconds
    metric_sources = [ordered]@{
        private_commit_bytes = 'GetProcessMemoryInfo PROCESS_MEMORY_COUNTERS_EX.PrivateUsage'
        working_set_bytes = 'GetProcessMemoryInfo PROCESS_MEMORY_COUNTERS_EX.WorkingSetSize'
        private_working_set_bytes = 'QueryWorkingSet resident pages with PSAPI_WORKING_SET_BLOCK.Shared clear'
        cpu_percent = 'System.Diagnostics.Process.TotalProcessorTime delta / wall-time delta / logical processor count'
    }
}
Save-Json $envInfo (Join-Path $OutputRoot 'environment.json')

$bin = Join-Path $OutputRoot 'dxui-native-baseline.exe'
$env:CGO_ENABLED = '0'
& go build -trimpath "-ldflags=-s -w" -o $bin ./examples/native_baseline 2>&1 | Tee-Object -FilePath (Join-Path $OutputRoot 'build.log')
if ($LASTEXITCODE -ne 0) { throw "baseline build failed: $LASTEXITCODE" }
Save-Json ([ordered]@{ path=$bin; bytes=(Get-Item $bin).Length; cgo_enabled='0'; command='go build -trimpath -ldflags=-s -w -o <evidence>/dxui-native-baseline.exe ./examples/native_baseline' }) (Join-Path $OutputRoot 'build.json')

function Invoke-Run([string]$Scenario,[string]$Renderer,[string]$Mode,[TimeSpan]$Duration,[int]$Iteration) {
    $name = "$Scenario-$Renderer-$Mode-$Iteration"
    $dir = Join-Path $OutputRoot $name; New-Item -ItemType Directory -Force -Path $dir | Out-Null
    $diag = Join-Path $dir 'diagnostics.jsonl'; $stdout=Join-Path $dir 'stdout.log'; $stderr=Join-Path $dir 'stderr.log'
    $args=@('-scenario',$Scenario,'-mode',$Mode,'-duration',("{0}s" -f [int]$Duration.TotalSeconds),'-diagnostic-sample',("{0}s" -f $SampleSeconds),'-diagnostic-file',$diag)
    if($Renderer -eq 'software'){$args += '-software'}
    $si=[Diagnostics.ProcessStartInfo]::new($bin); $si.UseShellExecute=$false; $si.CreateNoWindow=$true; $si.WindowStyle=[Diagnostics.ProcessWindowStyle]::Hidden
    $si.RedirectStandardOutput=$true; $si.RedirectStandardError=$true
    # Windows PowerShell 5.1 does not expose ProcessStartInfo.ArgumentList.
    # These generated arguments contain no quotes; quote each value so paths
    # with spaces still reach the child as one argument.
    $si.Arguments = (($args | ForEach-Object { '"' + ($_ -replace '"','\"') + '"' }) -join ' ')
    $p=[Diagnostics.Process]::new(); $p.StartInfo=$si; $start=Get-Date; [void]$p.Start()
    $samples=[Collections.Generic.List[object]]::new(); $previousCPU=$p.TotalProcessorTime; $previousAt=Get-Date
    while(-not $p.HasExited){
        $now=Get-Date; $errorText=$null
        try { $p.Refresh(); $m=[NativeBaselineProcess]::Memory($p.Handle); $cpu=$p.TotalProcessorTime; $wall=($now-$previousAt).TotalSeconds; $cpuPct=100*($cpu-$previousCPU).TotalSeconds/$wall/[Environment]::ProcessorCount; $previousCPU=$cpu; $previousAt=$now
          $samples.Add([pscustomobject]@{timestamp=$now.ToString('o');elapsed_ms=[int64]($now-$start).TotalMilliseconds;cpu_percent=$cpuPct;private_commit_bytes=$m[0];working_set_bytes=$m[1];private_working_set_bytes=$m[2];peak_private_commit_bytes=$m[3];peak_working_set_bytes=$m[4];sample_error=$null})
        } catch { $samples.Add([pscustomobject]@{timestamp=$now.ToString('o');elapsed_ms=[int64]($now-$start).TotalMilliseconds;cpu_percent=$null;private_commit_bytes=$null;working_set_bytes=$null;private_working_set_bytes=$null;peak_private_commit_bytes=$null;peak_working_set_bytes=$null;sample_error=$_.Exception.Message}) }
        Start-Sleep -Seconds $SampleSeconds
    }
    $p.WaitForExit(); $p.StandardOutput.ReadToEnd() | Set-Content -Encoding utf8 $stdout; $p.StandardError.ReadToEnd() | Set-Content -Encoding utf8 $stderr
    $samples | Export-Csv -NoTypeInformation -Encoding utf8 (Join-Path $dir 'process.csv')
    Save-Json ([ordered]@{name=$name;scenario=$Scenario;renderer_request=$Renderer;mode=$Mode;iteration=$Iteration;duration_seconds=$Duration.TotalSeconds;sample_seconds=$SampleSeconds;process_id=$p.Id;exit_code=$p.ExitCode;started=$start.ToString('o');ended=(Get-Date).ToString('o');arguments=$args}) (Join-Path $dir 'run.json')
    if($p.ExitCode -ne 0){throw "$name exited $($p.ExitCode); see $stderr"}
}

if($Quick){
    foreach($renderer in @('auto','software')){foreach($scenario in @('minimal','login','list','images')){Invoke-Run $scenario $renderer 'steady' ([TimeSpan]::FromSeconds(5)) 1}}
} else {
    foreach($renderer in @('auto','software')){foreach($scenario in @('minimal','login','list','images')){1..3|ForEach-Object{Invoke-Run $scenario $renderer 'steady' ([TimeSpan]::FromSeconds(30)) $_}}}
    foreach($renderer in @('auto','software')){foreach($scenario in @('minimal','login')){Invoke-Run $scenario $renderer 'idle' ([TimeSpan]::FromMinutes(10)) 1}}
    foreach($renderer in @('auto','software')){foreach($scenario in @('login','list')){Invoke-Run $scenario $renderer 'interact' ([TimeSpan]::FromMinutes(30)) 1}}
}
Write-Output $OutputRoot
