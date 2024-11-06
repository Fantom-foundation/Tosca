#!/bin/bash

# rustup component add llvm-tools-preview
# apt install bolt-18
# #cargo install cargo-pgo

BENCH=1
# segfault with tail-call
FEATURES=mimalloc,stack-array,custom-evmc,jumptable,hash-cache,code-analysis-cache,opcode-fn-ptr-conversion
RUNS=1
RUST_LLVM_DIR=$(rustc --print sysroot)/lib/rustlib/x86_64-unknown-linux-gnu/bin

DATE=$(date +'%Y-%m-%dT%H:%M')
GIT_REF=$(git rev-parse --short=7 HEAD)
OUTPUT_DIR=output/benches/$DATE#$GIT_REF#$RUNS#pgo
mkdir -p $OUTPUT_DIR

# benchmark without pgo
if [ $BENCH -eq 1 ]; then
    cargo clean
    cargo build --release --features $FEATURES
    taskset --cpu-list 0 \
        go test ../go/integration_test/interpreter/... \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --count $RUNS --timeout 1h \
        | tee $OUTPUT_DIR/without-pgo
fi

# build & run pgo instrumented
cargo clean
RUSTFLAGS="-Cprofile-generate=/tmp/pgo-data" \
    cargo build --release --features $FEATURES

rm -rf /tmp/pgo-data

go test ../go/integration_test/interpreter/... \
    --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
    --count $RUNS --timeout 1h

$RUST_LLVM_DIR/llvm-profdata merge -o /tmp/pgo-data/merged.profdata /tmp/pgo-data

# bench pgo optimized
if [ $BENCH -eq 1 ]; then
    RUSTFLAGS="-Cprofile-use=/tmp/pgo-data/merged.profdata -Cllvm-args=-pgo-warn-missing-function" \
        cargo build --release --features $FEATURES
    taskset --cpu-list 0 \
        go test ../go/integration_test/interpreter/... \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --count $RUNS --timeout 1h \
        | tee $OUTPUT_DIR/with-pgo
    cd $OUTPUT_DIR
    benchstat without-pgo with-pgo | tee comparison
    cd -
fi

# .cargo/config.toml
#[target.x86_64-unknown-linux-gnu]
#rustflags = ["-C", "link-arg=-fuse-ld=lld"]

## build & run pgo optimized bolt instrumented
#RUSTFLAGS="-Cprofile-use=/tmp/pgo-data/merged.profdata -Cllvm-args=-pgo-warn-missing-function -C link-args=-Wl,-q" \
    #cargo build --release --features $FEATURES

#llvm-bolt-18 target/release/libevmrs.so -o target/release/libevmrs.so -instrument
#merge-fdata-18 /tmp/*.fdata > merged.profdata

##perf record -e cycles:u -j any,u -a -o perf.data -- \
    ##go test ../go/integration_test/interpreter/... \
    ##--run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
    ##--count $RUNS --timeout 1h
##perf2bolt -p perf.data -o perf.fdata target/release/libevmrs.so

#llvm-bolt-18 target/release/libevmrs.so -o target/release/libevmrs.so -data=perf.fdata -reorder-blocks=ext-tsp -reorder-functions=hfsort -split-functions -split-all-cold -split-eh -dyno-stats

## bench optimized pgo + bolt
#if [ $BENCH -eq 1 ]; then
    #taskset --cpu-list 0 \
        #go test ../go/integration_test/interpreter/... \
        #--run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        #--count $RUNS --timeout 1h \
        #| tee with-pgo-bolt
    #benchstat without-pgo with-pgo with-pgo-bolt
#fi

# with cargo-pgo (does not work)
#cargo pgo instrument build -- --features $FEATURES
#mkdir -p target/release
#cp target/x86_64-unknown-linux-gnu/release/libevmrs.so target/release/
#go test ../go/integration_test/interpreter/... --run none --bench ^Benchmark[a-zA-Z]+/./evmrs --count 1
#$RUST_LLVM_DIR/llvm-profdata merge -o /tmp/pgo-data/merged.profdata /tmp/pgo-data
#cargo pgo optimize build -- --features $FEATURES