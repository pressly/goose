GO_TEST_FLAGS ?= -race -count=1 -v -timeout=10m

.PHONY: dist
dist:
	@mkdir -p ./bin
	@rm -f ./bin/*
	GOOS=darwin  GOARCH=amd64 go build -o ./bin/goose-darwin64       ./cmd/goose
	GOOS=linux   GOARCH=amd64 go build -o ./bin/goose-linux64        ./cmd/goose
	GOOS=linux   GOARCH=386   go build -o ./bin/goose-linux386       ./cmd/goose
	GOOS=windows GOARCH=amd64 go build -o ./bin/goose-windows64.exe  ./cmd/goose
	GOOS=windows GOARCH=386   go build -o ./bin/goose-windows386.exe ./cmd/goose

.PHONY: build
build:
	go build -o $$GOBIN/goose ./cmd/goose

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
	go test $(GO_TEST_FLAGS) ./tests/e2e/postgres

test-e2e-mysql:
	go test $(GO_TEST_FLAGS) ./tests/e2e/mysql

test-e2e-clickhouse:
	go test $(GO_TEST_FLAGS) ./tests/e2e/clickhouse -test.short

test-e2e-vertica:
	go test $(GO_TEST_FLAGS) ./tests/e2e/vertica

docker-cleanup:
	docker stop -t=0 $$(docker ps --filter="label=goose_test" -aq)

docker-postgres:
	docker run --rm -d \
		-e POSTGRES_USER=dbuser \
		-e POSTGRES_PASSWORD=password1 \
		-e POSTGRES_DB=testdb \
		-p 5433:5432 \
		-l goose_test \
		postgres:14-alpine -c log_statement=all

todo:
	rg --type go --ignore-case '//.*(todo|feat)\('

gh-links:
	rg --type go --ignore-case '//.*github.com/.*/(issues|pull)/[0-9]+'
