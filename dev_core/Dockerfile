FROM golang:1.26-alpine AS builder

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /src

COPY dev_core/go.mod dev_core/go.sum ./
RUN go mod download

COPY dev_core ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dev_core ./cmd/dev_core

FROM alpine:3.23

RUN addgroup -g 1000 -S induforge && adduser -u 1000 -S induforge -G induforge \
    && mkdir -p /var/lib/induforge/workspaces \
    && chown -R induforge:induforge /var/lib/induforge

WORKDIR /app
COPY --from=builder /out/dev_core /app/dev_core

ENV DEV_CORE_ADDR=:18101
ENV CODE_WORKSPACE_ROOT=/var/lib/induforge/workspaces
EXPOSE 18101

USER induforge
ENTRYPOINT ["/app/dev_core"]
