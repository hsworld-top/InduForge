# 正式发布只需要 Node、Corepack 与经审核的离线 pnpm store；不能复用包含 IDE、
# code-server 和 AI 工作台的开发镜像，以控制中心节点的磁盘与攻击面。
FROM node:24.19.0-bookworm-slim AS assets
RUN corepack enable && corepack prepare pnpm@11.21.0 --activate
WORKDIR /source
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY contracts/project-templates ./contracts/project-templates
COPY runtime/web-sdk ./runtime/web-sdk
COPY designer/code-workspace/prepare-runtime-templates.mjs ./designer/code-workspace/prepare-runtime-templates.mjs
RUN mkdir -p /tmp/induforge-runtime-sdk /opt/induforge/pnpm-store \
    && pnpm --dir runtime/web-sdk pack --pack-destination /tmp/induforge-runtime-sdk \
    && node designer/code-workspace/prepare-runtime-templates.mjs /source/contracts/project-templates /tmp/induforge-runtime-sdk/induforge-runtime-sdk-0.2.0.tgz /opt/induforge/pnpm-store \
    && for template in /source/contracts/project-templates/vite-*; do pnpm --dir "$template" install --frozen-lockfile --ignore-workspace --config.trust-lockfile=true --ignore-scripts --store-dir /opt/induforge/pnpm-store; done \
    && sha256sum /source/contracts/project-templates/vite-*/pnpm-lock.yaml > /opt/induforge/pnpm-store.version

FROM node:24.19.0-bookworm-slim
RUN corepack enable && mkdir -p /opt/induforge/corepack && COREPACK_HOME=/opt/induforge/corepack corepack install -g pnpm@11.21.0
COPY --from=assets --chown=1000:1000 /opt/induforge/pnpm-store /opt/induforge/pnpm-store
COPY --from=assets --chown=1000:1000 /opt/induforge/pnpm-store.version /opt/induforge/pnpm-store.version
COPY --from=assets --chown=1000:1000 /source/contracts/project-templates /opt/induforge/templates
RUN rm -rf /opt/induforge/templates/*/node_modules /opt/induforge/templates/*/dist /opt/induforge/templates/*/.pnpm-store \
    && find /opt/induforge/templates -type d -exec chmod 755 {} + \
    && find /opt/induforge/templates -type f -exec chmod 644 {} + \
    && chmod -R a-w /opt/induforge/pnpm-store /opt/induforge/pnpm-store.version /opt/induforge/templates

USER 1000:1000
WORKDIR /build

ENV PNPM_CONFIG_STORE_DIR=/tmp/pnpm-store \
    PNPM_CONFIG_TRUST_LOCKFILE=true \
    XDG_CACHE_HOME=/tmp/pnpm-cache \
    COREPACK_HOME=/tmp/corepack
