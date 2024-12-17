# `evmrs` Development Instructions

## Requirements

- install Rust toolchain >= 1.83.0 from [here](https://rustup.rs/)

## Building `evmrs` (shared C library)

- build `evmrs` in release mode
    ```sh
    cargo build --lib --release --features performance # OR run `make` or `make tosca-rust` in Tosca directory
    ```
- build `evmrs` with debug symbols (the `profiling` profile inherits `release` and enables debug symbols)
    ```sh
    cargo build --lib --profile profiling --features performance
    ```

## Lint

To run the [Rust linter](https://doc.rust-lang.org/clippy/) on the whole project run:
```sh
cargo clippy --workspace --all-targets --features performance
```

## Documentation

To generate documentation an open it in a browser run:
```sh
cargo doc --workspace --document-private-items --open
```

## Testing

- Rust tests
    ```sh
    cargo test --features performance
    ```
- Go tests
    ```sh
    cargo build --lib --release --features performance
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
cargo llvm-cov --open --features performance
```

To generate code coverage for CT see [coverage.sh](./scripts/coverage.sh).

## Optimizations

By default, `evmrs` is build with the simplest implementation.
All optimizations are behind feature flags.
A list of all features can be found in the `[features]` section in [Cargo.toml](./Cargo.toml).
For convenience there is also a feature named `performance` which enables all other features that improve overall performance.

Most `cargo` commands accept the `--features` flag followed by a list of features to enable, e.g.
```sh
cargo build --features mimalloc,custom-evmc
```

## Benchmarking

- Rust Benchmarks
    - Run with criterion benchmark harness (provides statistics about execution times)
        ```sh
        cargo bench --package benchmarks --release --features performance
        ```
    - Run as normal executable (no statistics but also no overhead from any harness - better suited for profiling)
        ```sh
        cargo run --package benchmarks --release --features performance
        ```
- Go VM Benchmarks (for more information see [../BUILD.md](../BUILD.md#running-benchmarks))
    - Run with evmrs by hand.
    ```sh
    cargo build --release --features performance # or run make in Tosca directory
    go test ../go/integration_test/interpreter \
        --run none --bench ^Benchmark[a-zA-Z]+/./evmrs \
        --timeout 1h --count 20
    ```
    - Run benchmarks for evmzero, lfvm, geth and evmrs with different feature combinations compare times and store results in `./output/benches/<time>#<git ref>#<runs>`.
    ```sh
    cargo run --package bencher
    ```

## Profiling

> Note:
Unless you are profiling function dispatch, it might make sense to disable feature `tail-call`. 
Otherwise, stack traces get very long and stack overflows can occur.
This is currently the default but might change in the future again. Just make sure `tail-call` is not in the list of features enabled by feature `performance` in [Cargo.toml](Cargo.toml)

> Note: 
In the examples below, the rust benchmarks (`./target/profiling/benchmarks`) are always run with the parameters `10` (runs) and `fib2` (benchmark name). 
You can obviously choose other parameters.
Run `./target/profiling/benchmarks --help` to get a list of available benchmarks.

> Note:
For profiling it makes sense to use the `profiling` profile (which is the release profile with debug symbols).
When running the go benchmarks the link paths have to be adjusted in `../go/interpreter/evmrs/evmrs.go` and `../go/lib/rust/coverage.go` (replace `target/release` by `target/profiling`).

### Perf + Flamegraph

- Rust benchmarks
    ```sh
    # EITHER use cargo flamegraph
    cargo install cargo-flamegraph
    cargo flamegraph --package benchmarks --profile profiling --features performance

    # OR build benchmarks and then run perf manually
    cargo install inferno
    cargo build --package benchmarks --profile profiling --features performance
    perf record --call-graph dwarf -F 25000 ./target/profiling/benchmarks 10 fib20
    perf script | inferno-collapse-perf | inferno-flamegraph > flamegraph.svg
    ```
- Go VM benchmarks
    ```sh
    cargo install inferno
    cargo clean # make sure Go does not pick up the release build
    cargo build --profile profiling --features performance
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
cargo build --package benchmarks --profile profiling --features performance
samply record ./target/profiling/benchmarks 10 fib20 # this collects profiling data, opens firefox profiler in your default browser and serves the data
```

### Intel VTune

```sh
cargo build --package benchmarks --profile profiling --features performance
```
run `./target/profiling/benchmarks` with Intel VTune 

### DHAT

[DHAT](https://valgrind.org/docs/manual/dh-manual.html) is a *dynamic heap analysis tool*.
It can be used to investigate where, how much and how often memory gets allocated and how those allocations get used.

> Note:
DHAT does not work properly if feature mimalloc is enabled.
To disable mimalloc, just comment out `mimalloc` in the list of features enabled by feature `performance` in [Cargo.toml](Cargo.toml)

```sh
cargo build --package benchmarks --profile profiling --features performance
valgrind --tool=dhat ./target/profiling/benchmarks 10 fib20

# open DHAT viewer
firefox https://nnethercote.github.io/dh_view/dh_view.html
# OR
open file:///usr/libexec/valgrind/dh_view.html
# then load dhat.out.<pid>
```

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
