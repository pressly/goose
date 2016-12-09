dist:
	@mkdir -p ./bin
	@rm -f ./bin/*
	GOOS=darwin GOARCH=amd64 go build -o ./bin/goose-darwin64 ./cmd/goose
	GOOS=linux GOARCH=amd64 go build -o ./bin/goose-linux64 ./cmd/goose
	GOOS=linux GOARCH=386 go build -o ./bin/goose-linux386 ./cmd/goose

