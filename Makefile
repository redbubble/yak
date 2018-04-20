.PHONY: vendor test install

vendor:
	dep ensure

test:
	go test -v -json ./... | go-passe

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/yak_linux_x86
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/yak_darwin_x86

fmt:
	go fmt ./...

install:
	go install
