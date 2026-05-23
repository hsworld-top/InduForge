FROM chrislusf/seaweedfs:3.85

COPY start-object-store.sh /usr/local/bin/induforge-object-store
RUN chmod +x /usr/local/bin/induforge-object-store

ENTRYPOINT ["induforge-object-store"]
