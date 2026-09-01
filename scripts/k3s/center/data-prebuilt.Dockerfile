FROM scratch

COPY data-service /data-service
COPY collector-protocols /contracts/collector-protocols

USER 1000:1000
ENTRYPOINT ["/data-service"]
