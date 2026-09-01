FROM golang:1.25-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH=arm64
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/industrial_collector ./cmd/industrial_collector

FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.opencontainers.image.version="1.0.0" org.opencontainers.image.ref.name="induforge/collector-engine:1.0.0"
COPY --from=build /out/industrial_collector /industrial_collector
USER nonroot:nonroot
ENTRYPOINT ["/industrial_collector"]
