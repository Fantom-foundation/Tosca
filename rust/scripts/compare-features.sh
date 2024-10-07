#!/bin/bash

# Run Rust benchmarks and create flamegraphs with different feature sets.
# This script must be called from Tosca/rust with a list of features sets.

# Requirements:
# cargo install cargo-flamegraph

if [ $(cat /proc/sys/kernel/perf_event_paranoid) -ne 0 ]; then
    echo 'running: "echo 0 | sudo tee /proc/sys/kernel/perf_event_paranoid"'
    echo 0 | sudo tee /proc/sys/kernel/perf_event_paranoid
fi

DATE=$(date +'%Y-%m-%dT%H:%M')
GIT_REF=$(git show-ref --hash=7 $BRANCH)

OUTPUT_DIR=output/profiling/$DATE#$GIT_REF
mkdir -p $OUTPUT_DIR

INTERPRETER="evmrs"
for FEATURES in "$@"; do
    echo running with features: $FEATURES
    cargo bench --package benchmarks --profile profiling --features "$FEATURES" | tee $OUTPUT_DIR/features#$FEATURES.criterion

    FLAMEGRAPH_FILE=$(realpath ./$OUTPUT_DIR/features#$FEATURES.svg)
    cargo flamegraph \
        --freq 25000 \
        --deterministic \
        --flamechart \
        --package benchmarks \
        --profile profiling \
        --features "$FEATURES" \
        --output $FLAMEGRAPH_FILE

    firefox $FLAMEGRAPH_FILE
done
