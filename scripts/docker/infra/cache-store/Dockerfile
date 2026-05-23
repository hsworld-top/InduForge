FROM redis:7.2-alpine

COPY start-cache-store.sh /usr/local/bin/induforge-cache-store
RUN chmod +x /usr/local/bin/induforge-cache-store

ENTRYPOINT ["induforge-cache-store"]
