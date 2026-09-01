FROM scratch

COPY data-service /data-service
COPY collector-protocols /contracts/collector-protocols
COPY runtime /contracts/runtime

USER 1000:1000
ENTRYPOINT ["/data-service"]
