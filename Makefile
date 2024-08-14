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
TOSCA_CPP_COVERAGE = OFF

.PHONY: all tosca tosca-go tosca-cpp tosca-rust test test-go test-cpp test-rust test-cpp-asan \
        bench bench-go clean clean-go clean-cpp clean-rust evmone evmone-clean license-headers

all: tosca

tosca: tosca-go

tosca-go: tosca-cpp tosca-rust evmone
	go build ./...

tosca-cpp:
	cd cpp ; \
	cmake \
		-Bbuild \
		-DCMAKE_BUILD_TYPE="$(TOSCA_CPP_BUILD)" \
		-DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so \
		-DTOSCA_ASSERT="$(TOSCA_CPP_ASSERT)" \
		-DTOSCA_COVERAGE="$(TOSCA_CPP_COVERAGE)" \
		-DTOSCA_ASAN="$(TOSCA_CPP_ASAN)"; \
	cmake --build build --parallel

tosca-cpp-coverage: TOSCA_CPP_BUILD = Debug
tosca-cpp-coverage: TOSCA_CPP_COVERAGE = ON
tosca-cpp-coverage: tosca-cpp

tosca-rust:
	cargo build --release

evmone:
	@cd third_party/evmone ; \
	cmake -Bbuild -DCMAKE_BUILD_TYPE=Release -DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so ; \
	cmake --build build --parallel -t evmone

coverage-go: DATE=$(shell date +"%Y-%m-%d-%T")
coverage-go: TOSCA_GO_COVERAGE_DIR=./go/build/coverage/${TOSCA_GO_COVERAGE_EVM}/${DATE}
coverage-go:
	go build -cover -o ${TOSCA_GO_COVERAGE_DIR}/driver -coverpkg=./go/...,${TOSCA_GO_COVERAGE_DEPENDENCY_PACKAGES} ./go/ct/driver/ ; \
	GOCOVERDIR=${TOSCA_GO_COVERAGE_DIR} ${TOSCA_GO_COVERAGE_DIR}/driver run --max-errors 1 ${TOSCA_GO_COVERAGE_EVM} ; \
	go tool covdata textfmt --i ${TOSCA_GO_COVERAGE_DIR} -o ${TOSCA_GO_COVERAGE_DIR}/driver_coverage_report.txt ; \
	go tool cover -html ${TOSCA_GO_COVERAGE_DIR}/driver_coverage_report.txt -o ${TOSCA_GO_COVERAGE_DIR}/coverage_output.html

ct-coverage-lfvm: TOSCA_GO_COVERAGE_EVM=lfvm
ct-coverage-lfvm: coverage-go

ct-coverage-geth: TOSCA_GO_COVERAGE_EVM=geth
ct-coverage-geth: TOSCA_GO_COVERAGE_DEPENDENCY_PACKAGES=github.com/ethereum/go-ethereum/core/vm/...
ct-coverage-geth: coverage-go

ct-coverage-evmzero: tosca-cpp-coverage
ct-coverage-evmzero: 
	go run ./go/ct/driver run evmzero ; \
	echo "Coverage report generated in cpp/build/coverage/index.html"
	@cd cpp/build ; \
	cmake --build .  --target coverage 

test: test-go test-cpp test-rust

test-go: tosca-go
	@go test ./... -count 1 --coverprofile=cover.out

test-cpp: tosca-cpp
	@cd cpp/build ; \
	ctest --output-on-failure

test-rust:
	cargo test

test-cpp-asan: TOSCA_CPP_BUILD = Debug
test-cpp-asan: TOSCA_CPP_ASAN = ON
test-cpp-asan: test-cpp

cpp-coverage-report: 
	@cd cpp/build ; \
	cmake --build .  --target coverage 

test-cpp-coverage: TOSCA_CPP_BUILD = Debug
test-cpp-coverage: TOSCA_CPP_COVERAGE = ON
test-cpp-coverage: test-cpp
	@cd cpp/build ; \
	cmake --build .  --target coverage 

bench: TOSCA_CPP_ASSERT = OFF
bench: tosca-cpp bench-go

bench-go: TOSCA_CPP_ASSERT = OFF
bench-go:
	@go test -bench=. ./...

clean: clean-go clean-cpp clean-rust clean-evmone

clean-evmone:
	$(RM) -r ./third_party/evmone/build

clean-go:
	$(RM) -r ./go/build/*

clean-cpp:
	$(RM) -r ./cpp/build

clean-rust:
	cargo clean

license-headers:
	cd ./scripts/license; ./add_license_header.sh

fuzz-lfvm:
	go test -fuzz=FuzzLfvm ./go/ct/

fuzz-lfvm-diff:
	go test -fuzz=FuzzDifferentialLfvmVsGeth ./go/ct/

# TODO: disabbled until test is fixed #549
# fuzz-evmzero-diff:
# 	go test -fuzz=FuzzDifferentialEvmZeroVsGeth ./go/ct/

test-coverage: test-go coverage-report

coverage-report:
	@go install github.com/vladopajic/go-test-coverage/v2@v2.10.1
	@go-test-coverage --config .testcoverage.yml