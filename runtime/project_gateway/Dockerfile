FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/project-gateway ./cmd/project-gateway

# Release 子制品在 NodeAgent 激活的只读根目录中。入口脚本只解开 client
# 到 emptyDir，绝不改写 Release，也不下载或解释外部路径。
FROM alpine:3.22
RUN addgroup -S induforge && adduser -S -G induforge -u 65532 induforge
COPY --from=build /out/project-gateway /usr/local/bin/project-gateway
COPY docker-entrypoint.sh /usr/local/bin/project-gateway-entrypoint
RUN chmod 0555 /usr/local/bin/project-gateway-entrypoint
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/project-gateway-entrypoint"]
