BINARY=kegerator
BUILD=$$(vtag --no-meta)
TAG="subtlepseudonym/${BINARY}:${BUILD}"

default: all

all: test build

build: format
	docker buildx build -f Dockerfile -o type=local,dest=./bin .

build-local: format
	go build -o kegerator -v *.go

docker: format
	docker buildx build \
		--output=type=registry \
		--platform linux/arm/v6,linux/arm/v7,linux/amd64 \
		--tag ${TAG} \
		-f Dockerfile .

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

.PHONY: all build build-local docker test format fmt clean get-tag
