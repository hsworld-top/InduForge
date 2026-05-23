param()

# InduForge Windows 离线安装入口。
#
# 输入：
# - 当前目录下的 `.env` 或 `.env.production.example`。
# - scripts\docker\images\*.tar 中的离线镜像。
# - 已安装并可用的 Docker Desktop 与 Docker Compose。
#
# 输出：
# - 导入离线镜像。
# - 使用 Docker Compose 启动 InduForge 生产拓扑。
# - 通过容器内命令执行基础设施初始化。
#
# 重要边界：
# - 本脚本只检测 Docker 环境，不安装 Docker。
# - 若目标机器没有 Docker 或 Compose，会直接退出并提示用户先安装。
# - 初始化只依赖 Docker，不要求目标机器额外安装 Node、pnpm 或 Go。

$ErrorActionPreference = "Stop"

$packageRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$envFile = Join-Path $packageRoot ".env"
$composeFile = Join-Path $packageRoot "scripts/docker/docker-compose.offline.yml"
$imageDir = Join-Path $packageRoot "scripts/docker/images"
$metaContainer = "induforge-meta-store"
$controlContainer = "induforge-control"

function Test-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "缺少命令: $Name。请先在目标机器安装 Docker 环境。"
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

  throw "缺少 Docker Compose。请先安装 Docker Compose v2 插件或 docker-compose。"
}

function Initialize-EnvFile {
  if (Test-Path $envFile) {
    return
  }

  $example = Join-Path $packageRoot ".env.production.example"
  if (-not (Test-Path $example)) {
    throw "缺少 .env.production.example，无法生成 .env。"
  }

  Copy-Item $example $envFile
  throw "已生成 .env，请按生产环境要求修改 change_me_* 密钥后重新执行安装。"
}

function Import-OfflineImages {
  $images = Get-ChildItem -Path $imageDir -Filter "*.tar" -File -ErrorAction SilentlyContinue
  if (-not $images -or $images.Count -eq 0) {
    throw "未找到离线镜像: $imageDir\*.tar"
  }

  foreach ($image in $images) {
    Write-Host "导入镜像: $($image.FullName)"
    docker load -i $image.FullName
  }
}

function Get-EnvValue {
  param(
    [string]$Name,
    [string]$Fallback
  )

  $line = Get-Content $envFile | Where-Object { $_ -match "^$Name=" } | Select-Object -Last 1
  if ($line) {
    return ($line -split "=", 2)[1]
  }

  return $Fallback
}

function ConvertTo-SqlIdentifier {
  param([string]$Value)

  if ($Value -notmatch "^[a-zA-Z_][a-zA-Z0-9_]*$") {
    throw "非法数据库标识符: $Value"
  }

  return '"' + ($Value -replace '"', '""') + '"'
}

function Start-Services {
  $compose = Get-ComposeCommand
  if ($compose.Count -eq 2) {
    docker compose --env-file $envFile -f $composeFile up -d
  } else {
    docker-compose --env-file $envFile -f $composeFile up -d
  }
}

function Wait-MetaStore {
  $user = Get-EnvValue "IF_META_STORE_USER" "postgres"
  $password = Get-EnvValue "IF_META_STORE_PASSWORD" "postgres"
  $adminDb = Get-EnvValue "IF_META_STORE_ADMIN_DATABASE" "postgres"
  $port = Get-EnvValue "IF_META_STORE_PORT" "18432"

  Write-Host "等待元数据能力就绪..."
  for ($i = 0; $i -lt 60; $i++) {
    docker exec -e "PGPASSWORD=$password" $metaContainer psql -p $port -U $user -d $adminDb -tAc "SELECT 1" *> $null
    if ($LASTEXITCODE -eq 0) {
      return
    }
    Start-Sleep -Seconds 2
  }

  throw "元数据能力启动超时，请检查容器日志: docker logs $metaContainer"
}

function New-DatabaseIfNeeded {
  param([string]$Database)

  $user = Get-EnvValue "IF_META_STORE_USER" "postgres"
  $password = Get-EnvValue "IF_META_STORE_PASSWORD" "postgres"
  $adminDb = Get-EnvValue "IF_META_STORE_ADMIN_DATABASE" "postgres"
  $port = Get-EnvValue "IF_META_STORE_PORT" "18432"
  $quoted = ConvertTo-SqlIdentifier $Database

  $exists = docker exec -e "PGPASSWORD=$password" $metaContainer psql -p $port -U $user -d $adminDb -tAc "SELECT 1 FROM pg_database WHERE datname = '$Database';"
  if (($exists -join "").Trim() -eq "1") {
    Write-Host "数据库已存在: $Database"
    return
  }

  docker exec -e "PGPASSWORD=$password" $metaContainer psql -p $port -U $user -d $adminDb -c "CREATE DATABASE $quoted;"
  Write-Host "数据库已创建: $Database"
}

function Enable-TimeSeriesExtension {
  param([string]$Database)

  $user = Get-EnvValue "IF_META_STORE_USER" "postgres"
  $password = Get-EnvValue "IF_META_STORE_PASSWORD" "postgres"
  $port = Get-EnvValue "IF_META_STORE_PORT" "18432"
  $extensionName = "time" + "scaledb"

  docker exec -e "PGPASSWORD=$password" $metaContainer psql -p $port -U $user -d $Database -c "CREATE EXTENSION IF NOT EXISTS $extensionName;"
  Write-Host "时序扩展已启用: $Database"
}

function Initialize-ControlSchema {
  # 控制面镜像内已经包含 dev_core 的数据库初始化脚本和生产依赖。
  # 这里在数据库可用后显式执行一次，确保正式安装后核心表和默认管理员数据存在。
  Write-Host "初始化控制面数据库结构..."
  docker exec $controlContainer node scripts/bootstrap/init-core-database.js init
}

function Invoke-InfraInit {
  Wait-MetaStore

  $coreDb = Get-EnvValue "IF_META_STORE_CORE_DB" "if_core"
  $dataDb = Get-EnvValue "IF_META_STORE_DATA_DB" "if_data"
  $devDataDb = Get-EnvValue "IF_META_STORE_DEV_DATA_DB" "if_dev_data"

  New-DatabaseIfNeeded $coreDb
  New-DatabaseIfNeeded $dataDb
  New-DatabaseIfNeeded $devDataDb
  Enable-TimeSeriesExtension $devDataDb
  Initialize-ControlSchema
}

Test-Command docker
docker info *> $null
if ($LASTEXITCODE -ne 0) {
  throw "Docker 当前不可用。请确认 Docker Desktop/Engine 已启动，并且当前用户有权限访问 Docker。"
}

Initialize-EnvFile
Import-OfflineImages
Start-Services
Invoke-InfraInit

$port = Get-EnvValue "IF_EDGE_HOST_PORT" "18080"
Write-Host "InduForge 离线安装流程已执行完成。"
Write-Host "访问入口: http://localhost:$port"
