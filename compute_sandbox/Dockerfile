FROM golang:1.25-bookworm AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/compute-sandbox ./cmd

FROM node:22-bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends bubblewrap python3 util-linux ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin sandbox
COPY --from=builder /out/compute-sandbox /usr/local/bin/compute-sandbox
COPY runtime /opt/induforge/runtime
RUN chown -R root:root /opt/induforge/runtime /usr/local/bin/compute-sandbox \
    && chmod -R a-w /opt/induforge/runtime /usr/local/bin/compute-sandbox
# API 进程只负责创建隔离命名空间；实际脚本由 bubblewrap 强制切换到 10001 用户。
EXPOSE 18103
ENTRYPOINT ["/usr/local/bin/compute-sandbox"]
