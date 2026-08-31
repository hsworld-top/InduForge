#!/usr/bin/env bash
# macOS/Linux CI 无法运行 PowerShell；至少确保 Windows 安装器未遗漏 collector 和版本模板。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$SCRIPT_DIR/install-windows.ps1"
grep -Fq 'capabilities\collector\win-x64\industrial_collector.exe' "$SCRIPT"
grep -Fq 'Copy-Item "$PSScriptRoot\capabilities\collector\win-x64\industrial_collector.exe"' "$SCRIPT"
grep -Fq 'BUILD_VERSION' "$SCRIPT"
grep -Fq 'releases' "$SCRIPT"
echo 'install-windows static test passed'
