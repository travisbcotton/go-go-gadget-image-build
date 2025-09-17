# Builder image
FROM golang:1.25 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . . 
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/gogo-imgbuild ./cmd/gogo-imgbuild

# Run image 
FROM alpine:latest
RUN apk add rpm debootstrap buildah fuse-overlayfs libcap netavark aardvark-dns curl file
RUN setcap cap_setuid=ep "$(command -v newuidmap)" && \
    setcap cap_setgid=ep "$(command -v newgidmap)" &&\
    chmod 0755 "$(command -v newuidmap)" && \
    chmod 0755 "$(command -v newgidmap)" && \
    echo "builder:2000:50000" > /etc/subuid && \
    echo "builder:2000:50000" > /etc/subgid
# Create local user for rootless image builds
RUN adduser -u 1000 -D builder && \
    chown -R builder /home/builder
# Copy executable from builder image
COPY --from=builder /out/gogo-imgbuild /usr/local/bin/gogo-imgbuild
RUN chmod +x /usr/local/bin/gogo-imgbuild 

# Make builder the default user when running container
USER builder
WORKDIR /home/builder

ENV BUILDAH_ISOLATION=chroot

ENTRYPOINT ["/usr/bin/buildah", "unshare"]
