VERSION ?= 1.1.2

.PHONY: vendor test install

vendor:
	dep ensure

test:
	go test -v -json ./... | go-passe

fmt:
	go fmt ./...

install:
	go install

release:
	git tag -a "v${VERSION}" -m "Releasing version ${VERSION}"
	git push --tags
	goreleaser --rm-dist
