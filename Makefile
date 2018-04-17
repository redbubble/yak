.PHONY: vendor test install

vendor:
	dep ensure

test:
	go test ./...

fmt:
	go fmt ./...

install:
	go install
