FROM timescale/timescaledb:2.26.4-pg16

COPY scripts/docker/infra/meta-store/start-meta-store.sh /usr/local/bin/start-induforge-meta-store
RUN chmod 0755 /usr/local/bin/start-induforge-meta-store

ENTRYPOINT ["/usr/local/bin/start-induforge-meta-store"]
