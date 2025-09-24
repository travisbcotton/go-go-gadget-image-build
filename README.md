# gogo-imgbuild
A toy example testing building distro images in a super minimal way

## Build it
Easier in a container:
```bash
podman build -t gobuild:latest -f Dockerfile .
```

## Get a shell in a container
Get a shell in the build container to test it out
```bash
podman run \
    --userns=keep-id \
    --rm \
    --device /dev/fuse \
    -it \
    -v ./tests/:/data \
    localhost/gobuild:latest bash
```

## checkout the test file in `tests/micro.yaml`
In the container ENV try a build
```bash
gogo-imgbuild --config /data/micro.yaml
```

## Build on top of
You can use this now to install more packages and stuff
```bash
CNAME=$(buildah from localhost/custom-base:latest)
buildah run --tty $CNAME bash
dnf groupinstall "Minimal Install"
```