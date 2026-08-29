param(
  [string]$Prefix = "$env:ProgramFiles\InduForge\NodeAgent",
  [string]$ConfigDir = "$env:ProgramData\InduForge\NodeAgent",
  [switch]$Purge
)

$ErrorActionPreference = 'Stop'
$serviceName = 'InduForgeNodeAgent'

function Remove-AgentService {
  $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
  if (!$service) {
    return
  }
  if ($service.Status -ne [System.ServiceProcess.ServiceControllerStatus]::Stopped) {
    & sc.exe stop $serviceName | Out-Null
    if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 1062) {
      throw "failed to stop $serviceName (exit code $LASTEXITCODE)"
    }
    $service.Refresh()
    $service.WaitForStatus([System.ServiceProcess.ServiceControllerStatus]::Stopped, [TimeSpan]::FromSeconds(45))
  }

  & sc.exe delete $serviceName | Out-Null
  if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 1072) {
    throw "failed to delete $serviceName (exit code $LASTEXITCODE)"
  }

  # delete 后 SCM 可能短暂保留 marked-for-delete 服务；等待其释放，方便立即重装。
  $deadline = [DateTime]::UtcNow.AddSeconds(20)
  while ([DateTime]::UtcNow -lt $deadline) {
    if (!(Get-Service -Name $serviceName -ErrorAction SilentlyContinue)) {
      return
    }
    Start-Sleep -Milliseconds 250
  }
  throw "$serviceName is still pending deletion; close any Services console that is inspecting it and retry."
}

Remove-AgentService
Remove-Item "$Prefix\bin\node-agent.exe", "$Prefix\bin\node-agent-service.cmd" -Force -ErrorAction SilentlyContinue

if ($Purge) {
  foreach ($path in @($Prefix, $ConfigDir)) {
    $full = [IO.Path]::GetFullPath($path)
    if ($full -eq [IO.Path]::GetPathRoot($full) -or $full -eq $env:ProgramFiles -or $full -eq $env:ProgramData -or $full -notmatch '(?i)(induforge|nodeagent)') {
      throw "refusing unsafe purge target: $full"
    }
  }
  Remove-Item $Prefix, $ConfigDir -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Output 'NodeAgent binary removed. Use -Purge to remove retained config and state.'
