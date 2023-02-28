BINARY=kegerator
VERSION=$$(vtag)
BUILD=$$(vtag --no-meta)
TAG="subtlepseudonym/${BINARY}:${BUILD}"

PLATFORM?=linux/arm/v6

default: all

all: test build

build: format
	docker buildx build \
		--platform ${PLATFORM} \
		--build-arg VERSION=${VERSION} \
		-f Dockerfile \
		-o type=local,dest=./bin .
	docker stop buildx_buildkit_arm-builder0 && docker rm buildx_buildkit_arm-builder0

build-local: format
	go build -o kegerator -v *.go

docker: format
	docker buildx build \
		--output=type=registry \
		--platform linux/arm/v6,linux/arm/v7,linux/amd64 \
		--build-arg VERSION=${VERSION} \
		--tag ${TAG} \
		-f Dockerfile .
	docker stop buildx_buildkit_arm-builder0 && docker rm buildx_buildkit_arm-builder0

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
