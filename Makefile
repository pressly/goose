dist:
	@mkdir -p ./bin
	@rm -f ./bin/*
	$(CGO_FLAG) GOOS=darwin GOARCH=amd64 go build -o ./bin/goose-darwin64 ./cmd/goose
	$(CGO_FLAG) GOOS=linux GOARCH=amd64 go build -o ./bin/goose-linux64 ./cmd/goose
	$(CGO_FLAG) GOOS=linux GOARCH=386 go build -o ./bin/goose-linux386 ./cmd/goose

