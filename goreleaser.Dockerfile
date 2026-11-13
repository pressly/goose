# This Dockerfile is used by GoReleaser (see dockers_v2 in .goreleaser.yaml).
# It copies the prebuilt goose binary, so it does not work with a plain
# `docker build .` from the repository root.
#
# The distroless static image has CA certificates, tzdata, and a nonroot user,
# but no shell.
FROM gcr.io/distroless/static-debian13:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/goose /usr/local/bin/goose
WORKDIR /migrations
ENTRYPOINT ["/usr/local/bin/goose"]
