FROM node:24.19.0-bookworm-slim AS node-runtime

FROM node-runtime AS workspace-assets

RUN test "$(node --version)" = "v24.19.0" \
    && test "$(npm --version)" = "11.17.0" \
    && npm install --global pnpm@11.21.0

WORKDIR /source
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY designer/pi-web ./designer/pi-web
COPY contracts/project-templates ./contracts/project-templates
COPY runtime/web-sdk ./runtime/web-sdk
COPY designer/code-workspace/prepare-runtime-templates.mjs ./designer/code-workspace/prepare-runtime-templates.mjs

# Pi Web 的构建依赖只用于生成最终 tgz，不能进入运行时离线 store；否则会与
# 全局安装的 Pi Web 再次重复，中心节点导入镜像时会额外展开数 GiB。
RUN pnpm install --frozen-lockfile \
      --filter @induforge/pi-web... \
      --store-dir /tmp/induforge-build-pnpm-store \
    && pnpm --dir designer/pi-web build \
    && mkdir -p /tmp/induforge-pi-web \
    && mkdir -p /opt/induforge \
    && pnpm --dir designer/pi-web pack --pack-destination /tmp/induforge-pi-web \
    && printf 'node=%s\nnpm=%s\npnpm=%s\nlock=%s\n' \
      "$(node --version)" \
      "$(npm --version)" \
      "$(pnpm --version)" \
      "$(sha256sum pnpm-lock.yaml | cut -d ' ' -f 1)" \
      > /opt/induforge/pnpm-store.version

# 仅向最终镜像的 store 预热四套模板的运行、构建和 Vite 预览依赖。这里不能
# 复用上一步 Pi Web 的构建 store，否则会把无关的 Next/AI 依赖一并带入。
RUN mkdir -p /tmp/induforge-runtime-sdk /opt/induforge/pnpm-store \
    && pnpm --dir runtime/web-sdk pack --pack-destination /tmp/induforge-runtime-sdk \
    && node designer/code-workspace/prepare-runtime-templates.mjs \
      /source/contracts/project-templates \
      /tmp/induforge-runtime-sdk/induforge-runtime-sdk-0.2.0.tgz \
      /opt/induforge/pnpm-store \
    && for template in /source/contracts/project-templates/vite-*; do \
      pnpm --dir "$template" fetch \
        --prefer-offline \
        --ignore-workspace \
        --store-dir /opt/induforge/pnpm-store; \
    done \
    && printf 'templates=%s\n' \
      "$(sha256sum /source/contracts/project-templates/vite-*/pnpm-lock.yaml | sha256sum | cut -d ' ' -f 1)" \
      >> /opt/induforge/pnpm-store.version

# 在构建阶段清理和冻结模板。若在最终镜像 COPY 后递归 chown pnpm store，
# OverlayFS 会复制整套离线依赖，造成数 GB 的无效镜像层。
RUN rm -rf \
      /source/contracts/project-templates/*/node_modules \
      /source/contracts/project-templates/*/dist \
      /source/contracts/project-templates/*/.pnpm-store \
    && find /source/contracts/project-templates -type d -exec chmod 755 {} + \
    && find /source/contracts/project-templates -type f -exec chmod 644 {} + \
    && chown -R root:root /source/contracts/project-templates /opt/induforge/pnpm-store /opt/induforge/pnpm-store.version \
    && chmod -R a-w /source/contracts/project-templates /opt/induforge/pnpm-store /opt/induforge/pnpm-store.version

FROM codercom/code-server:4.131.0

USER root

COPY --from=node-runtime /usr/local/ /usr/local/
COPY --from=workspace-assets /tmp/induforge-pi-web/ /tmp/induforge-pi-web/

RUN pi_web_package="$(find /tmp/induforge-pi-web -maxdepth 1 -type f -name '*.tgz' -print -quit)" \
    && test -n "$pi_web_package" \
    && npm install --global pnpm@11.21.0 "$pi_web_package" \
    && npm cache clean --force \
    && rm -rf /tmp/induforge-pi-web \
    && apt-get update \
    && apt-get install --yes --no-install-recommends psmisc \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /workspace /cache /opt/induforge/templates /opt/induforge/pi/extensions /home/coder/entrypoint.d /home/coder/.pi/agent \
    && chown -R coder:coder /workspace /cache /home/coder/entrypoint.d /home/coder/.pi

COPY --from=workspace-assets /opt/induforge/pnpm-store /opt/induforge/pnpm-store
COPY --from=workspace-assets /opt/induforge/pnpm-store.version /opt/induforge/pnpm-store.version
COPY --from=workspace-assets /source/contracts/project-templates /opt/induforge/templates

COPY --chown=coder:coder designer/code-workspace/preview-control.mjs /opt/induforge/preview-control.mjs
COPY --chown=coder:coder designer/code-workspace/workspace-initializer.mjs /opt/induforge/workspace-initializer.mjs
COPY --chown=coder:coder designer/code-workspace/vite-runner.mjs /opt/induforge/vite-runner.mjs
COPY --chown=coder:coder designer/code-workspace/code-server-settings.json /opt/induforge/code-server-settings.json
COPY --chown=coder:coder --chmod=755 designer/code-workspace/initialize-cache.sh /home/coder/entrypoint.d/10-induforge-cache.sh
COPY --chown=coder:coder --chmod=755 designer/code-workspace/workspace-entrypoint.sh /usr/local/bin/induforge-workspace-entrypoint
COPY --chown=coder:coder --chmod=755 designer/code-workspace/start-workspace.sh /usr/local/bin/start-induforge-workspace

USER coder
WORKDIR /workspace

ENV PI_CODING_AGENT_DIR=/home/coder/.pi/agent \
    PI_WEB_EMBEDDED=1 \
    PI_WEB_WORKSPACE_ROOT=/workspace \
    PI_WEB_ALLOWED_PARENT_ORIGINS=http://127.0.0.1:18601,http://localhost:18601,http://127.0.0.1:18603,http://localhost:18603 \
    PI_WEB_PLATFORM_EXTENSIONS_DIR=/opt/induforge/pi/extensions \
    PI_WEB_HOSTNAME=0.0.0.0 \
    PI_WEB_NO_OPEN=1 \
    PI_WEB_PORT=30141 \
    WORKSPACE_ROOT=/workspace \
    WORKSPACE_TEMPLATES_ROOT=/opt/induforge/templates \
    PNPM_CONFIG_STORE_DIR=/cache/pnpm-store \
    PNPM_CONFIG_TRUST_LOCKFILE=true \
    PREVIEW_CONTROL_ALLOWED_ORIGINS=http://127.0.0.1:18601,http://localhost:18601,http://127.0.0.1:18603,http://localhost:18603

EXPOSE 3000 30141 5173 5174

HEALTHCHECK --interval=2s --timeout=2s --start-period=10s --retries=30 \
  CMD node -e "Promise.all([fetch('http://127.0.0.1:3000/healthz'),fetch('http://127.0.0.1:30141'),fetch('http://127.0.0.1:5174/health')]).then(responses=>process.exit(responses.every(response=>response.ok)?0:1)).catch(()=>process.exit(1))"

ENTRYPOINT ["/usr/local/bin/induforge-workspace-entrypoint"]
