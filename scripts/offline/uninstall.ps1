param(
  # 默认只删除容器和网络；传入该参数时才删除数据卷。
  [switch]$RemoveVolumes
)

# InduForge Windows 离线卸载入口。
#
# 输入：
# - 当前目录下的 `.env` 和 scripts\docker\docker-compose.offline.yml。
# - 可选参数 `-RemoveVolumes`，表示同时删除 Docker 数据卷。
#
# 输出：
# - 默认停止并删除 InduForge 容器和网络，保留数据卷。
# - 传入 `-RemoveVolumes` 时额外删除数据库、缓存、消息和对象存储数据卷。
#
# 安全边界：
# - 本脚本不删除 Docker 镜像 tar，不删除安装包目录。
# - 默认不删除数据卷，避免误删正式业务数据。

$ErrorActionPreference = "Stop"

$packageRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$envFile = Join-Path $packageRoot ".env"
$composeFile = Join-Path $packageRoot "scripts/docker/docker-compose.offline.yml"

function Test-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "缺少命令: $Name。请确认 Docker 环境仍可用后再卸载。"
  }
}

function Get-ComposeCommand {
  docker compose version *> $null
  if ($LASTEXITCODE -eq 0) {
    return @("docker", "compose")
  }

  if (Get-Command docker-compose -ErrorAction SilentlyContinue) {
    return @("docker-compose")
  }

  throw "缺少 Docker Compose。请先恢复 Docker Compose 后再卸载。"
}

Test-Command docker
docker info *> $null
if ($LASTEXITCODE -ne 0) {
  throw "Docker 当前不可用。请确认 Docker Desktop/Engine 已启动，并且当前用户有权限访问 Docker。"
}

$hasEnvFile = Test-Path $envFile
if (-not $hasEnvFile) {
  Write-Host "缺少 .env，卸载将使用 compose 默认值继续执行。"
}

$compose = Get-ComposeCommand
$downArgs = @()
if ($hasEnvFile) {
  $downArgs += @("--env-file", $envFile)
}
$downArgs += @("-f", $composeFile, "down", "--remove-orphans")

if ($RemoveVolumes) {
  Write-Host "即将删除 InduForge 容器、网络和数据卷。"
  $downArgs += "--volumes"
} else {
  Write-Host "即将删除 InduForge 容器和网络，数据卷会保留。"
}

if ($compose.Count -eq 2) {
  docker compose @downArgs
} else {
  docker-compose @downArgs
}

Write-Host "InduForge 卸载流程已完成。"
