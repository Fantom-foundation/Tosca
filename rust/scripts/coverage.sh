#!/bin/bash

# Run CT with evmrs and collect coverage data.
# This script must be called from Tosca/rust.

# Requirements:
# rustup component add llvm-tools-preview

INTERACTIVE=0

RUST_LLVM_DIR=$(rustc --print sysroot)/lib/rustlib/x86_64-unknown-linux-gnu/bin
CT_COV_TARGET=target/ct-cov-target
CT_COV=target/ct-cov
LLVM_COV_TARGET=target/llvm-cov-target
JOINT_COV_TARGET=target/joint-cov-target
JOINT_COV=target/joint-cov

cargo clean

mkdir -p $CT_COV_TARGET
mkdir -p $CT_COV

make -C ..
# build evmrs with coverage instrumentation
RUSTFLAGS="-C instrument-coverage" cargo build --release
# run CT
LLVM_PROFILE_FILE="$CT_COV_TARGET/rust-%p-%m.profraw" go run ../go/ct/driver run evmrs
$RUST_LLVM_DIR/llvm-profdata \
    merge \
    --sparse \
    --output $CT_COV_TARGET/rust.profdata \
    $CT_COV_TARGET/rust-*.profraw 
# generate html report
$RUST_LLVM_DIR/llvm-cov \
    show \
    ./target/release/libevmrs.so \
    --instr-profile=$CT_COV_TARGET/rust.profdata \
    --show-line-counts-or-regions \
    --output-dir=$CT_COV \
    --format=html \
    --ignore-filename-regex='/.cargo/|/rustc/'
if [ $INTERACTIVE -eq 1 ]; then
    open $CT_COV/index.html
else
    # generate text report
    $RUST_LLVM_DIR/llvm-cov \
        show \
        ./target/release/libevmrs.so \
        --instr-profile=$CT_COV_TARGET/rust.profdata \
        --show-line-counts-or-regions \
        --output-dir=$CT_COV \
        --format=text \
        --ignore-filename-regex='/.cargo/|/rustc/'
    cat $CT_COV/index.txt
fi

# run rust tests and generate html report
#cargo llvm-cov --open

# merge coverage of CT and rust tests
# this does not work yet - most likely because CT uses evmrs as a cdylib but cargo llvm-cov as a rlib
#mkdir -p $JOINT_COV_TARGET
#mkdir -p $JOINT_COV

#cp $CT_COV_TARGET/rust-*.profraw $JOINT_COV_TARGET
#cp $LLVM_COV_TARGET/rust-*.profraw $JOINT_COV_TARGET
#$RUST_LLVM_DIR/llvm-profdata \
    #merge \
    #--sparse \
    #--output $JOINT_COV_TARGET/rust.profdata \
    #$JOINT_COV_TARGET/rust-*.profraw
#$RUST_LLVM_DIR/llvm-cov \
    #show \
    #./target/release/libevmrs.so \
    #--instr-profile=$JOINT_COV_TARGET/rust.profdata \
    #--show-line-counts-or-regions \
    #--output-dir=$JOINT_COV \
    #--format=html \
    #--ignore-filename-regex='/.cargo/|/rustc/'
#open $JOINT_COV/index.html
