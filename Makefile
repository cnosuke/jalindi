NAME     := jalindi
VERSION  := $(shell git describe --tags)
REVISION := $(shell git rev-parse --short HEAD)
SRCS    := $(shell find . -type f -name '*.go')
LDFLAGS := -ldflags="-s -w -X \"main.Version=$(VERSION)\" -X \"main.Revision=$(REVISION)\" -extldflags \"-static\""

.PHONY: clean cross-build release-pack glide deps gen-proto gen-proto-gw

all: $(SRCS) bin/$(NAME) bin/gateway

bin/$(NAME): $(SRCS)
	go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o bin/$(NAME)

bin/gateway: $(shell find json_gateway -type f -name '*.go')
	go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o bin/gateway json_gateway/*.go

clean:
	rm -rf bin/* dist/* vendor/*

cross-build:
	for os in darwin linux; do \
		GOOS=$$os GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o dist/$$os-amd64/$(NAME); \
	done

release-pack: cross-build
	for os in darwin linux; do \
		zip -j dist/$$os-amd64.zip dist/$$os-amd64/$(NAME); \
	done

dep:
ifeq ($(shell command -v dep 2> /dev/null),)
	go install -v github.com/golang/dep/cmd/de
endif

deps: dep
	dep ensure -v

gen-proto:
	@mkdir -p pb/
	@protoc \
		-I proto/ \
		--go_out=plugins=grpc:pb/ \
		proto/*.proto

gen-proto-gw:
	@mkdir -p pb/
	@protoc \
		-I $$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I proto/ \
		--grpc-gateway_out=logtostderr=true:pb/ \
		proto/*.proto
