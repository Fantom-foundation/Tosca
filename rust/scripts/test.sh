#!/bin/bash

# Run Go tests and CT for evmrs on branch origin/evmrs-main.
# This script must be called from Tosca/rust.

BRANCH="origin/evmrs-main"

git fetch --all
git checkout $BRANCH

DATE=$(date +'%Y-%m-%dT%H:%M')
GIT_REF=$(git show-ref --hash=7 $BRANCH)

OUTPUT_DIR=output/tests/$DATE#$GIT_REF
mkdir -p $OUTPUT_DIR

export RUST_BACKTRACE=full

make -C .. 

cargo build --lib --release --no-default-features
go test ../go/... | tee $OUTPUT_DIR/go-test-no-default
go run ../go/ct/driver run evmrs | tee $OUTPUT_DIR/ct-full-no-default

cargo build --lib --release --features performance
go test ../go/... | tee $OUTPUT_DIR/go-test-performance
go run ../go/ct/driver run evmrs | tee $OUTPUT_DIR/ct-full-performance

cargo build --lib --release --all-features
go test ../go/... | tee $OUTPUT_DIR/go-test-all-features
go run ../go/ct/driver run evmrs | tee $OUTPUT_DIR/ct-full-all-features
