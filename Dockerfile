FROM alpine:latest

RUN apk add rpm debootstrap buildah fuse-overlayfs libcap netavark aardvark-dns curl

RUN setcap cap_setuid=ep "$(command -v newuidmap)" && \
    setcap cap_setgid=ep "$(command -v newgidmap)" &&\
    chmod 0755 "$(command -v newuidmap)" && \
    chmod 0755 "$(command -v newgidmap)" && \
    echo "builder:2000:50000" > /etc/subuid && \
    echo "builder:2000:50000" > /etc/subgid

# Create local user for rootless image builds
RUN adduser -u 1000 -D builder && \
    chown -R builder /home/builder

# Make builder the default user when running container
USER builder
WORKDIR /home/builder

ENV BUILDAH_ISOLATION=chroot

ENTRYPOINT ["/usr/bin/buildah", "unshare"]
