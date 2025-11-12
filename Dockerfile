# syntax=docker/dockerfile:1.8

FROM --platform=$BUILDPLATFORM golang:1.25.4-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GOOSE_BUILD_TAGS=""

ENV CGO_ENABLED=0 \
    GOOSE_BUILD_TAGS=${GOOSE_BUILD_TAGS}

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH /bin/sh -c '\
    set -e; \
    if [ -n "$GOOSE_BUILD_TAGS" ]; then \
      go build -trimpath -tags "$GOOSE_BUILD_TAGS" -ldflags "-s -w" -o /out/goose ./cmd/goose; \
    else \
      go build -trimpath -ldflags "-s -w" -o /out/goose ./cmd/goose; \
    fi'

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/goose /usr/local/bin/goose

WORKDIR /migrations

ENTRYPOINT ["goose"]
CMD ["--help"]
