# --------------------------------------------------------------------------
# Makefile for the Fantom Tosca - long format EVM
#
# v1.0 (2023/03/28) - Initial version
#
# (c) Fantom Foundation, 2023
# --------------------------------------------------------------------------

# what are we building
PROJECT := $(shell basename "$(PWD)")
GO_BIN := $(CURDIR)/go/build

# compile time variables will be injected into the app
APP_VERSION := 1.0
BUILD_DATE := $(shell date "+%a, %d %b %Y %T")
BUILD_COMPILER := $(shell go version)
BUILD_COMMIT := $(shell git show --format="%H" --no-patch)
BUILD_COMMIT_TIME := $(shell git show --format="%cD" --no-patch)
GOPROXY ?= "https://proxy.golang.org,direct"

TOSCA_CPP_BUILD = Release
TOSCA_CPP_ASSERT = ON
TOSCA_CPP_ASAN = OFF

.PHONY: all tosca tosca-go test test-go \
        bench bench-go clean clean-go

all: tosca

tosca: tosca-go

tosca-go:
	@cd go/vm/lfvm ; \
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/go-ethereum-substate \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Tosca/utils.GitCommit=$(BUILD_COMMIT)'" \
		-o $(GO_BIN)/tosca \

test: test-go

test-go: tosca-go
	@go test ./... -count 1

bench: TOSCA_CPP_ASSERT = OFF

bench-go: TOSCA_CPP_ASSERT = OFF
bench-go:
	@go test -bench=. ./...

clean: clean-go

clean-go:
	$(RM) -r ./go/build/*
