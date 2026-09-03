#!/bin/sh
set -eu

# 设计器由 Wujie 加载到 shadow DOM。若带 hash 的 CSS/JS 不存在却被 SPA 回退为
# index.html，浏览器仍会得到 200，随后把 HTML 当作样式或脚本，最终白屏。
# 本测试以实际 edge 镜像启动 Nginx，锁定设计器入口、资产 MIME 和缺失资产的 404 语义。
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../../.." && pwd)
NGINX_CONFIG="$REPO_ROOT/scripts/k3s/center/nginx.conf"
EDGE_IMAGE=${IF_EDGE_IMAGE:-induforge/edge:nginx-assets-test}
TEMP_DIR=$(mktemp -d)
CONTAINER_NAME="induforge-edge-assets-test-$$"

cleanup() {
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT INT TERM

if [ -z "${IF_EDGE_IMAGE+x}" ]; then
  docker build -t "$EDGE_IMAGE" -f "$SCRIPT_DIR/Dockerfile" "$REPO_ROOT"
fi

mkdir -p "$TEMP_DIR/tls"
openssl req -x509 -nodes -newkey rsa:2048 -days 1 -subj '/CN=localhost' \
  -keyout "$TEMP_DIR/tls/center.key" -out "$TEMP_DIR/tls/center.crt" >/dev/null 2>&1

docker run -d --rm --name "$CONTAINER_NAME" \
  -p 127.0.0.1::80 \
  -v "$NGINX_CONFIG:/etc/nginx/nginx.conf:ro" \
  -v "$TEMP_DIR/tls:/etc/nginx/tls:ro" \
  --entrypoint nginx \
  "$EDGE_IMAGE" -c /etc/nginx/nginx.conf -g 'daemon off;' >/dev/null

port=$(docker port "$CONTAINER_NAME" 80/tcp | sed -n '1s/.*://p')
base_url="http://127.0.0.1:$port"

attempt=0
until curl -fsS "$base_url/designer/" -o "$TEMP_DIR/designer.html" 2>/dev/null; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 20 ]; then
    echo "edge 设计器入口未就绪" >&2
    exit 1
  fi
  sleep 1
done

grep -Fq 'id="app"' "$TEMP_DIR/designer.html" || {
  echo "设计器入口未提供 Wujie 可见挂载根节点" >&2
  exit 1
}

assets=$(grep -Eo '/designer/assets/[0-9A-Za-z_.-]+\.(css|js)' "$TEMP_DIR/designer.html" | sort -u)
[ -n "$assets" ] || {
  echo "设计器入口未引用 CSS/JS 资产" >&2
  exit 1
}

printf '%s\n' "$assets" | while IFS= read -r asset; do
  headers="$TEMP_DIR/headers"
  body="$TEMP_DIR/body"
  curl -fsS -D "$headers" -o "$body" "$base_url$asset"
  case "$asset" in
    *.css) expected_type='text/css' ;;
    *.js) expected_type='application/javascript' ;;
    *) echo "未识别的设计器资产类型: $asset" >&2; exit 1 ;;
  esac
  grep -Eqi "^Content-Type: $expected_type" "$headers" || {
    echo "设计器资产 Content-Type 不正确: $asset" >&2
    exit 1
  }
  if grep -Eqi '<!doctype html|<html' "$body"; then
    echo "设计器资产错误返回了 HTML: $asset" >&2
    exit 1
  fi
done

# 旧页面缓存的 hash 在新镜像中不存在时，必须明确 404，不能被 /designer/ 的 SPA 回退吞掉。
status=$(curl -sS -D "$TEMP_DIR/stale.headers" -o "$TEMP_DIR/stale.body" -w '%{http_code}' \
  "$base_url/designer/assets/stale-CvEIMyRj.css")
[ "$status" = '404' ] || {
  echo "缺失的设计器 CSS 未返回 404，实际为 $status" >&2
  exit 1
}

echo "edge 设计器资产 HTTP/Wujie 挂载测试通过"
