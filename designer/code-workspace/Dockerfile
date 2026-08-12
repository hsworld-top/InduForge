FROM node:22.18.0-bookworm-slim AS node-runtime

FROM codercom/code-server:4.131.0

USER root

COPY --from=node-runtime /usr/local/ /usr/local/

RUN npm install --global pnpm@10.19.0 \
    && npm cache clean --force \
    && mkdir -p /workspace /opt/induforge/template /opt/induforge/pnpm-store /home/coder/entrypoint.d \
    && chown -R coder:coder /workspace /opt/induforge /home/coder/entrypoint.d

COPY --chown=coder:coder dev_core/internal/project/template/package.json /opt/induforge/template/package.json
COPY --chown=coder:coder dev_core/internal/project/template/pnpm-lock.yaml /opt/induforge/template/pnpm-lock.yaml
COPY --chown=coder:coder dev_core/internal/project/template/.induforge/runtime-sdk /opt/induforge/template/.induforge/runtime-sdk
COPY --chown=coder:coder --chmod=755 designer/code-workspace/initialize-cache.sh /home/coder/entrypoint.d/10-induforge-cache.sh

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

EXPOSE 3000

HEALTHCHECK --interval=2s --timeout=2s --start-period=5s --retries=30 \
  CMD node -e "fetch('http://127.0.0.1:3000/healthz').then(response=>process.exit(response.ok?0:1)).catch(()=>process.exit(1))"

ENTRYPOINT ["/usr/bin/entrypoint.sh"]
CMD ["--bind-addr", "0.0.0.0:3000", "--auth", "none", "--disable-telemetry", "--idle-timeout-seconds", "1800", "/workspace"]
