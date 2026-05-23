#!/bin/sh
set -e

if [ -n "${IF_CACHE_STORE_PASSWORD:-}" ]; then
  exec redis-server --appendonly yes --requirepass "$IF_CACHE_STORE_PASSWORD"
fi

exec redis-server --appendonly yes
