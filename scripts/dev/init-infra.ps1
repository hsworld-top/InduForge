param(
  # 只检查环境是否连通，不创建数据库、不启用扩展。
  # 适合新机器复制 .env 后先确认容器和端口是否可用。
  [switch]$CheckOnly
)

# 遇到命令失败立即停止，避免初始化只完成一半但脚本仍显示成功。
$ErrorActionPreference = "Stop"

# PowerShell 脚本只做跨平台入口封装，真正的初始化逻辑放在同目录 Node 脚本中。
# 这样 Windows 用户可以直接执行 ps1，Linux/macOS 用户可以直接执行 mjs。
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "../..")
$nodeScript = Join-Path $PSScriptRoot "init-infra.mjs"

# 明确检查脚本文件是否存在，避免 node 报出不直观的模块加载错误。
if (-not (Test-Path $nodeScript)) {
  throw "缺少初始化脚本: $nodeScript"
}

# 将 PowerShell 参数转换成 Node 脚本参数。
# 后续如需增加初始化选项，只需要在这里追加参数映射。
$args = @($nodeScript)
if ($CheckOnly) {
  $args += "--check-only"
}

# 固定从仓库根目录运行，确保 Node 脚本读取的是根目录 `.env`。
# finally 中恢复调用者原始目录，避免影响用户后续命令。
Push-Location $repoRoot
try {
  node @args
} finally {
  Pop-Location
}
