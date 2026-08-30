FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/runtime-engine ./cmd/runtime-engine

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/runtime-engine /usr/local/bin/runtime-engine
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/runtime-engine"]
