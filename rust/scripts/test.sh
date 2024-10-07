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

make -C .. clean-rust
make -C .. tosca-rust

export RUST_BACKTRACE=full
go test ../go/... | tee $OUTPUT_DIR/go-test
go run ../go/ct/driver run --full-mode evmrs | tee $OUTPUT_DIR/ct-full
