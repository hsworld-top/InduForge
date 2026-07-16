#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PACKAGE_DIR="${SCRIPT_DIR}/industrial_sim"
REQUIREMENTS_FILE="${SCRIPT_DIR}/requirements.txt"
DEFAULT_VENV_DIR="${REPO_ROOT}/.venv-industrial-sim"
DEFAULT_VENV_PYTHON="${DEFAULT_VENV_DIR}/bin/python"

PYTHON_USER_SET=0
if [[ -n "${PYTHON:-}" ]]; then
  PYTHON_USER_SET=1
else
  PYTHON=""
fi
SCENARIO="${SCENARIO:-normal}"
HOST="${HOST:-127.0.0.1}"
UPDATE_MS="${UPDATE_MS:-500}"
PORT=""
MODBUS_PORT="${MODBUS_PORT:-18502}"
OPCUA_PORT="${OPCUA_PORT:-18540}"
S7_PORT="${S7_PORT:-18503}"
VERBOSE=0
PROTOCOL=""
PIDS=()

usage() {
  cat <<'EOF'
Usage:
  bash scripts/test/industrial-sim/run.sh <modbus|opcua|s7|all> [options]

Options:
  --python PATH          Python 解释器；默认自动使用/创建仓库根目录 .venv-industrial-sim
  --scenario NAME        场景：normal/startup/alarm/noisy/intermittent，默认 normal
  --host HOST            监听地址，默认 127.0.0.1
  --port PORT            单协议端口；仅用于 modbus/opcua/s7
  --update-ms MS         设备刷新周期，默认 500
  --modbus-port PORT     all 模式下的 Modbus TCP 端口，默认 18502
  --opcua-port PORT      all 模式下的 OPC UA 端口，默认 18540
  --s7-port PORT         all 模式下的 S7 端口，默认 18503
  --verbose              输出调试日志
  -h, --help             显示帮助

Examples:
  bash scripts/test/industrial-sim/run.sh modbus --python ./.venv-industrial-sim/bin/python
  bash scripts/test/industrial-sim/run.sh opcua --scenario alarm
  bash scripts/test/industrial-sim/run.sh s7 --port 18503
  bash scripts/test/industrial-sim/run.sh all
EOF
}

require_value() {
  local name="$1"
  local value="${2:-}"
  if [[ -z "${value}" || "${value}" == --* ]]; then
    echo "参数 ${name} 缺少值" >&2
    exit 2
  fi
}

while (($# > 0)); do
  case "$1" in
    --python)
      require_value "$1" "${2:-}"
      PYTHON="$2"
      PYTHON_USER_SET=1
      shift 2
      ;;
    --scenario)
      require_value "$1" "${2:-}"
      SCENARIO="$2"
      shift 2
      ;;
    --host)
      require_value "$1" "${2:-}"
      HOST="$2"
      shift 2
      ;;
    --port)
      require_value "$1" "${2:-}"
      PORT="$2"
      shift 2
      ;;
    --update-ms)
      require_value "$1" "${2:-}"
      UPDATE_MS="$2"
      shift 2
      ;;
    --modbus-port)
      require_value "$1" "${2:-}"
      MODBUS_PORT="$2"
      shift 2
      ;;
    --opcua-port)
      require_value "$1" "${2:-}"
      OPCUA_PORT="$2"
      shift 2
      ;;
    --s7-port)
      require_value "$1" "${2:-}"
      S7_PORT="$2"
      shift 2
      ;;
    --verbose)
      VERBOSE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      echo "不支持的参数：$1" >&2
      usage >&2
      exit 2
      ;;
    *)
      if [[ -n "${PROTOCOL}" ]]; then
        echo "只能指定一个协议：modbus、opcua、s7 或 all" >&2
        exit 2
      fi
      PROTOCOL="$1"
      shift
      ;;
  esac
done

if [[ -z "${PROTOCOL}" ]]; then
  echo "缺少协议参数：modbus、opcua、s7 或 all" >&2
  usage >&2
  exit 2
fi

case "${PROTOCOL}" in
  modbus|opcua|s7|all) ;;
  *)
    echo "不支持的协议：${PROTOCOL}" >&2
    exit 2
    ;;
esac

case "${SCENARIO}" in
  normal|startup|alarm|noisy|intermittent) ;;
  *)
    echo "不支持的场景：${SCENARIO}" >&2
    exit 2
    ;;
esac

if [[ "${PROTOCOL}" == "all" && -n "${PORT}" ]]; then
  echo "all 模式不支持 --port，请使用 --modbus-port/--opcua-port/--s7-port。" >&2
  exit 2
fi

select_python() {
  if ((PYTHON_USER_SET)); then
    return
  fi

  if [[ -x "${DEFAULT_VENV_PYTHON}" ]]; then
    PYTHON="${DEFAULT_VENV_PYTHON}"
  else
    PYTHON="python3"
  fi
}

required_modules() {
  case "$1" in
    modbus) echo "pymodbus" ;;
    opcua) echo "asyncua" ;;
    s7) echo "snap7" ;;
    all)
      echo "pymodbus"
      echo "asyncua"
      echo "snap7"
      ;;
  esac
}

missing_modules() {
  "${PYTHON}" -c 'import importlib.util, sys; missing = [name for name in sys.argv[1:] if importlib.util.find_spec(name) is None]; print(",".join(missing)); sys.exit(1 if missing else 0)' "$@"
}

create_default_venv() {
  echo "未找到默认虚拟环境，正在创建：${DEFAULT_VENV_DIR}"
  # 默认只在未显式指定 Python 时托管依赖，避免污染系统 Python 或用户指定的解释器。
  if ! python3 -m venv "${DEFAULT_VENV_DIR}"; then
    echo "创建虚拟环境失败。WSL Ubuntu 可先执行：sudo apt install -y python3-venv" >&2
    exit 1
  fi
}

ensure_default_venv_pip() {
  if "${DEFAULT_VENV_PYTHON}" -m pip --version >/dev/null 2>&1; then
    return
  fi

  echo "默认虚拟环境缺少 pip，正在尝试修复：${DEFAULT_VENV_DIR}"
  # Ubuntu 的最小 Python 环境可能创建出不含 pip 的 venv；优先用标准 ensurepip 补齐。
  if ! "${DEFAULT_VENV_PYTHON}" -m ensurepip --upgrade >/dev/null 2>&1; then
    echo "默认虚拟环境缺少 pip，且 ensurepip 不可用。" >&2
    echo "WSL Ubuntu 可先执行：sudo apt install -y python3-venv python3-pip，然后删除 ${DEFAULT_VENV_DIR} 后重试。" >&2
    exit 1
  fi

  if ! "${DEFAULT_VENV_PYTHON}" -m pip --version >/dev/null 2>&1; then
    echo "默认虚拟环境 pip 修复后仍不可用，请删除 ${DEFAULT_VENV_DIR} 后重试。" >&2
    exit 1
  fi
}

install_requirements() {
  echo "正在安装工业协议模拟依赖：${REQUIREMENTS_FILE}"
  ensure_default_venv_pip
  if ! "${DEFAULT_VENV_PYTHON}" -m pip install -r "${REQUIREMENTS_FILE}"; then
    echo "依赖安装失败，请检查 WSL 网络或 pip 源，然后重试。" >&2
    exit 1
  fi
}

ensure_python_dependencies() {
  local modules=()
  while IFS= read -r module; do
    modules+=("${module}")
  done < <(required_modules "${PROTOCOL}")

  local missing=""
  if missing="$(missing_modules "${modules[@]}")"; then
    return
  fi

  if ((PYTHON_USER_SET)); then
    echo "Python 解释器缺少模块：${missing}" >&2
    echo "请执行：${PYTHON} -m pip install -r ${REQUIREMENTS_FILE}" >&2
    exit 1
  fi

  if [[ ! -x "${DEFAULT_VENV_PYTHON}" ]]; then
    create_default_venv
  fi

  PYTHON="${DEFAULT_VENV_PYTHON}"
  install_requirements

  if missing="$(missing_modules "${modules[@]}")"; then
    return
  fi

  echo "依赖安装后仍缺少模块：${missing}" >&2
  exit 1
}

select_python

if ! command -v "${PYTHON}" >/dev/null 2>&1; then
  echo "未找到 Python 解释器：${PYTHON}" >&2
  echo "请先创建虚拟环境，或通过 --python 指定可执行文件。" >&2
  exit 1
fi

ensure_python_dependencies

module_script() {
  case "$1" in
    modbus) echo "${PACKAGE_DIR}/modbus_sim.py" ;;
    opcua) echo "${PACKAGE_DIR}/opcua_sim.py" ;;
    s7) echo "${PACKAGE_DIR}/s7_sim.py" ;;
  esac
}

module_port() {
  case "$1" in
    modbus) echo "${PORT:-${MODBUS_PORT}}" ;;
    opcua) echo "${PORT:-${OPCUA_PORT}}" ;;
    s7) echo "${PORT:-${S7_PORT}}" ;;
  esac
}

module_name() {
  case "$1" in
    modbus) echo "Modbus TCP" ;;
    opcua) echo "OPC UA" ;;
    s7) echo "S7" ;;
  esac
}

build_args() {
  local protocol="$1"
  local args=(
    "$(module_script "${protocol}")"
    --host "${HOST}"
    --port "$(module_port "${protocol}")"
    --scenario "${SCENARIO}"
    --update-ms "${UPDATE_MS}"
  )

  if ((VERBOSE)); then
    args+=(--verbose)
  fi

  printf '%s\0' "${args[@]}"
}

run_single() {
  local protocol="$1"
  local args=()
  while IFS= read -r -d '' item; do
    args+=("${item}")
  done < <(build_args "${protocol}")

  echo "启动 $(module_name "${protocol}") 模拟设备，scenario=${SCENARIO}, port=$(module_port "${protocol}")"
  # 单协议以前台运行，输入输出和 Ctrl+C 都直接交给 Python 服务进程。
  exec "${PYTHON}" "${args[@]}"
}

cleanup() {
  local status=$?
  trap - EXIT INT TERM

  if ((${#PIDS[@]} > 0)); then
    echo
    echo "正在停止工业协议模拟设备..."
    for pid in "${PIDS[@]}"; do
      if kill -0 "${pid}" >/dev/null 2>&1; then
        kill "${pid}" >/dev/null 2>&1 || true
      fi
    done
    wait "${PIDS[@]}" 2>/dev/null || true
  fi

  exit "${status}"
}

start_background() {
  local protocol="$1"
  local args=()
  while IFS= read -r -d '' item; do
    args+=("${item}")
  done < <(build_args "${protocol}")

  # all 模式后台启动三个协议服务，并记录 PID，方便异常退出或 Ctrl+C 时统一回收。
  "${PYTHON}" "${args[@]}" &
  local pid=$!
  PIDS+=("${pid}")
  echo "$(module_name "${protocol}") 已启动，PID=${pid}, port=$(module_port "${protocol}")"
}

if [[ "${PROTOCOL}" != "all" ]]; then
  run_single "${PROTOCOL}"
fi

trap cleanup EXIT INT TERM

echo "启动 InduForge 工业模拟设备，scenario=${SCENARIO}"
start_background modbus
start_background opcua
start_background s7
echo "按 Ctrl+C 停止全部服务。"

# 任一子进程退出都表示模拟环境不完整，立即触发清理，避免后台残留进程。
wait -n
