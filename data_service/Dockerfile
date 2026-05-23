FROM golang:1.26-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/data-service ./cmd

FROM alpine:3.22

WORKDIR /app

RUN sed -i 's#https://dl-cdn.alpinelinux.org/alpine#https://mirrors.aliyun.com/alpine#g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates

COPY --from=builder /out/data-service ./data-service

EXPOSE 18102

CMD ["./data-service"]
