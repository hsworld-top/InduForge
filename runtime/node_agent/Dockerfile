# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o node_agent ./cmd/main.go

# 运行阶段
FROM alpine:latest

WORKDIR /root/

# 安装必要的工具
RUN apk --no-cache add ca-certificates

# 从构建阶段复制二进制文件
COPY --from=builder /app/node_agent .

# 创建目录
RUN mkdir -p /var/lib/node_agent/data \
    /var/lib/node_agent/runtime \
    /var/log/node_agent/runtime

# 设置权限
RUN chmod +x node_agent

# 暴露端口
EXPOSE 8080

# 设置入口点
ENTRYPOINT ["./node_agent"]
