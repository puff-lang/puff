# Makefile
.PHONY: build test fmt vet check clean

build:
	cd src && go build -o ../bin/puff ./cmd/puff

test:
	cd src && go test ./...

fmt:
	cd src && go fmt ./...

vet:
	cd src && go vet ./...

check: fmt vet test

clean:
	rm -rf bin dist coverage.out
