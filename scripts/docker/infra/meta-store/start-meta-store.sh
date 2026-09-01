#!/bin/sh
set -eu

# 重新进入上游 entrypoint，保留首次初始化、权限收敛和降权逻辑，同时允许
# 离线安装配置决定容器内部监听端口。
exec docker-entrypoint.sh postgres \
  -p "${IF_META_STORE_PORT:-18432}" \
  -c shared_preload_libraries=timescaledb
