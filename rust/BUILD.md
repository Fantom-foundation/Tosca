# `evmrs` Development Instructions

## Requirements

- install Rust toolchain >= 1.81.0 from [here](https://rustup.rs/)

## Building evmrs (shared c library)

- build `evmrs` in release mode
    ```sh
    cargo build --release # OR run `make` or `make tosca-rust` in Tosca directory
    ```
- build `evmrs` with debug symbols
    ```sh
    cargo build --profile profiling
    ```

## Lint

```sh
cargo clippy --workspace --tests --benches --examples
```

## Doc

```sh
cargo doc --workspace --document-private-items --open
```

## Optimizations

By default, `evmrs` is build with the simplest and most idiomatic version.
All optimizations are behind feature flags.
A list of all features can be found in the `[features]` section in [Cargo.toml](./Cargo.toml).
For convenience there is also a feature named `performance` which enables all other features that improve overall performance.

Most `cargo` commands accept the `--features` flag followed by a list of features to enable, e.g.
```sh
cargo build --features mimalloc,stack-array
```

## Testing

- Rust tests
    ```sh
    cargo test
    ```
- Go tests
    ```sh
    cargo build --release # OR run `make` or `make tosca-rust` in Tosca directory
    go test ../go/...
    ```
- CT
    ```sh
    cargo build --release # TODO remove when https://github.com/Fantom-foundation/Tosca/pull/778 is merged
    cd ..
    make
    go run ./go/ct/driver run --full-mode evmrs
    ```

Also see [test.sh](./scripts/test.sh) which runs the Go tests and CT in full mode.

## Benchmarking

- Rust Benchmarks
    ```sh
    cargo bench --package benchmarks --release
    ```
- Go VM Benchmarks
    ```sh
    cargo build --release # or run make in Tosca directory
    go test ../go/integration_test/interpreter \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --timeout 1h --count 20
    ```

## Coverage

Code coverage can be generated for both the Rust test and CT.
- Rust test coverage
    ```sh
    cargo install cargo-llvm-cov
    cargo llvm-cov --open
    ```
- CT coverage: see [coverage.sh](./scripts/coverage.sh).

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

Also see [flamegraph.sh](./scripts/flamegraph.sh) which when provided with a list of features sets, runs the rust benchmarks and generates a flamegraph for each feature set.

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
