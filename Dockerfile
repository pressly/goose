FROM alpine:3.15
# FROM debian:buster-slim
ENV GOOSE_VERSION=3.4.1
RUN apk --no-cache add curl
# RUN apt-get update && apt-get install -y curl
RUN curl -ksSL https://github.com/pressly/goose/releases/download/v${GOOSE_VERSION}/goose_linux_x86_64 -o goose
RUN chmod +x goose
RUN curl -ksSL https://raw.githubusercontent.com/cloudflare/cfssl/master/certdb/sqlite/migrations/001_CreateCertificates.sql -o 001_CreateCertificates.sql
RUN ./goose sqlite3 ./data.db up
RUN ./goose sqlite3 ./data.db status