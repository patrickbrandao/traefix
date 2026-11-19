.PHONY: default check test build image run

IMAGE_NAME := tmsoftbrasil/traefix-infopage

default: check test build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o infopage

test:
	go test -v -cover ./...

check:
	golangci-lint run

image:
	docker build -t $(IMAGE_NAME) .

run: build
	./infopage -port 8080 -name local

protoc:
	 protoc --proto_path . ./grpc.proto --go-grpc_out=./ --go_out=./
