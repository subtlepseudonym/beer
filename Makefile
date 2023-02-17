BINARY=kegerator

BUILD=$$(vtag)

REVISION=`git rev-list -n1 HEAD`
BUILDTAGS=
LDFLAGS=--ldflags "-X main.Version=${BUILD} -X main.Revision=${REVISION}"

default: all

all: test build

build: format
	docker buildx build -f Dockerfile.build -o type=local,dest=./bin .

build-local: format
	go build -o kegerator -v *.go

test:
	gotest --race ./...

format fmt:
	gofmt -l -w .

clean:
	go mod tidy
	go mod vendor
	go clean

get-tag:
	echo ${BUILD}

.PHONY: all build dev-build test format fmt clean get-tag
