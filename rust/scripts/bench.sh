#!/bin/bash

# Run Go interpreter benchmarks for evmzero, lfvm and geth as well as for evmrs with different feature sets on branch origin/evmrs-main.
# This script must be called from the parent directory of Tosca with a list of feature sets.

# Requirements:
# go install golang.org/x/perf/cmd/benchstat@latest

if [ $# -eq 0 ]; then
    echo "Usage: $0 [--evmrs-only] <feature-set> [<feature-set> ...]"
    exit 1
fi

EVMRS_ONLY=0

if [ "$1" == "--evmrs-only" ]; then
    EVMRS_ONLY=1
    shift

    if [ $# -eq 0 ]; then
        echo "Usage: $0 [--evmrs-only] <feature-set> [<feature-set> ...]"
        exit 1
    fi
fi

RUNS=20
TIMEOUT=1h
BRANCH="origin/evmrs-main"

cd Tosca

git fetch --all
git checkout $BRANCH

DATE=$(date +'%Y-%m-%dT%H:%M')
GIT_REF=$(git show-ref --hash=7 $BRANCH)

OUTPUT_DIR=../benches/$DATE#$GIT_REF#$RUNS
mkdir -p $OUTPUT_DIR

if [ ! $EVMRS_ONLY ]; then
    make

    INTERPRETERS=("evmzero" "lfvm" "geth")
    for INTERPRETER in "${INTERPRETERS[@]}"; do
        echo running $INTERPRETER

        OUTPUT_FILE=$OUTPUT_DIR/$INTERPRETER
        go test ./go/integration_test/interpreter \
            --run none --bench ^Benchmark[a-zA-Z]+/./$INTERPRETER$ \
            --timeout $TIMEOUT --count $RUNS | tee $OUTPUT_FILE
        sed -i "s/$INTERPRETER-//g" $OUTPUT_FILE
    done
fi

INTERPRETER="evmrs"
for FEATURES in "$@"; do
    echo running $INTERPRETER with features: $FEATURES

    cd rust
    cargo build --lib --release --features "$FEATURES"
    cd ..

    OUTPUT_FILE=$OUTPUT_DIR/$INTERPRETER-$FEATURES
    go test ./go/integration_test/interpreter \
        --run none --bench ^Benchmark[a-zA-Z]+/./$INTERPRETER$ \
        --timeout $TIMEOUT --count $RUNS | tee $OUTPUT_FILE
    sed -i "s/$INTERPRETER-//g" $OUTPUT_FILE
done

cd $OUTPUT_DIR
benchstat * | tee comparison
