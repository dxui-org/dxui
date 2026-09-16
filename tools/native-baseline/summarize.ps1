param([Parameter(Mandatory=$true)][string]$EvidenceRoot)
$ErrorActionPreference='Stop'
$EvidenceRoot=(Resolve-Path $EvidenceRoot).Path
function N($v){if($null -eq $v -or $v -eq ''){return $null};return [double]$v}
function Stats($values){
  $a=@($values|Where-Object{$null-ne$_}|Sort-Object); if($a.Count-eq 0){return $null}
  [ordered]@{min=[double]$a[0];mean=[double](($a|Measure-Object -Average).Average);max=[double]$a[-1]}
}
$runs=@()
Get-ChildItem -Directory $EvidenceRoot | Sort-Object Name | ForEach-Object {
  $runFile=Join-Path $_.FullName 'run.json'; if(-not(Test-Path $runFile)){return}
  $run=Get-Content -Raw $runFile|ConvertFrom-Json; $p=@(Import-Csv (Join-Path $_.FullName 'process.csv'))
  $d=@(Get-Content (Join-Path $_.FullName 'diagnostics.jsonl')|ForEach-Object{$_|ConvertFrom-Json})
  $shown=$d|Where-Object kind -eq shown|Select-Object -First 1; $last=$d|Where-Object kind -eq complete|Select-Object -Last 1; if(-not$last){$last=$d|Select-Object -Last 1}
  $valid=@($p|Where-Object{-not $_.sample_error}); $steady=@($valid|Where-Object{[int64]$_.elapsed_ms -ge 2000})
  $runs += [pscustomobject][ordered]@{
    run=$run.name;scenario=$run.scenario;renderer_request=$run.renderer_request;renderer_actual=$shown.diagnostics.RendererName;mode=$run.mode;iteration=$run.iteration
    startup_to_onshown_ms=$shown.at_ms;executable_bytes=(Get-Content -Raw (Join-Path $EvidenceRoot 'build.json')|ConvertFrom-Json).bytes
    private_commit=Stats @($steady|ForEach-Object{N $_.private_commit_bytes});private_working_set=Stats @($steady|ForEach-Object{N $_.private_working_set_bytes});working_set=Stats @($steady|ForEach-Object{N $_.working_set_bytes});cpu_percent=Stats @($steady|ForEach-Object{N $_.cpu_percent})
    peak_private_commit_bytes=($valid|ForEach-Object{N $_.peak_private_commit_bytes}|Measure-Object -Maximum).Maximum;peak_working_set_bytes=($valid|ForEach-Object{N $_.peak_working_set_bytes}|Measure-Object -Maximum).Maximum
    frames_start=$shown.diagnostics.FrameCount;frames_end=$last.diagnostics.FrameCount;events=$last.diagnostics.EventCount;builds=$last.diagnostics.BuildCount;layouts=$last.diagnostics.LayoutCount;paints=$last.diagnostics.PaintCount
    go_heap_bytes=$last.diagnostics.GoHeapBytes;go_total_alloc_bytes=$last.diagnostics.GoTotalAllocBytes;go_heap_objects=$last.diagnostics.GoHeapObjects;go_mallocs=$last.diagnostics.GoMallocs;goroutines=$last.diagnostics.Goroutines
    cache_bytes=$last.diagnostics.CacheBytes;cache_entries=$last.diagnostics.CacheEntries;textures_created=$last.diagnostics.TextureCreates;textures_destroyed=$last.diagnostics.TextureDestroys;renderer_resources=$last.diagnostics.RendererResources;image_resources=$last.diagnostics.ImageResources;font_resources=$last.diagnostics.FontResources
    frame_p50_ns=$last.diagnostics.FrameTime.P50NS;frame_p95_ns=$last.diagnostics.FrameTime.P95NS;frame_p99_ns=$last.diagnostics.FrameTime.P99NS;event_present_p50_ns=$last.diagnostics.EventToPresent.P50NS;event_present_p95_ns=$last.diagnostics.EventToPresent.P95NS;event_present_p99_ns=$last.diagnostics.EventToPresent.P99NS
    sample_errors=@($p|Where-Object sample_error).Count
  }
}
$runs|ConvertTo-Json -Depth 8|Set-Content -Encoding utf8 (Join-Path $EvidenceRoot 'summary.json')
$runs|Select-Object run,scenario,renderer_request,renderer_actual,mode,iteration,startup_to_onshown_ms,@{n='private_commit_mean';e={$_.private_commit.mean}},@{n='private_ws_mean';e={$_.private_working_set.mean}},@{n='working_set_mean';e={$_.working_set.mean}},@{n='cpu_mean';e={$_.cpu_percent.mean}},peak_private_commit_bytes,peak_working_set_bytes,frames_start,frames_end,events,builds,layouts,paints,go_heap_bytes,go_total_alloc_bytes,go_heap_objects,cache_bytes,cache_entries,textures_created,textures_destroyed,renderer_resources,frame_p50_ns,frame_p95_ns,frame_p99_ns,event_present_p50_ns,event_present_p95_ns,event_present_p99_ns,sample_errors|Export-Csv -NoTypeInformation -Encoding utf8 (Join-Path $EvidenceRoot 'summary.csv')
$runs|Format-Table run,startup_to_onshown_ms,@{n='PC MiB';e={[math]::Round($_.private_commit.mean/1MB,2)}},@{n='PWS MiB';e={[math]::Round($_.private_working_set.mean/1MB,2)}},@{n='CPU %';e={[math]::Round($_.cpu_percent.mean,3)}},frames_end,renderer_actual -AutoSize
