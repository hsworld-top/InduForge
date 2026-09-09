#!/bin/sh
# 中心内置节点只开放镜像导入能力，不开放外部节点的集群安装/卸载入口。
set -eu
[ "$(id -u)" -eq 0 ] || { echo "请使用 sudo 安装中心镜像服务" >&2; exit 1; }
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
BINARY="${IF_CENTER_HOSTD_BINARY:-$SCRIPT_DIR/bin/node-hostd}"
[ -x "$BINARY" ] || { echo "中心安装包缺少 bin/node-hostd" >&2; exit 1; }
GROUP=$(getent group 1000 | cut -d: -f1)
[ -n "$GROUP" ] || { echo "中心服务组 1000 不存在" >&2; exit 1; }
install -d -m 0755 /opt/induforge/center-k3s/bin
if [ "$BINARY" != /opt/induforge/center-k3s/bin/node-hostd ]; then
  install -m 0755 "$BINARY" /opt/induforge/center-k3s/bin/node-hostd
fi
install -d -o 1000 -g 1000 -m 0700 /var/lib/induforge/image-cache
install -d -o root -g 1000 -m 0750 /run/induforge-center
install -d -o root -g root -m 0700 /var/lib/induforge/center-image-service
cat > /etc/systemd/system/induforge-center-images.service <<EOF
[Unit]
Description=InduFrame center local image service
After=network-online.target
[Service]
User=root
Group=root
Environment=INDUFORGE_HOSTD_IMAGES_ONLY=true
Environment=INDUFORGE_HOSTD_GROUP=$GROUP
Environment=INDUFORGE_HOSTD_SOCKET=/run/induforge-center/hostd.sock
Environment=INDUFORGE_HOSTD_STATE_DIR=/var/lib/induforge/center-image-service
ExecStart=/opt/induforge/center-k3s/bin/node-hostd
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now induforge-center-images.service
