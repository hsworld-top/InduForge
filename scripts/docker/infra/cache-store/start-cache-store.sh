#!/bin/sh
set -e

if [ -n "${IF_CACHE_STORE_PASSWORD:-}" ]; then
  exec redis-server --port "${IF_CACHE_STORE_PORT:-18379}" --appendonly yes --requirepass "$IF_CACHE_STORE_PASSWORD"
fi

exec redis-server --port "${IF_CACHE_STORE_PORT:-18379}" --appendonly yes
