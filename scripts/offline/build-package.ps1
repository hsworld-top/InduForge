param(
  # 离线包目录名，默认输出到 dist/induforge-offline-package。
  [string]$PackageName = "induforge-offline-package"
)

# InduForge Windows 离线交付包构建脚本。
#
# 输入：
# - 本机 Docker 中已经存在的正式镜像。
# - 仓库内的生产环境模板、离线 compose 和安装脚本。
#
# 输出：
# - dist\<PackageName>\ 目录。
# - dist\<PackageName>.zip 压缩包。
#
# 本脚本只检测 Docker 是否可用并导出镜像，不负责安装 Docker，也不负责构建业务镜像。

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptDir "../..")
$distDir = Join-Path $repoRoot "dist"
$packageDir = Join-Path $distDir $PackageName
$imageDir = Join-Path $packageDir "scripts/docker/images"

$requiredImages = @(
  "induforge/edge:latest",
  "induforge/control:latest",
  "induforge/designer-code-server:4.131.0-node24.19.0-pnpm11.21.0-8147b161-arm64",
  "induforge/data:latest",
  "induforge/meta-store:latest",
  "induforge/cache-store:latest",
  "induforge/message-hub:latest",
  "induforge/object-store:latest"
)

$productImageSources = @(
  "meta-store",
  "cache-store",
  "message-hub",
  "object-store"
)

function Test-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "缺少命令: $Name"
  }
}

function ConvertTo-ImageTarName {
  param([string]$Image)
  # Docker 镜像名包含 `/` 和 `:`，这里替换成 Windows/Linux 都安全的文件名。
  return (($Image -replace "[/:]", "_") + ".tar")
}

function Test-RequiredImages {
  $missing = @()
  foreach ($image in $requiredImages) {
    docker image inspect $image *> $null
    if ($LASTEXITCODE -ne 0) {
      $missing += $image
    }
  }

  if ($missing.Count -gt 0) {
    $message = "缺少以下 Docker 镜像，无法生成正式离线包:`n" + (($missing | ForEach-Object { "  - $_" }) -join "`n")
    throw "$message`n请先在构建机完成镜像构建或拉取，再重新执行本脚本。"
  }
}

function Ensure-ProductImageTags {
  # 正式交付包只保存 InduForge 产品体系镜像名。
  # 产品镜像通过轻量 wrapper 固化启动命令，避免离线 compose 暴露底层产品命令。
  foreach ($product in $productImageSources) {
    Write-Host "构建产品体系镜像: induforge/$product`:latest"
    docker build `
      -t "induforge/$product`:latest" `
      -f (Join-Path $repoRoot "scripts/docker/infra/$product/Dockerfile") `
      (Join-Path $repoRoot "scripts/docker/infra/$product")
  }
}

function Copy-DeployFiles {
  Copy-Item (Join-Path $repoRoot ".env.production.example") (Join-Path $packageDir ".env.production.example")
  Copy-Item (Join-Path $repoRoot "scripts/docker/docker-compose.offline.yml") (Join-Path $packageDir "scripts/docker/docker-compose.offline.yml")
  Copy-Item (Join-Path $repoRoot "scripts/offline/install.sh") (Join-Path $packageDir "install.sh")
  Copy-Item (Join-Path $repoRoot "scripts/offline/install.ps1") (Join-Path $packageDir "install.ps1")
  Copy-Item (Join-Path $repoRoot "scripts/offline/uninstall.sh") (Join-Path $packageDir "uninstall.sh")
  Copy-Item (Join-Path $repoRoot "scripts/offline/uninstall.ps1") (Join-Path $packageDir "uninstall.ps1")
  Copy-Item (Join-Path $repoRoot "scripts/offline/README.md") (Join-Path $packageDir "README.md")
}

function Save-Images {
  foreach ($image in $requiredImages) {
    $tarFile = Join-Path $imageDir (ConvertTo-ImageTarName $image)
    Write-Host "保存镜像: $image -> $tarFile"
    docker save -o $tarFile $image
  }
}

Test-Command docker

Ensure-ProductImageTags
Test-RequiredImages

if (Test-Path $packageDir) {
  Remove-Item -Recurse -Force $packageDir
}

New-Item -ItemType Directory -Force $imageDir | Out-Null
New-Item -ItemType Directory -Force (Join-Path $packageDir "scripts/docker") | Out-Null
New-Item -ItemType Directory -Force (Join-Path $packageDir "scripts/offline") | Out-Null
New-Item -ItemType Directory -Force (Join-Path $packageDir "scripts") | Out-Null

Copy-DeployFiles
Save-Images

$zipPath = Join-Path $distDir "$PackageName.zip"
if (Test-Path $zipPath) {
  Remove-Item -Force $zipPath
}

Compress-Archive -Path $packageDir -DestinationPath $zipPath
Write-Host "离线交付包已生成: $zipPath"
