#!/bin/bash

# Run Go interpreter benchmarks for evmzero, lfvm and geth in as well as for evmrs with different features in branch origin/evmrs-main.
# This script must be called from the parent directory of Tosca.

RUNS=20
TIMEOUT=1h
BRANCH="origin/evmrs-main"

DATE=$(date +'%Y-%m-%dT%H:%M')

BENCH_DIR=benches/$DATE
mkdir -p $BENCH_DIR

cd Tosca

git fetch --all
git checkout ${BRANCH}
GIT_REF=$(git show-ref --hash=7 ${BRANCH})

make

# bench all non Rust interpreters
INTERPRETERS=("evmzero" "lfvm" "geth")
for INTERPRETER in "${INTERPRETERS[@]}"; do
    BENCH_NAME=$(basename ${GIT_REF})-${INTERPRETER}-${RUNS}
    echo running ${BENCH_NAME}
    go test ./go/integration_test/interpreter \
        --run none --bench ^Benchmark[a-zA-Z]+/./${INTERPRETER}$ \
        --timeout $TIMEOUT --count $RUNS | tee ../${BENCH_DIR}/${BENCH_NAME}
    sed -i "s/${INTERPRETER}-//g" ../${BENCH_DIR}/${BENCH_NAME}
done

# bench evmrs with different features
INTERPRETER="evmrs"
FEATURES_LIST=("" "performance")
for FEATURES in "${FEATURES_LIST[@]}"; do
    cd rust
    cargo build --lib --release --features "$FEATURES"
    cd ..

    BENCH_NAME=$(basename ${GIT_REF})-${INTERPRETER}-${FEATURES}-${RUNS}
    echo running ${BENCH_NAME}
    go test ./go/integration_test/interpreter \
        --run none --bench ^Benchmark[a-zA-Z]+/./${INTERPRETER}$ \
        --timeout $TIMEOUT --count $RUNS | tee ../${BENCH_DIR}/${BENCH_NAME}
    sed -i "s/${INTERPRETER}-//g" ../${BENCH_DIR}/${BENCH_NAME}
done
