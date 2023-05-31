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

.PHONY: all tosca tosca-go tosca-cpp clean clean-go clean-cpp help test test-go test-cpp bench bench-go

all: tosca

tosca: tosca-go tosca-cpp

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
	@cd cpp ; \
	cmake -Bbuild -DCMAKE_BUILD_TYPE=Release -DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so ; \
	cmake --build build --parallel

tosca-cpp-asan:
	@cd cpp ; \
	cmake -Bbuild -DCMAKE_BUILD_TYPE=Debug -DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so -DTOSCA_ASAN=ON ; \
	cmake --build build --parallel

test: test-go test-cpp

test-go:
	@go test ./...

test-cpp: tosca-cpp
	@cd cpp/build ; \
	ctest --output-on-failure

test-cpp-asan: tosca-cpp-asan
	@cd cpp/build ; \
	ctest --output-on-failure

bench: bench-go

bench-go:
	@go test -bench=. ./...

clean: clean-go clean-cpp

clean-go:
	rm -fr ./go/build/*

clean-cpp:
	@cd cpp ; \
	cmake --build build --target clean

help: Makefile
	@echo "Choose a make command in "$(PROJECT)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
