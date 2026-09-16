param([Parameter(Mandatory=$true)][string]$Executable,[Parameter(Mandatory=$true)][string]$OutputDirectory,[string[]]$Scenarios=@('minimal','login','list','images'))
$ErrorActionPreference='Stop'; Add-Type -AssemblyName System.Drawing
Add-Type @'
using System; using System.Runtime.InteropServices; using System.Text;
public static class BaselineWindow { [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
[UnmanagedFunctionPointer(CallingConvention.Winapi)] delegate bool EnumProc(IntPtr h,IntPtr l);
[DllImport("user32.dll")] static extern bool EnumWindows(EnumProc p,IntPtr l);
[DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
[DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h,StringBuilder s,int n);
[DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h,out RECT r);
[DllImport("user32.dll")] public static extern bool MoveWindow(IntPtr h,int x,int y,int w,int height,bool repaint);
[DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h,IntPtr dc,uint flags);
public static IntPtr Find(uint wanted) { IntPtr found=IntPtr.Zero; EnumWindows(delegate(IntPtr h,IntPtr l){uint pid;GetWindowThreadProcessId(h,out pid);if(pid!=wanted)return true;var s=new StringBuilder(256);GetWindowText(h,s,s.Capacity);if(s.ToString().StartsWith("dxui native baseline:")){found=h;return false;}return true;},IntPtr.Zero);return found; } }
'@
New-Item -ItemType Directory -Force $OutputDirectory|Out-Null
foreach($renderer in @('auto','software')){foreach($scenario in $Scenarios){
  $name="$scenario-$renderer";$diag=Join-Path $OutputDirectory "$name.jsonl";$args="-scenario `"$scenario`" -mode idle -duration 8s -diagnostic-file `"$diag`"";if($renderer-eq'software'){$args+=' -software'}
  $p=Start-Process -FilePath $Executable -ArgumentList $args -PassThru
  $limit=(Get-Date).AddSeconds(8);$window=[IntPtr]::Zero;do{Start-Sleep -Milliseconds 100;$window=[BaselineWindow]::Find($p.Id)}until($window-ne[IntPtr]::Zero-or(Get-Date)-gt$limit)
  if($window-eq[IntPtr]::Zero){$p|Stop-Process;throw "no dxui window for $name"}
  if(-not[BaselineWindow]::MoveWindow($window,0,0,800,600,$true)){throw "MoveWindow failed for $name"};Start-Sleep -Seconds 1
  $r=[BaselineWindow+RECT]::new();if(-not[BaselineWindow]::GetWindowRect($window,[ref]$r)){throw "GetWindowRect failed for $name"}
  $bmp=[Drawing.Bitmap]::new($r.Right-$r.Left,$r.Bottom-$r.Top);$g=[Drawing.Graphics]::FromImage($bmp)
  try{$dc=$g.GetHdc();try{if(-not[BaselineWindow]::PrintWindow($window,$dc,2)){throw "PrintWindow failed for $name"}}finally{$g.ReleaseHdc($dc)};$bmp.Save((Join-Path $OutputDirectory "$name.png"),[Drawing.Imaging.ImageFormat]::Png)}finally{$g.Dispose();$bmp.Dispose()}
  $p.WaitForExit();if($p.ExitCode-ne0){throw "$name exited $($p.ExitCode)"}
}}
