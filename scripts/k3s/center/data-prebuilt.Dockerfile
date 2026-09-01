FROM scratch

COPY data-service /data-service

USER 1000:1000
ENTRYPOINT ["/data-service"]
