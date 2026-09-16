# Windows native baseline runner

`measure.ps1` builds and runs the fixed `examples/native_baseline` scenes and
records process and dxui diagnostics without adding work to the UI loop. The
PowerShell process remains resident for each run; it does not start a shell per
sample. Child console windows are hidden while the measured SDL window remains
visible.

Use `-Quick` only to validate the tooling. A complete baseline uses three
independent 30-second steady runs for every scene/renderer, ten-minute idle
runs for minimal and login, and 30-minute interaction runs for login/list.

```powershell
pwsh -File tools/native-baseline/measure.ps1 -Quick
pwsh -File tools/native-baseline/measure.ps1
```

Windows metric sources are recorded in each run's metadata. `private_commit_bytes`
comes from `GetProcessMemoryInfo.PROCESS_MEMORY_COUNTERS_EX.PrivateUsage`;
`working_set_bytes` comes from `WorkingSetSize`; `private_working_set_bytes`
is calculated by `QueryWorkingSet` and counts resident pages whose Shared bit
is clear. CPU is the change in `TotalProcessorTime`, divided by elapsed wall
time and logical processor count. A failed API call produces a sample error,
never a synthetic zero.
