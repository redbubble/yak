VERSION ?= 1.0.0

.PHONY: vendor test install

vendor:
	dep ensure

test:
	go test -v -json ./... | go-passe

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/yak_linux_${VERSION}_x86_64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/yak_darwin_${VERSION}_x86_64

fmt:
	go fmt ./...

install:
	go install

release:
	git tag -a "v${VERSION}" -m "Releasing version ${VERSION}"
	git push --tags
	goreleaser
