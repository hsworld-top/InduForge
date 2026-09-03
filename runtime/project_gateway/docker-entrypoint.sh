#!/bin/sh
set -eu

: "${IF_RELEASE_ROOT:?IF_RELEASE_ROOT is required}"
: "${IF_WORK_ROOT:?IF_WORK_ROOT is required}"
: "${IF_DEPLOYMENT_ID:?IF_DEPLOYMENT_ID is required}"
: "${IF_PROJECT_ID:?IF_PROJECT_ID is required}"
: "${IF_ENVIRONMENT_ID:?IF_ENVIRONMENT_ID is required}"
: "${IF_NODE_ID:?IF_NODE_ID is required}"
: "${IF_RELEASE_ID:?IF_RELEASE_ID is required}"

exec /usr/local/bin/project-gateway \
  --listen 0.0.0.0:18080 \
  --release-root "$IF_RELEASE_ROOT" \
  --client-root "$IF_WORK_ROOT/client" \
  --runtime-api http://127.0.0.1:18081 \
  --viewer-token-file /var/run/induforge/runtime-viewer/token \
  --deployment-id "$IF_DEPLOYMENT_ID" \
  --account-id "$IF_PROJECT_ID" \
  --project-id "$IF_PROJECT_ID" \
  --site-id "$IF_ENVIRONMENT_ID" \
  --node-id "$IF_NODE_ID" \
  --version "$IF_RELEASE_ID" \
  --execution-form native-linux
