.PHONY: dist
dist:
	@mkdir -p ./bin
	@rm -f ./bin/*
	GOOS=darwin  GOARCH=amd64 go build -o ./bin/goose-darwin64       ./cmd/goose
	GOOS=linux   GOARCH=amd64 go build -o ./bin/goose-linux64        ./cmd/goose
	GOOS=linux   GOARCH=386   go build -o ./bin/goose-linux386       ./cmd/goose
	GOOS=windows GOARCH=amd64 go build -o ./bin/goose-windows64.exe  ./cmd/goose
	GOOS=windows GOARCH=386   go build -o ./bin/goose-windows386.exe ./cmd/goose

test-packages:
	go test -v $$(go list ./... | grep -v -e /tests -e /bin -e /cmd -e /examples)

test-e2e: test-e2e-postgres test-e2e-mysql

test-e2e-postgres:
	go test -v ./tests/e2e -dialect=postgres

test-e2e-mysql:
	go test -v ./tests/e2e -dialect=mysql

test-clickhouse:
	go test -timeout=10m -count=1 -race -v ./tests/clickhouse -test.short

test-vertica:
	go test -count=1 -v ./tests/vertica

docker-cleanup:
	docker stop -t=0 $$(docker ps --filter="label=goose_test" -aq)

start-postgres:
	docker run --rm -d \
		-e POSTGRES_USER=${GOOSE_POSTGRES_DB_USER} \
		-e POSTGRES_PASSWORD=${GOOSE_POSTGRES_PASSWORD} \
		-e POSTGRES_DB=${GOOSE_POSTGRES_DBNAME} \
		-p ${GOOSE_POSTGRES_PORT}:5432 \
		-l goose_test \
		postgres:14-alpine

.PHONY: clean
clean:
	@find . -type f -name '*.FAIL' -delete

.PHONY: lint
lint: tools
	@golangci-lint run ./... --fix

.PHONY: tools
tools:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1
