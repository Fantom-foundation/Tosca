#!/bin/bash

# Requirements:
# rustup component add llvm-tools-preview
# build bolt from source (see https://github.com/llvm/llvm-project/blob/main/bolt/README.md) and set BOLT accordingly

BENCH=1
# segfault with tail-call
FEATURES=performance
BENCH_RUNS=6
PROFILE_RUNS=3
RUST_LLVM_DIR=$(rustc --print sysroot)/lib/rustlib/x86_64-unknown-linux-gnu/bin
BOLT=~/Downloads/build/bin/llvm-bolt

if [ $(cat /proc/sys/kernel/perf_event_paranoid) -ne 0 ]; then
    echo 'running: "echo 0 | sudo tee /proc/sys/kernel/perf_event_paranoid"'
    echo 0 | sudo tee /proc/sys/kernel/perf_event_paranoid
fi

DATE=$(date +'%Y-%m-%dT%H:%M')
GIT_REF=$(git rev-parse --short=7 HEAD)
OUTPUT_DIR=output/benches/$DATE#$GIT_REF#$BENCH_RUNS#pgo
mkdir -p $OUTPUT_DIR

# benchmark without pgo
if [ $BENCH -eq 1 ]; then
    cargo clean
    cargo build --release --features $FEATURES
    taskset --cpu-list 0 \
        go test ../go/integration_test/interpreter/... \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --count $BENCH_RUNS --timeout 1h \
        | tee $OUTPUT_DIR/without-pgo
fi

# build & run pgo instrumented
CC=clang RUSTFLAGS="-Cprofile-generate=/tmp/pgo-data" \
    cargo build --release --features $FEATURES

rm -rf /tmp/pgo-data

go test ../go/integration_test/interpreter/... \
    --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
    --count $PROFILE_RUNS --timeout 1h

$RUST_LLVM_DIR/llvm-profdata merge -o /tmp/pgo-data/merged.profdata /tmp/pgo-data

# bench pgo optimized
if [ $BENCH -eq 1 ]; then
    CC=clang RUSTFLAGS="-Cprofile-use=/tmp/pgo-data/merged.profdata -Cllvm-args=-pgo-warn-missing-function" \
        cargo build --release --features $FEATURES
    taskset --cpu-list 0 \
        go test ../go/integration_test/interpreter/... \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --count $BENCH_RUNS --timeout 1h \
        | tee $OUTPUT_DIR/with-pgo
fi

## build & run pgo optimized bolt sampling
CC=clang CFLAGS="-fPIC" RUSTFLAGS="-Cprofile-use=/tmp/pgo-data/merged.profdata -Cllvm-args=-pgo-warn-missing-function -Clink-arg=-fuse-ld=lld -Crelocation-model=pic -C link-args=-Wl,--emit-relocs" \
    cargo build --release --features $FEATURES

perf record -e cycles:u -j any,u -a -o perf.data -- \
    go test ../go/integration_test/interpreter/... \
    --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
    --count $PROFILE_RUNS --timeout 1h

$BOLT target/release/libevmrs.so -o target/release/libevmrsopt.so -p=perf.data -reorder-blocks=ext-tsp -reorder-functions=hfsort -split-functions -split-all-cold -split-eh -dyno-stats

mv target/release/libevmrsopt.so target/release/libevmrs.so

## bench optimized pgo + bolt
if [ $BENCH -eq 1 ]; then
    taskset --cpu-list 0 \
        go test ../go/integration_test/interpreter/... \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --count $BENCH_RUNS --timeout 1h \
        | tee $OUTPUT_DIR/with-pgo-bolt
    cd $OUTPUT_DIR
    benchstat without-pgo with-pgo with-pgo-bolt | tee comparison
    cd -
fi
