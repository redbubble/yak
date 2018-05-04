VERSION ?= 1.1.3
GIT_HASH = $(shell git rev-parse --short HEAD)

.PHONY: vendor test install

vendor:
	dep ensure

test:
	go test -v -json ./... | go-passe

fmt:
	go fmt ./...

install:
	go install -ldflags "-X main.version=${VERSION}-${GIT_HASH}-dev"

release:
	git tag -a "v${VERSION}" -m "Releasing version ${VERSION}"
	git push --tags
	goreleaser --rm-dist
