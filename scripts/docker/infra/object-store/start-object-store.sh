#!/bin/sh
set -e

cat > /tmp/induforge-s3.json <<EOF
{"identities":[{"name":"induforge-admin","credentials":[{"accessKey":"${IF_OBJECT_STORE_ACCESS_KEY}","secretKey":"${IF_OBJECT_STORE_SECRET_KEY}"}],"actions":["Admin","Read","Write","List","Tagging"]}]}
EOF

exec weed server \
  -ip=object-store \
  -ip.bind=0.0.0.0 \
  -master.port=${IF_OBJECT_STORE_MASTER_PORT:-18333} \
  -master.volumeSizeLimitMB=1024 \
  -volume.port=${IF_OBJECT_STORE_VOLUME_PORT:-18081} \
  -dir=/data/volume \
  -filer \
  -filer.port=${IF_OBJECT_STORE_FILER_PORT:-18888} \
  -s3 \
  -s3.port=${IF_OBJECT_STORE_PORT:-18500} \
  -s3.config=/tmp/induforge-s3.json
