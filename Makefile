# Should contain src/github.com/Kagami/go-avif
export GOPATH = $(PWD)/../../../..

all: build
precommit: gofmt-staged

build:
	go get ./...

gofmt:
	go fmt ./...

gofmt-staged:
	./gofmt-staged.sh

test:
	go test -v
