# --------------------------------------------------------------------------
# Makefile for the Fantom Tosca - long format EVM
#
# v1.0 (2023/03/28) - Initial version
#
# (c) Fantom Foundation, 2023
# --------------------------------------------------------------------------

TOSCA_CPP_BUILD = Release
TOSCA_CPP_ASSERT = ON
TOSCA_CPP_ASAN = OFF

.PHONY: all tosca tosca-go tosca-cpp test test-go test-cpp test-cpp-asan \
        bench bench-go clean clean-go clean-cpp evmone evmone-clean

all: tosca

tosca: tosca-go

tosca-go: tosca-cpp evmone
	go build ./...

tosca-cpp:
	cd cpp ; \
	cmake -Bbuild -DCMAKE_BUILD_TYPE="$(TOSCA_CPP_BUILD)" -DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so \
		-DTOSCA_ASSERT="$(TOSCA_CPP_ASSERT)" -DTOSCA_ASAN="$(TOSCA_CPP_ASAN)"; \
	cmake --build build --parallel

evmone:
	@cd third_party/evmone ; \
	cmake -Bbuild -DCMAKE_BUILD_TYPE=Release -DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so ; \
	cmake --build build --parallel

test: test-go test-cpp

test-go: tosca-go
	@go test ./... -count 1

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

clean: clean-go clean-cpp clean-evmone

clean-evmone:
	$(RM) -r ./third_party/evmone/build

clean-go:
	$(RM) -r ./go/build/*

clean-cpp:
	$(RM) -r ./cpp/build
