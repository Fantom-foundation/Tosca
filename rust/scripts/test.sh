#!/bin/bash

# Run Go tests and CT for evmrs on branch origin/evmrs-main.
# This script must be called from the parent directory of Tosca.

mkdir -p tests

cd Tosca
git fetch --all

BRANCH="origin/evmrs-main"

git checkout ${BRANCH}
make clean-rust
make tosca-rust

GIT_REF=$(git show-ref --hash=7 ${BRANCH})

export RUST_BACKTRACE=full
go test ./go/... | tee ../tests/ct-full-${GIT_REF}
go run ./go/ct/driver run --full-mode evmrs | tee ../tests/go-test-${GIT_REF}
