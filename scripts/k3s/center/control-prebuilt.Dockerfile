FROM scratch

COPY dev_core /dev_core

USER 1000:1000
ENTRYPOINT ["/dev_core"]
