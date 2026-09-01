FROM golang:1.26-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /src

COPY go.mod go.sum ./
# go.mod 使用本地 socket.io replace，依赖下载前必须先提供该模块元数据。
COPY third_party/socket.io-golang ./third_party/socket.io-golang
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/data-service ./cmd

FROM alpine:3.22

WORKDIR /app

RUN sed -i 's#https://dl-cdn.alpinelinux.org/alpine#https://mirrors.aliyun.com/alpine#g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates

COPY --from=builder /out/data-service ./data-service
# 采集协议目录由运行时配置指定为 /contracts/collector-protocols，需随镜像发布。
COPY contracts/collector-protocols /contracts/collector-protocols

EXPOSE 18102

CMD ["./data-service"]
