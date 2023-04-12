# syntax = docker/dockerfile:1.1-experimental
# Dockerfile.build
FROM golang:1.20-alpine as build
WORKDIR /src

RUN apk add build-base

ARG VERSION
COPY . .
RUN --mount=type=cache,target=/root/cache \
	mkdir -p /tmp/bin && \
	CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	cd cmd && \
	for dir in *; do \
		go build -v \
			-ldflags="-X main.Version=$VERSION -extldflags=-static" \
			-mod vendor \
			-o "/tmp/bin/$dir" \
			"./$dir"; \
	done


FROM scratch
COPY --from=build /tmp/bin/kegerator /kegerator

EXPOSE 9220/tcp

CMD ["/kegerator", "--file", "/data/state.json"]
