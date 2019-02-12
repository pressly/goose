dist:
	@mkdir -p ./bin
	@rm -f ./bin/*
	GOOS=darwin GOARCH=amd64 go build -o ./bin/gander-darwin64 ./cmd/gander
	GOOS=linux GOARCH=amd64 go build -o ./bin/gander-linux64 ./cmd/gander
	GOOS=linux GOARCH=386 go build -o ./bin/gander-linux386 ./cmd/gander

