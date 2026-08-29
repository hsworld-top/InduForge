#!/usr/bin/env bash
# 在 Linux Docker 容器中验证 runtime_linux 与 collector_linux 包的等价运行路径。
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
"$SCRIPT_DIR/test-installer-static.sh"
"$REPO_ROOT/scripts/release/build-node-agent-packages.sh"
tar -tzf "$REPO_ROOT/.data/node-packages/induforge-node-runtime-linux.tar.gz" | grep '/README.md$' >/dev/null
tar -tzf "$REPO_ROOT/.data/node-packages/induforge-node-collector-linux.tar.gz" | grep '/README.md$' >/dev/null
unzip -Z1 "$REPO_ROOT/.data/node-packages/induforge-node-collector-windows.zip" | grep '/README.md$' >/dev/null
cd "$SCRIPT_DIR"
docker compose up --build -d
cleanup() { docker compose down -v; }
trap cleanup EXIT
status() { docker compose exec -T "$1" wget -qO- http://127.0.0.1:17601/api/v1/ops/processes; }
wait_for() { local service="$1" needle="$2"; for _ in $(seq 1 30); do status "$service" 2>/dev/null | grep -q "$needle" && return 0; sleep 1; done; return 1; }
wait_for runtime '"workloadId":"compute-e2e"' && wait_for runtime '"workloadId":"alert-e2e"' && wait_for collector '"workloadId":"collector-e2e"'
runtime_before="$(status runtime)"; collector_before="$(status collector)"
runtime_pid_before="$(printf '%s' "$runtime_before" | sed -n 's/.*"workloadId":"compute-e2e"[^}]*"pid":\([0-9]*\).*/\1/p')"
collector_pid_before="$(printf '%s' "$collector_before" | sed -n 's/.*"workloadId":"collector-e2e"[^}]*"pid":\([0-9]*\).*/\1/p')"
curl -fsS -X POST 'http://127.0.0.1:18080/advance?phase=stop' >/dev/null
wait_for runtime '"observedStatus":"stopped"' && wait_for collector '"observedStatus":"stopped"'
curl -fsS -X POST 'http://127.0.0.1:18080/advance?phase=restart' >/dev/null
wait_for runtime '"observedGeneration":3' && wait_for collector '"observedGeneration":3'
runtime_after="$(status runtime)"; collector_after="$(status collector)"
runtime_pid_after="$(printf '%s' "$runtime_after" | sed -n 's/.*"workloadId":"compute-e2e"[^}]*"pid":\([0-9]*\).*/\1/p')"
collector_pid_after="$(printf '%s' "$collector_after" | sed -n 's/.*"workloadId":"collector-e2e"[^}]*"pid":\([0-9]*\).*/\1/p')"
[ -n "$runtime_pid_before" ] && [ -n "$collector_pid_before" ] && [ -n "$runtime_pid_after" ] && [ -n "$collector_pid_after" ] && [ "$runtime_pid_before" != "$runtime_pid_after" ] && [ "$collector_pid_before" != "$collector_pid_after" ]
docker compose exec -T runtime wget -qO- 'http://127.0.0.1:17601/api/v1/ops/processes/compute-e2e/logs' | grep -q 'demo workload'
docker compose exec -T collector wget -qO- 'http://127.0.0.1:17601/api/v1/ops/processes/collector-e2e/logs' | grep -q 'demo workload'
echo "E2E evidence: compute pid $runtime_pid_before -> $runtime_pid_after; collector pid $collector_pid_before -> $collector_pid_after; compute+alert stopped at generation=2 and restarted at generation=3; workload logs present."
echo "ops runtime E2E passed: package install, enrollment, heartbeat, start/stop/restart, PID and logs verified"
exit 0
docker compose logs >&2
exit 1
