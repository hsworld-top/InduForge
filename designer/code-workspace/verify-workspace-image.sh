#!/bin/sh
set -eu

image="${1:-induforge/designer-code-server:workspace-templates-source}"
verification_script=$(mktemp)
trap 'rm -f "$verification_script"' EXIT HUP INT TERM

# 不要在循环内直接使用 here-document。`docker run -i` 会消费调用方的标准输入，
# 某些 shell 下第二个模板起将得到 EOF，脚本却可能只验证第一个模板。每次从临时脚本
# 重新定向输入，确保四种模板都实际执行离线初始化与构建。
cat >"$verification_script" <<'CONTAINER'
set -eu

for generated_path in node_modules dist .pnpm-store; do
  test ! -e "/opt/induforge/templates/$TEMPLATE_ID/$generated_path"
done

mkdir -p /cache/pnpm-store
cp -a /opt/induforge/pnpm-store/. /cache/pnpm-store/
chmod -R u+rwX /cache/pnpm-store
node --input-type=module -e "
import { createWorkspaceInitializer } from '/opt/induforge/workspace-initializer.mjs'
const initializer = createWorkspaceInitializer({ storeDir: '/cache/pnpm-store' })
console.log(await initializer.initialize(process.env.TEMPLATE_ID))
"

test "$(git branch --show-current)" = main
test "$(git remote)" = ""
test "$(git log -1 --pretty=%s)" = "chore: 初始化工程模板"
test "$(pnpm config get store-dir)" = /cache/pnpm-store
test "$(pnpm config get trust-lockfile)" = true
if ! pnpm build >/tmp/template-build.log 2>&1; then
  cat /tmp/template-build.log >&2
  exit 1
fi
test -f dist/index.html

if [ "$TEMPLATE_ID" = vite-vue-js ]; then
  node /opt/induforge/vite-runner.mjs >/tmp/vite.log 2>&1 &
  vite_pid=$!
  node -e "
const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
for (let index = 0; index < 60; index += 1) {
  try {
    const response = await fetch('http://127.0.0.1:5173')
    if (response.ok) {
      const html = await response.text()
      process.exit(html.includes('id=\"root\"') || html.includes('id=\"app\"') ? 0 : 2)
    }
  } catch {}
  await wait(250)
}
process.exit(3)
"
  kill "$vite_pid"
  wait "$vite_pid" 2>/dev/null || true
fi

echo "$TEMPLATE_ID ok"
CONTAINER

for template in vite-vue-js vite-vue-ts vite-react-js vite-react-ts; do
  echo "=== ${template} ==="
  docker run --rm --interactive --network none --entrypoint sh -e TEMPLATE_ID="$template" "$image" -s <"$verification_script"
done
