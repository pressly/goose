FROM golang:1.20 as build

COPY . /duck/
WORKDIR /duck
RUN go mod tidy
ENV CGO_ENABLED=0
RUN go build -o duck \
    -ldflags="-s -w" \
    -tags="no_clickhouse no_mssql no_mysql no_sqlite3 no_vertica" \
    ./cmd/goose

FROM scratch
COPY --from=build /duck/duck /goose
ENTRYPOINT ["/goose"]