#!/bin/bash

# https://clang.llvm.org/docs/UsersManual.html#profile-guided-optimization
# https://doc.rust-lang.org/rustc/codegen-options/index.html
# https://github.com/rust-lang/rust/commit/a17193dbb931ea0c8b66d82f640385bce8b4929a#diff-047672bc6da76afeeb5c06d57462ad63e3c3a052bb7716b35d4e617baf40a5faR25

# rustup component add llvm-tools-preview
# apt install bolt-18
# #cargo install cargo-pgo

if [ $(cat /proc/sys/kernel/perf_event_paranoid) -ne 0 ]; then
    echo 'running: "echo 0 | sudo tee /proc/sys/kernel/perf_event_paranoid"'
    echo 0 | sudo tee /proc/sys/kernel/perf_event_paranoid
fi

BENCH=0
# segfault with tail-call
#FEATURES=mimalloc,stack-array,custom-evmc,jumptable-dispatch,hash-cache,code-analysis-cache,fn-ptr-conversion-expanded-dispatch
FEATURES=performance
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
#RUSTFLAGS="-Cdebug--for-profiling -Cunique-internal-linkage-names" \
RUSTFLAGS="-Zdebug-info-for-profiling" \
    cargo +nightly build --release --features $FEATURES

perf record -b -e BR_INST_RETIRED.NEAR_TAKEN:uppp \
    go test ../go/integration_test/interpreter/... \
    --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
    --count $RUNS --timeout 1h

llvm-profgen-16 --binary=target/release/libevmrs.so --output=code.prof --perfdata=perf.data
#create_llvm_prof --binary=target/release/libevmrs.so --out=code.prof

#RUSTFLAGS="-Cdebug-info-for-profiling -Cunique-internal-linkage-names -Cprofile-sample-use=code.prof" \
RUSTFLAGS="-Zprofile-sample-use=$(pwd)/code.prof" \
    cargo +nightly build --release --features $FEATURES

# bench pgo optimized
if [ $BENCH -eq 1 ]; then
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