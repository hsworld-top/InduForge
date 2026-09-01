FROM golang:1.25-bookworm AS builder
ARG TARGETOS=linux
ARG TARGETARCH=arm64
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/compute-sandbox ./cmd

FROM node:22-bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends bubblewrap python3 python3-pip util-linux ca-certificates tini \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin sandbox
COPY --from=builder /out/compute-sandbox /usr/local/bin/compute-sandbox
COPY runtime /opt/induforge/runtime
RUN mkdir -p /dependencies \
    && chown -R root:root /opt/induforge/runtime /usr/local/bin/compute-sandbox \
    && chmod -R a-w /opt/induforge/runtime /usr/local/bin/compute-sandbox
# tini 作为容器 PID 1 回收 bubblewrap 隔离进程遗留的孤儿子进程，避免长时调试后耗尽 PID 限额。
# API 进程只负责创建隔离命名空间；实际脚本由 bubblewrap 强制切换到 10001 用户。
EXPOSE 18103
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/compute-sandbox"]
