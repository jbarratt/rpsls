export GO111MODULE=on
BINARY_NAME=smarter_sensibo
.PHONY: all build test clean deps upgrade

all: deps build
build:
	go build -o $(BINARY_NAME) cmd/smarter_sensibo/main.go
test:
	go test -v ./...
clean:
	go clean
deps:
	go build -v ./...
upgrade:
	go get -u
