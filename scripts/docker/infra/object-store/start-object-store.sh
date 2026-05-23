#!/bin/sh
set -e

cat > /tmp/induforge-s3.json <<EOF
{"identities":[{"name":"induforge-admin","credentials":[{"accessKey":"${IF_OBJECT_STORE_ACCESS_KEY}","secretKey":"${IF_OBJECT_STORE_SECRET_KEY}"}],"actions":["Admin","Read","Write","List","Tagging"]}]}
EOF

exec weed server \
  -ip=object-store \
  -ip.bind=0.0.0.0 \
  -master.port=9333 \
  -master.volumeSizeLimitMB=1024 \
  -volume.port=8080 \
  -dir=/data/volume \
  -filer \
  -filer.port=8888 \
  -s3 \
  -s3.port=8333 \
  -s3.config=/tmp/induforge-s3.json
