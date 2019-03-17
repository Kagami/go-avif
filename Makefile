# Should contain src/github.com/Kagami/go-avif
export GOPATH = $(PWD)/../../../..

all: build
precommit: gofmt-staged

build:
	go get github.com/Kagami/go-avif/...

gofmt:
	go fmt github.com/Kagami/go-avif/...

gofmt-staged:
	./gofmt-staged.sh

test:
	go test -v
