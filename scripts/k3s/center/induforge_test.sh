#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT INT TERM

cat > "$temp_dir/manager" <<'EOF'
#!/bin/sh
case "$1" in
  summary) printf 'control|healthy\nedge|healthy\ndatabase|healthy\ncache|healthy\nobject-store|healthy\n' ;;
  doctor) echo healthy ;;
  logs) echo log-line ;;
  restart) exit 0 ;;
  *) exit 2 ;;
esac
EOF
chmod 0755 "$temp_dir/manager"

NO_COLOR=1 INDUFORGE_CENTER_MANAGER="$temp_dir/manager" "$SCRIPT_DIR/induforge" status > "$temp_dir/status"
grep -Fq '中心开发系统' "$temp_dir/status"
grep -Fq 'IDE 访问入口' "$temp_dir/status"
if grep -Fq '物理节点' "$temp_dir/status" && ! grep -Fq '请在 Web 运维管理中查看' "$temp_dir/status"; then
  echo "status exposed node details" >&2
  exit 1
fi
if LC_ALL=C grep -q "$(printf '\033')" "$temp_dir/status"; then
  echo "NO_COLOR output contains ANSI escapes" >&2
  exit 1
fi

json=$(NO_COLOR=1 INDUFORGE_CENTER_MANAGER="$temp_dir/manager" "$SCRIPT_DIR/induforge" status --json)
case "$json" in
  '{"status":"healthy","services":['*'"name":"control"'*'"name":"object-store"'*) ;;
  *) echo "unexpected status JSON: $json" >&2; exit 1 ;;
esac

NO_COLOR=1 INDUFORGE_CENTER_MANAGER="$temp_dir/manager" "$SCRIPT_DIR/induforge" doctor | grep -Fq '诊断通过'
NO_COLOR=1 INDUFORGE_CENTER_MANAGER="$temp_dir/manager" "$SCRIPT_DIR/induforge" logs control | grep -Fq 'log-line'
NO_COLOR=1 INDUFORGE_CENTER_MANAGER="$temp_dir/manager" "$SCRIPT_DIR/induforge" restart control | grep -Fq '重启请求已提交'

echo "induforge CLI tests passed"
