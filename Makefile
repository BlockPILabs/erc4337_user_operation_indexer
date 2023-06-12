.PHONY: all build clean

VERSION := $(shell git describe --tags --always --long)

GOLDFLAGS += -X github.com/BlockPILabs/erc4337_user_operation_indexer/version.Version=$(VERSION)
GOFLAGS = -ldflags "$(GOLDFLAGS)"
all: build

build:
	go build $(GOFLAGS) -o ./build/indexer ./cmd/
clean:
	rm -rf build/
