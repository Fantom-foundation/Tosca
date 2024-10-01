# `evmrs` Development Instructions

## Requirements

- install Rust toolchain >= 1.81.0 from [here](https://rustup.rs/)

## Building `evmrs` (shared C library)

- build `evmrs` in release mode
    ```sh
    cargo build --release # OR run `make` or `make tosca-rust` in Tosca directory
    ```
- build `evmrs` with debug symbols (the `profiling` profile inherits `release` and enables debug symbols)
    ```sh
    cargo build --profile profiling
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
    cargo build --release # OR run `make` or `make tosca-rust` in Tosca directory
    go test ../go/interpreter/evmrs/...
    ```
- CT
    ```sh
    cd ..
    make
    go run ./go/ct/driver run --full-mode evmrs
    ```

Also see [test.sh](./scripts/test.sh) which runs the Go tests and CT in full mode.

## Coverage

To generate code coverage for the Rust tests using [cargo-llvm-cov](https://crates.io/crates/cargo-llvm-cov/0.1.13) run:
```sh
cargo install cargo-llvm-cov
cargo llvm-cov --open
```

To generate code coverage for CT see [coverage.sh](./scripts/coverage.sh).

## Optimizations

By default, `evmrs` is build with the simplest and most idiomatic implementation.
All optimizations are behind feature flags.
A list of all features can be found in the `[features]` section in [Cargo.toml](./Cargo.toml).
For convenience there is also a feature named `performance` which enables all other features that improve overall performance.

Most `cargo` commands accept the `--features` flag followed by a list of features to enable, e.g.
```sh
cargo build --features mimalloc,stack-array
```

### Optimization Workflow

1. Identify a possible optimization opportunity by
    - running the Go VM benchmarks and comparing in which cases `evmrs` is slower than the other interpreters
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
    cargo build --release --features performance
    go test ../go/integration_test/interpreter \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --timeout 1h --count 20
    cargo build --release --features performance,my-new-feature
    go test ../go/integration_test/interpreter \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --timeout 1h --count 20
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
    Note: 
    As mentioned in [../BUILD.md](../BUILD.md#diffing-benchmarks) you can use `benchstat` to compare different benchmark runs.
    However, the benchmarks names must match up.
    Because the benchmark names contain the name of the interpreter, you have to remove the interpreter name beforehand.
    ```sh
    sed -i "s/<interpreter 1 name>-//g" ./my-bench-output-1
    sed -i "s/<interpreter 2 name>-//g" ./my-bench-output-2
    ...
    benchstat ./my-bench-output-1 ./my-bench-output-2 ...
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

TODO

### Samply + Firefox Profiler

```sh
cargo install --locked samply
cargo build --package benchmarks --profile profiling
samply record ./target/profiling/benchmarks
```

### Intel VTune

```sh
cargo build --package benchmarks --profile profiling
```
run `./target/profiling/benchmarks` with Intel VTune 
