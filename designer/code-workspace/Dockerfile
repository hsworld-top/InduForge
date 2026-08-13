FROM node:22.19.0-bookworm-slim AS node-runtime

FROM codercom/code-server:4.131.0

USER root

COPY --from=node-runtime /usr/local/ /usr/local/

RUN npm install --global pnpm@10.19.0 @agegr/pi-web@0.8.8-beta.1 \
    && npm cache clean --force \
    && apt-get update \
    && apt-get install --yes --no-install-recommends psmisc \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /workspace /cache /opt/induforge/template /opt/induforge/pnpm-store /home/coder/entrypoint.d /home/coder/.pi/agent \
    && chown -R coder:coder /workspace /cache /opt/induforge /home/coder/entrypoint.d /home/coder/.pi

COPY --chown=coder:coder contracts/project-templates/vue-vite /opt/induforge/template
COPY --chown=coder:coder designer/code-workspace/preview-control.mjs /opt/induforge/preview-control.mjs
COPY --chown=coder:coder designer/code-workspace/code-server-settings.json /opt/induforge/code-server-settings.json
COPY --chown=coder:coder --chmod=755 designer/code-workspace/initialize-cache.sh /home/coder/entrypoint.d/10-induforge-cache.sh
COPY --chown=coder:coder --chmod=755 designer/code-workspace/workspace-entrypoint.sh /usr/local/bin/induforge-workspace-entrypoint
COPY --chown=coder:coder --chmod=755 designer/code-workspace/start-workspace.sh /usr/local/bin/start-induforge-workspace

USER coder
WORKDIR /opt/induforge/template

RUN node --version \
    && pnpm --version \
    && pnpm fetch --frozen-lockfile --store-dir /opt/induforge/pnpm-store \
    && printf 'pnpm=%s\nlock=%s\n' \
      "$(pnpm --version)" \
      "$(sha256sum pnpm-lock.yaml | cut -d ' ' -f 1)" \
      > /opt/induforge/pnpm-store.version

WORKDIR /workspace

ENV PI_CODING_AGENT_DIR=/home/coder/.pi/agent \
    PI_WEB_HOSTNAME=0.0.0.0 \
    PI_WEB_NO_OPEN=1 \
    PI_WEB_PORT=30141 \
    PREVIEW_CONTROL_ALLOWED_ORIGINS=http://127.0.0.1:18601,http://localhost:18601,http://127.0.0.1:18603,http://localhost:18603

EXPOSE 3000 30141 5173 5174

HEALTHCHECK --interval=2s --timeout=2s --start-period=10s --retries=30 \
  CMD node -e "Promise.all([fetch('http://127.0.0.1:3000/healthz'),fetch('http://127.0.0.1:30141'),fetch('http://127.0.0.1:5174/health')]).then(responses=>process.exit(responses.every(response=>response.ok)?0:1)).catch(()=>process.exit(1))"

ENTRYPOINT ["/usr/local/bin/induforge-workspace-entrypoint"]
