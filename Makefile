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

.PHONY: all tosca tosca-go tosca-cpp test test-go test-cpp test-cpp-asan \
        bench bench-go clean clean-go clean-cpp

all: tosca

tosca: tosca-go

tosca-go: tosca-cpp
	@cd third_party/evmone ; \
	cmake -Bbuild -DCMAKE_BUILD_TYPE=Release -DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so ; \
	cmake --build build --parallel
	@cd go/vm/lfvm ; \
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/go-ethereum-substate \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Tosca/utils.GitCommit=$(BUILD_COMMIT)'" \
		-o $(GO_BIN)/tosca \

tosca-cpp:
	cd cpp ; \
	bazel build //...

test: test-go test-cpp

test-go: tosca-go
	@go test ./... -count 1

test-cpp:
	@cd cpp ; \
	bazel test --config=asan --nocache_test_results --test_output=errors //...

bench: bench-cpp bench-go

bench-go:
	@go test -bench=. ./...

bench-cpp:
	@cd cpp ; \
	bazel test --config=bench --nocache_test_results --test_output=errors //...

clean: clean-go clean-cpp

clean-go:
	$(RM) -r ./go/build/*

clean-cpp:
	cd cpp ; \
	bazel clean --expunge
