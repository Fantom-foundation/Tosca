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
	cmake -Bbuild -DCMAKE_BUILD_TYPE="$(TOSCA_CPP_BUILD)" -DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so \
		-DTOSCA_ASSERT="$(TOSCA_CPP_ASSERT)" -DTOSCA_ASAN="$(TOSCA_CPP_ASAN)"; \
	cmake --build build --parallel

test: test-go test-cpp

test-go: tosca-go
	@go test ./...

test-cpp: tosca-cpp
	@cd cpp/build ; \
	ctest --output-on-failure

test-cpp-asan: TOSCA_CPP_BUILD = Debug
test-cpp-asan: TOSCA_CPP_ASAN = ON
test-cpp-asan: test-cpp

bench: TOSCA_CPP_ASSERT = OFF
bench: tosca-cpp bench-go

bench-go: TOSCA_CPP_ASSERT = OFF
bench-go:
	@go test -bench=. ./...

clean: clean-go clean-cpp

clean-go:
	$(RM) -r ./go/build/*

clean-cpp:
	$(RM) -r ./cpp/build
