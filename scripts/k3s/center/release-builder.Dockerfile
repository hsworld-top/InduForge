# 正式发布只需要 Node、Corepack 与经审核的离线 pnpm store；不能复用包含 IDE、
# code-server 和 AI 工作台的开发镜像，以控制中心节点的磁盘与攻击面。
FROM induforge/designer-code-server:workspace-templates-source AS workspace-assets

FROM node:24.19.0-bookworm-slim

RUN corepack enable \
    && mkdir -p /opt/induforge/pnpm-store /opt/induforge/corepack /cache \
    && COREPACK_HOME=/opt/induforge/corepack corepack install -g pnpm@11.21.0 \
    && chown -R 1000:1000 /opt/induforge /cache

COPY --from=workspace-assets --chown=1000:1000 /opt/induforge/pnpm-store/ /opt/induforge/pnpm-store/
COPY --from=workspace-assets --chown=1000:1000 /opt/induforge/templates/vite-vue-js/ /opt/induforge/templates/vite-vue-js/

RUN rm -rf /opt/induforge/templates/vite-vue-js/node_modules \
      /opt/induforge/templates/vite-vue-js/dist \
      /opt/induforge/templates/vite-vue-js/.pnpm-store \
    && find /opt/induforge/templates -type d -exec chmod 755 {} + \
    && find /opt/induforge/templates -type f -exec chmod 644 {} +

USER 1000:1000
WORKDIR /build

ENV PNPM_CONFIG_STORE_DIR=/cache/pnpm-store \
    XDG_CACHE_HOME=/cache \
    COREPACK_HOME=/cache/corepack
