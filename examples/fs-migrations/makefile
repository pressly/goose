all: build

setup:
	go get -u github.com/shurcool/vfsgen

build:
	cd postgres && go generate -tags dev && cd -
	go build -ldflags '-X main.version=1.0'

clean:
	go clean
	rm -f postgres/assets_vfsdata.go