#!/bin/bash

# Run Go interpreter benchmarks for evmrs, evmzero, lfvm and geth in branch origin/main and evmrs in branch origin/evmrs-main.
# This script must be called from the parent directory of Tosca.

mkdir -p benches

cd Tosca
git fetch --all

RUNS=20
TIMEOUT=1h
REF_BRANCH="origin/main"
NEW_BRANCH="origin/evmrs-main"

BRANCH=$REF_BRANCH
git checkout ${BRANCH}
make clean-rust
make

GIT_REF=$(git show-ref --hash=7 ${BRANCH})
INTERPRETERS=("evmrs" "evmzero" "lfvm" "geth")
for INTERPRETER in "${INTERPRETERS[@]}"; do
    BENCH_NAME=$(basename ${BRANCH})-${GIT_REF}-${INTERPRETER}-${RUNS}
    echo running ${BENCH_NAME}
    go test ./go/integration_test/interpreter \
        --run none -bench ^Benchmark[a-zA-Z]+/./${INTERPRETER}$ \
        -timeout $TIMEOUT -count $RUNS | tee ../benches/${BENCH_NAME}
    sed -i "s/${INTERPRETER}-//g" ../benches/${BENCH_NAME}
done

BRANCH=$NEW_BRANCH
git checkout ${BRANCH}
make clean-rust
make tosca-rust

GIT_REF=$(git show-ref --hash=7 ${BRANCH})
INTERPRETER="evmrs"
BENCH_NAME=$(basename ${BRANCH})-${GIT_REF}-${INTERPRETER}-${RUNS}
echo running ${BENCH_NAME}
go test ./go/integration_test/interpreter \
    --run none -bench ^Benchmark[a-zA-Z]+/./${INTERPRETER}$ \
    -timeout $TIMEOUT -count $RUNS | tee ../benches/${BENCH_NAME}
sed -i "s/${INTERPRETER}-//g" ../benches/${BENCH_NAME}
