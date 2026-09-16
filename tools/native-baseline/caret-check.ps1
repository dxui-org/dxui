param([Parameter(Mandatory=$true)][string]$Executable,[Parameter(Mandatory=$true)][string]$OutputDirectory)
$ErrorActionPreference='Stop'
Add-Type @'
using System; using System.Runtime.InteropServices;
public static class CaretInput { [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
[DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h,out RECT r);
[DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
[DllImport("user32.dll")] public static extern bool SetCursorPos(int x,int y);
[DllImport("user32.dll")] public static extern void mouse_event(uint f,uint x,uint y,uint d,UIntPtr e);
[DllImport("user32.dll")] public static extern void keybd_event(byte key,byte scan,uint flags,UIntPtr extra); }
'@
New-Item -ItemType Directory -Force $OutputDirectory|Out-Null
foreach($renderer in @('auto','software')){
  $diag=Join-Path $OutputDirectory "caret-$renderer.jsonl";$args="-scenario login -mode caret -duration 25s -diagnostic-sample 1s -diagnostic-file `"$diag`"";if($renderer-eq'software'){$args+=' -software'}
  $p=Start-Process $Executable -ArgumentList $args -PassThru;$limit=(Get-Date).AddSeconds(8);do{Start-Sleep -Milliseconds 100;$p.Refresh()}until($p.MainWindowHandle-ne[IntPtr]::Zero-or(Get-Date)-gt$limit)
  if($p.MainWindowHandle-eq[IntPtr]::Zero){throw "no window for $renderer"};$r=[CaretInput+RECT]::new();[void][CaretInput]::GetWindowRect($p.MainWindowHandle,[ref]$r);[void][CaretInput]::SetForegroundWindow($p.MainWindowHandle)
  Start-Sleep 3;[CaretInput]::keybd_event(9,0,0,[UIntPtr]::Zero);[CaretInput]::keybd_event(9,0,2,[UIntPtr]::Zero)
  Start-Sleep 12;1..2|ForEach-Object{[CaretInput]::keybd_event(9,0,0,[UIntPtr]::Zero);[CaretInput]::keybd_event(9,0,2,[UIntPtr]::Zero);Start-Sleep -Milliseconds 80}
  $p.WaitForExit();if($p.ExitCode-ne0){throw "$renderer exited $($p.ExitCode)"}
}
