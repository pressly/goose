GO_TEST_FLAGS ?= -race -count=1 -v -timeout=10m

# These are the default values for the test database. They can be overridden
DB_USER ?= dbuser
DB_PASSWORD ?= password1
DB_NAME ?= testdb
DB_POSTGRES_PORT ?= 5433
DB_MYSQL_PORT ?= 3307
DB_CLICKHOUSE_PORT ?= 9001

.PHONY: dist
dist:
	@mkdir -p ./bin
	@rm -f ./bin/*
	GOOS=darwin  GOARCH=amd64 go build -o ./bin/goose-darwin64       ./cmd/goose
	GOOS=linux   GOARCH=amd64 go build -o ./bin/goose-linux64        ./cmd/goose
	GOOS=linux   GOARCH=386   go build -o ./bin/goose-linux386       ./cmd/goose
	GOOS=windows GOARCH=amd64 go build -o ./bin/goose-windows64.exe  ./cmd/goose
	GOOS=windows GOARCH=386   go build -o ./bin/goose-windows386.exe ./cmd/goose

.PHONY: clean
clean:
	@find . -type f -name '*.FAIL' -delete

.PHONY: lint
lint: tools
	@golangci-lint run ./... --fix

.PHONY: tools
tools:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

test-packages:
	go test $(GO_TEST_FLAGS) $$(go list ./... | grep -v -e /tests -e /bin -e /cmd -e /examples)

test-e2e: test-e2e-postgres test-e2e-mysql test-e2e-clickhouse test-e2e-vertica

test-e2e-postgres:
	go test $(GO_TEST_FLAGS) ./tests/e2e -dialect=postgres

test-e2e-mysql:
	go test $(GO_TEST_FLAGS) ./tests/e2e -dialect=mysql

test-e2e-clickhouse:
	go test $(GO_TEST_FLAGS) ./tests/clickhouse -test.short

test-e2e-vertica:
	go test $(GO_TEST_FLAGS) ./tests/vertica

docker-cleanup:
	docker stop -t=0 $$(docker ps --filter="label=goose_test" -aq)

docker-postgres:
	docker run --rm -d \
		-e POSTGRES_USER=$(DB_USER) \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-e POSTGRES_DB=$(DB_NAME) \
		-p $(DB_POSTGRES_PORT):5432 \
		-l goose_test \
		postgres:14-alpine -c log_statement=all

docker-mysql:
	docker run --rm -d \
		-e MYSQL_ROOT_PASSWORD=rootpassword1 \
		-e MYSQL_DATABASE=$(DB_NAME) \
		-e MYSQL_USER=$(DB_USER) \
		-e MYSQL_PASSWORD=$(DB_PASSWORD) \
		-p $(DB_MYSQL_PORT):3306 \
		-l goose_test \
		mysql:8.0.31

docker-clickhouse:
	docker run --rm -d \
		-e CLICKHOUSE_DB=$(DB_NAME) \
		-e CLICKHOUSE_USER=$(DB_USER) \
		-e CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT=1 \
		-e CLICKHOUSE_PASSWORD=$(DB_PASSWORD) \
		-p $(DB_CLICKHOUSE_PORT):9000/tcp \
		-l goose_test \
		clickhouse/clickhouse-server:23-alpine
