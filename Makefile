BINARY=kegerator
BUILD=$$(vtag --no-meta)
TAG="subtlepseudonym/${BINARY}:${BUILD}"

default: all

all: test build

build: format
	docker buildx build -f Dockerfile -o type=local,dest=./bin/kegerator .

build-local: format
	go build -o kegerator -v *.go

docker: format
	docker build --network=host --tag ${TAG} -f Dockerfile .

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
