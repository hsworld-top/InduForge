FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/runtime-api ./cmd/runtime-api

FROM alpine:3.22
RUN addgroup -S induforge && adduser -S -G induforge -u 65532 induforge
COPY --from=build /out/runtime-api /usr/local/bin/runtime-api
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/runtime-api"]
