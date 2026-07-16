param(
  [string]$Python = "python",
  [ValidateSet("normal", "startup", "alarm", "noisy", "intermittent")]
  [string]$Scenario = "normal"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$PackageRoot = Join-Path $Root "industrial_sim"

$modules = @(
  @{ Name = "Modbus TCP"; Script = Join-Path $PackageRoot "modbus_sim.py"; Port = "18502" },
  @{ Name = "OPC UA"; Script = Join-Path $PackageRoot "opcua_sim.py"; Port = "18540" },
  @{ Name = "S7"; Script = Join-Path $PackageRoot "s7_sim.py"; Port = "18503" }
)

Write-Host "启动 InduForge 工业模拟设备，scenario=$Scenario"
foreach ($module in $modules) {
  $arguments = @($module.Script, "--host", "127.0.0.1", "--port", $module.Port, "--scenario", $Scenario)
  $process = Start-Process -FilePath $Python -ArgumentList $arguments -PassThru
  Write-Host ("{0} 已启动，PID={1}, port={2}" -f $module.Name, $process.Id, $module.Port)
}

Write-Host "停止服务请结束上述 PID，或关闭对应 Python 进程。"
