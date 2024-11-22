# `evmrs` Development Instructions

## Requirements

- install Rust toolchain >= 1.81.0 from [here](https://rustup.rs/)

## Building `evmrs` (shared C library)

- build `evmrs` in release mode
    ```sh
    cargo build --lib --release # OR run `make` or `make tosca-rust` in Tosca directory
    ```
- build `evmrs` with debug symbols (the `profiling` profile inherits `release` and enables debug symbols)
    ```sh
    cargo build --lib --profile profiling
    ```

## Lint

To run the [Rust linter](https://doc.rust-lang.org/clippy/) on the whole project run:
```sh
cargo clippy --workspace --tests --benches --examples
```

## Documentation

To generate documentation an open it in a browser run:
```sh
cargo doc --workspace --document-private-items --open
```

## Testing

- Rust tests
    ```sh
    cargo test
    ```
- Go tests
    ```sh
    cargo build --lib --release # OR run `make` or `make tosca-rust` in Tosca directory
    go test ../go/interpreter/evmrs/...
    ```
- CT
    ```sh
    cd ..
    make
    go run ./go/ct/driver run --full-mode evmrs
    ```

Also see [run_ct.sh](./scripts/run_ct.sh) which runs the Go tests and CT.

## Coverage

To generate code coverage for the Rust tests using [cargo-llvm-cov](https://crates.io/crates/cargo-llvm-cov/0.1.13) run:
```sh
cargo install cargo-llvm-cov
cargo llvm-cov --open
```

To generate code coverage for CT see [coverage.sh](./scripts/coverage.sh).

## Optimizations

By default, `evmrs` is build with the simplest implementation.
All optimizations are behind feature flags.
A list of all features can be found in the `[features]` section in [Cargo.toml](./Cargo.toml).
For convenience there is also a feature named `performance` which enables all other features that improve overall performance.

Most `cargo` commands accept the `--features` flag followed by a list of features to enable, e.g.
```sh
cargo build --features mimalloc,stack-array
```

### Optimization Workflow

1. Identify a possible optimization opportunity by
    - running the Go VM benchmarks and comparing `evmrs` with other interpreters
        ```sh
        cargo run --package bencher
        ```
    - running a profiler of you choice and identifying a bottleneck
1. Add a feature in [Cargo.toml](Cargo.toml)
1. Implement the optimization and put it behind this new feature
1. Run [compare-features.sh](./scripts/compare-features.sh)
    ```sh
    ./compare-features.sh performance performance,my-new-feature
    ```
   This will run the Rust benchmarks and generate flamegraphs for all currently implemented features and for all currently implemented features and the new feature
1. Run Go VM Benchmarks 
    ```sh
    cargo run --package bencher -- --evmrs-only performance,my-new-feature
    ```
1. If all benchmarks indicate that the performance with the optimization is better that before, add the feature name to the features enabled by the `performance` feature in [Cargo.toml](Cargo.toml).

## Benchmarking

- Rust Benchmarks
    ```sh
    cargo bench --package benchmarks --release
    ```
- Go VM Benchmarks (for more information see [../BUILD.md](../BUILD.md#running-benchmarks))
    ```sh
    cargo build --release # or run make in Tosca directory
    go test ../go/integration_test/interpreter \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --timeout 1h --count 20
    ```
    To run benchmarks for evmzero, lfvm, geth and evmrs with different feature sets run:
    ```sh
    ./scripts/bench.sh <feature-set-1> <feature-set-2>
    ```

## Profiling

### Perf + Flamegraph

- Rust benchmarks
    ```sh
    # EITHER use cargo flamegraph
    cargo install cargo-flamegraph
    cargo flamegraph --package benchmarks --profile profiling

    # OR build benchmarks and then run perf manually
    cargo install inferno
    cargo build --package benchmarks --profile profiling
    perf record --call-graph dwarf -F 25000 ./target/profiling/benchmarks
    perf script | inferno-collapse-perf | inferno-flamegraph > flamegraph.svg
    ```
- Go VM benchmarks
    ```sh
    cargo install inferno
    cargo clean # make sure Go does not pick up the release build
    cargo build --profile profiling
    perf record --call-graph dwarf -F 25000 \
        go test ../go/integration_test/interpreter \
        --run none --bench '^Benchmark[a-zA-Z]+/./evmrs$' \
        --timeout 1h --count 1
    perf script | inferno-collapse-perf | inferno-flamegraph > flamegraph.svg
    ```

Also see [compare-features.sh](./scripts/compare-features.sh) which when provided with a list of features sets, runs the rust benchmarks and generates a flamegraph for each feature set.

### Perf + Firefox Profiler

```sh
cargo install --locked samply
# now run the same commands as for Perf + Flamegraph except for the last line with `perf script ...`
samply import perf.data # this converts perf.data, opens firefox profiler in your default browser and serves the data
```

### Samply + Firefox Profiler

```sh
cargo install --locked samply
cargo build --package benchmarks --profile profiling
samply record ./target/profiling/benchmarks # this collects profiling data, opens firefox profiler in your default browser and serves the data
```

### Intel VTune

```sh
cargo build --package benchmarks --profile profiling
```
run `./target/profiling/benchmarks` with Intel VTune 

## Miri & Sanitizers

### Miri

[Miri](https://github.com/rust-lang/miri) *is an Undefined Behavior detection tool for Rust*. It interprets Rust mid-intermediate representation as has therefore much more information available than when executing a binary. However, that makes it also relatively slow. Furthermore, it is only available on nightly Rust, which is not a big deal because it is only used for testing anyway.
*Because miri runs as a platform independent interpreter, it has no access to most platform-specific APIs or FFI.* It is therefore not possible to run it with a custom allocator an hence also not with feature *mimalloc* or *performance*.

```sh
# install miri
rustup +nightly component add miri

export MIRIFLAGS "-Zmiri-disable-stacked-borrows -Zmiri-permissive-provenance -Zmiri-backtrace=full"

# run tests with miri
cargo +nightly miri test
# run benchmark binary with miri
cargo +nightly miri run --package benchmarks -- 1 all-short
```

### Sanitizers

Rust supports various kinds of sanitizers. However, they are currently only available on nightly Rust. For more information see the [official documentation](https://doc.rust-lang.org/beta/unstable-book/compiler-flags/sanitizer.html).

It is recommended to pass `--target` and `-Zbuild-std` to cargo.
`--target` makes sure that the rustflags (for sanitizer instrumentation) are not applied to build scripts and procedural macros.
`-Zbuild-std` rebuilds the standard library with instrumentation.

Examples:
```sh
# required for -Zbuild-std
rustup +nightly component add rust-src

# run tests with address sanitizer
RUSTFLAGS=-Zsanitizer=address \
    cargo +nightly test -Zbuild-std --target x86_64-unknown-linux-gnu

# run tests with memory sanitizer
RUSTFLAGS="-Zsanitizer=memory -Zsanitizer-memory-track-origins" \
    cargo +nightly test -Zbuild-std --target x86_64-unknown-linux-gnu

# run tests with thread sanitizer
CFLAGS=-fsanitize=thread RUSTFLAGS=-Zsanitizer=thread \
    cargo +nightly test -Zbuild-std --target x86_64-unknown-linux-gnu

# run benchmarks with address sanitizer
RUSTFLAGS=-Zsanitizer=address \
    cargo +nightly run -Zbuild-std --target x86_64-unknown-linux-gnu --package benchmarks -- 1 all-short

# run benchmarks with memory sanitizer
RUSTFLAGS="-Zsanitizer=memory -Zsanitizer-memory-track-origins" \
    cargo +nightly run -Zbuild-std --target x86_64-unknown-linux-gnu --package benchmarks -- 1 all-short

# run benchmarks with thread sanitizer
CFLAGS=-fsanitize=thread RUSTFLAGS=-Zsanitizer=thread \
    cargo +nightly run -Zbuild-std --target x86_64-unknown-linux-gnu --package benchmarks -- 1 all-short
```

## Fuzzing

Fuzzing is done with [libfuzzer](https://llvm.org/docs/LibFuzzer.html) an *in-process, coverage-guided, evolutionary fuzzing engine*.
It is using the Rust binding [libfuzzer-sys](https://crates.io/crates/libfuzzer-sys) together with cargo integration [cargo fuzz](https://crates.io/crates/cargo-fuzz).

```sh
# install cargo integration
cargo install cargo-fuzz

# run fuzzer
cargo fuzz run --sanitizer none evmc_execute
# run fuzzer and stop after 10s
cargo fuzz run --sanitizer none evmc_execute -- -max_total_time=10
# run fuzzer with multiple jobs
cargo fuzz run --jobs <jobs> --sanitizer none evmc_execute
```
