param(
  [string]$Prefix = "$env:ProgramFiles\InduForge\NodeAgent",
  [string]$ConfigDir = "$env:ProgramData\InduForge\NodeAgent",
  [string]$ServerUrl = "",
  [string]$EnrollmentCode = "",
  [switch]$NoService
)

$ErrorActionPreference = 'Stop'
$serviceName = 'InduForgeNodeAgent'

if ($ServerUrl -match "[\r\n]" -or $EnrollmentCode -match "[\r\n]") {
  throw 'invalid enrollment value'
}
if (!(Test-Path "$PSScriptRoot\bin\node-agent.exe" -PathType Leaf)) {
  throw 'node-agent.exe is missing from this package'
}
if (!(Test-Path "$PSScriptRoot\BUILD_VERSION" -PathType Leaf)) {
  throw 'package BUILD_VERSION is missing'
}
if (!(Test-Path "$PSScriptRoot\capabilities\collector\win-x64\industrial_collector.exe" -PathType Leaf)) {
  throw 'collector is missing from this package'
}
$buildVersion = (Get-Content -LiteralPath "$PSScriptRoot\BUILD_VERSION" -Raw).Trim()
if (!$buildVersion) {
  throw 'package BUILD_VERSION is empty'
}

function Set-ConfigString {
  param([string]$Path, [string]$Key, [string]$Value)

  # ConvertTo-Json 生成合法的双引号 YAML 标量，并可正确保留 URL 中的特殊字符。
  $encoded = ConvertTo-Json -InputObject $Value -Compress
  $lines = @(Get-Content -LiteralPath $Path)
  $pattern = '^\s*' + [regex]::Escape($Key) + '\s*:'
  $found = $false
  $updated = foreach ($line in $lines) {
    if ($line -match $pattern) {
      $found = $true
      "$Key`: $encoded"
    } else {
      $line
    }
  }
  if (!$found) {
    $updated += "$Key`: $encoded"
  }
  [IO.File]::WriteAllLines($Path, [string[]]$updated, (New-Object System.Text.UTF8Encoding($false)))
}

function Invoke-Sc {
  param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments)
  & sc.exe @Arguments | Out-Null
  if ($LASTEXITCODE -ne 0) {
    throw "sc.exe $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

function Stop-ServiceForUpdate {
  param([System.ServiceProcess.ServiceController]$Service)
  if ($Service.Status -eq [System.ServiceProcess.ServiceControllerStatus]::Stopped) {
    return
  }
  & sc.exe stop $serviceName | Out-Null
  if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 1062) {
    throw "failed to stop $serviceName (exit code $LASTEXITCODE)"
  }
  $Service.Refresh()
  $Service.WaitForStatus([System.ServiceProcess.ServiceControllerStatus]::Stopped, [TimeSpan]::FromSeconds(45))
}

New-Item -ItemType Directory -Force -Path "$Prefix\bin", "$Prefix\capabilities\collector\win-x64", "$Prefix\data", "$Prefix\runtime", "$Prefix\releases", "$Prefix\logs", $ConfigDir | Out-Null
Copy-Item "$PSScriptRoot\bin\node-agent.exe" "$Prefix\bin\node-agent.exe" -Force
Copy-Item "$PSScriptRoot\capabilities\collector\win-x64\industrial_collector.exe" "$Prefix\capabilities\collector\win-x64\industrial_collector.exe" -Force
if (!(Test-Path "$ConfigDir\config.yaml" -PathType Leaf)) {
  Copy-Item "$PSScriptRoot\config.yaml" "$ConfigDir\config.yaml"
  (Get-Content -LiteralPath "$ConfigDir\config.yaml" -Raw).Replace('__BUILD_VERSION__', $buildVersion) | Set-Content -LiteralPath "$ConfigDir\config.yaml" -Encoding utf8 -NoNewline
}
if ($ServerUrl) { Set-ConfigString -Path "$ConfigDir\config.yaml" -Key 'serverUrl' -Value $ServerUrl }
if ($EnrollmentCode) { Set-ConfigString -Path "$ConfigDir\config.yaml" -Key 'enrollmentCode' -Value $EnrollmentCode }

$wrapper = "$Prefix\bin\node-agent-service.cmd"
@"
@echo off
set "NODE_AGENT_WORKDIR=$Prefix"
set "NODE_AGENT_CONFIG=$ConfigDir\config.yaml"
set "NODE_AGENT_DATA_DIR=$Prefix\data"
"$Prefix\bin\node-agent.exe" --service
"@ | Set-Content -LiteralPath $wrapper -Encoding ascii -NoNewline

if (!$NoService) {
  # SCM 不支持为单个服务声明环境变量，故通过受控 cmd 包装器设置运行目录和配置路径。
  $commandLine = "`"$env:SystemRoot\System32\cmd.exe`" /d /s /c `"`"$wrapper`"`""
  $existing = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
  $wasRunning = $existing -and $existing.Status -ne [System.ServiceProcess.ServiceControllerStatus]::Stopped
  if ($existing) {
    Stop-ServiceForUpdate -Service $existing
    Invoke-Sc config $serviceName "binPath= $commandLine" 'start= auto'
  } else {
    Invoke-Sc create $serviceName "binPath= $commandLine" 'start= auto'
  }

  # 仅写入完整注册地址时自动首启；升级已运行服务则保持其原有运行意图。
  if (($ServerUrl -and $EnrollmentCode) -or $wasRunning) {
    & sc.exe start $serviceName | Out-Null
    if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 1056) {
      throw "failed to start $serviceName (exit code $LASTEXITCODE)"
    }
  } else {
    Write-Output 'Service installed but not started: supply -ServerUrl and -EnrollmentCode after approval.'
  }
}

Write-Output "Installed NodeAgent $buildVersion. Capability templates can claim a node but stay disabled until local release configuration is complete."
