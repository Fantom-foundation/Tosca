# Tosca Build / Development Instructions

Be sure to initialize all submodules

```
git submodule update --init --recursive
```

## Build Requirements

- Go toolchain, minimum version 1.19
    - Ubuntu/Debian package: `golang-go`
- C/C++ toolchain (+ standard library) supporting C++20, Clang >= 14 recommended
    - Ubuntu/Debian package: `clang`
    - Recommended: install `clang-format`, `clangd`, and `gdb` for development
- [mockgen](https://github.com/golang/mock)
    - Install via Go:
      ```
      go install github.com/golang/mock/mockgen@v1.6.0
      ```
- [CMake](https://cmake.org/)
    - Ubuntu/Debian package: `cmake`

¹ Go installs programs into `$GOPATH/bin`, where `GOPATH` defaults to `$HOME/go`.
  Add this `bin` directory to your `PATH`.

## Building

Use the provided Makefile to build and test the project.

```
make
make test
```

## Development Experience C++

Open the `cpp` subdirectory in VSCode:
- Install recommended extensions: press `F1` → *Show Recommended Extensions*

### Build / Run in VSCode

Select the wanted compiler (i.e. kit), build configuration and build target in VSCode's status line.
The selected target can be build via the *Build* button in the status line, or by pressing `F7`.

### Debug in VSCode

Pressing the 🐜 button in VSCode's status line launches the currently selected target in the debugger.
The same can be achieved by pressing `Ctrl + F5`.

### Build / Run Manually

To build different configurations, invoke CMake in the `cpp` subdirectory:

```bash
# Debug Configuration (with AddressSanitizer)
cmake -Bbuild -DCMAKE_BUILD_TYPE=Debug -DTOSCA_ASAN=ON
cmake --build build --parallel

# Release Configuration
cmake -BBuild -DCMAKE_BUILD_TYPE=Release -DTOSCA_ASAN=OFF
cmake --build build --parallel

# Run all tests
ctest --test-dir build --output-on-failure

# Run individual test
ctest --test-dir build --output-on-failure -R <test-name>
```

`ctest` executes each unit test in isolation which is relatively slow to execute.
Alternatively, you can just run the corresponding unit test binary.
For instance:

```bash
# Run specific test binary directly
./build/vm/evmzero/uint256_test

# Build and run specific test binary directly
cmake --build build --parallel --target uint256_test && ./build/vm/evmzero/uint256_test
```

> Note: Invoking ctest does **not** trigger compilation.
> You have to invoke the build process beforehand.
> 
> ```bash
> cmake --build build --parallel && ctest --test-dir build --output-on-failure
> ```

> Note: For OSX, add the following argument to the initial CMake invocation:
> ```
> -DCMAKE_SHARED_LIBRARY_SUFFIX_CXX=.so
> ```

> Note: VSCode's multi-root workspace feature does not play nice with these extensions.

## Running Benchmarks

The Tosca project includes a set of benchmarks covering a range of EVM implementations and variants thereof. The benchmarks are part of the `github.com/Fantom-foundation/Tosca/go/vm/test` module.

The benchmarks are implemented using Go's benchmark infrastructure. For a comprehensive introduction see the corresponding chapter in [Practical Go Lessons](https://www.practical-go-lessons.com/chap-34-benchmarks) or the [Go Test Package Documentation](https://pkg.go.dev/testing#hdr-Benchmarks).

To run all benchmarks, use the following command:
```
go test ./go/vm/test -run=NONE -bench=.
```
The path `./go/vm/test` points to the Go package containing the benchmarks. The flag `-run=NONE` disables the execution of unit tests, and `-bench=<regex>` enables the execution of benchmarks.

Optionally, to include memory usage metrics, the `-benchmem` flag can be added:
```
go test ./go/vm/test -run=NONE -bench=. -benchmem
```

The command produces benchmark results in a table format:
```
pkg: github.com/Fantom-foundation/Tosca/go/vm/test
cpu: Intel(R) Core(TM) i7-5820K CPU @ 3.30GHz
BenchmarkInc/1/geth-12  	  121488	     10529 ns/op	    1768 B/op	      16 allocs/op
BenchmarkInc/1/lfvm-12  	  124447	      8744 ns/op	    4330 B/op	      17 allocs/op
BenchmarkInc/1/lfvm-si-12         	  140642	      8472 ns/op	    4330 B/op	      17 allocs/op
BenchmarkInc/1/lfvm-no-sha-cache-12         	  132420	      9165 ns/op	    4330 B/op	      17 allocs/op
BenchmarkInc/1/lfvm-no-code-cache-12        	  137041	      8330 ns/op	    4330 B/op	      17 allocs/op
BenchmarkInc/1/evmone-12                    	  193581	      6250 ns/op	    1088 B/op	      15 allocs/op
BenchmarkInc/1/evmone-basic-12              	  186457	      6330 ns/op	    1088 B/op	      15 allocs/op
BenchmarkInc/1/evmone-advanced-12           	  136111	      7381 ns/op	    1088 B/op	      15 allocs/op
BenchmarkInc/1/evmzero-12                   	  162552	      7317 ns/op	    1088 B/op	      15 allocs/op
```
The columns are as follows:
 - the name of the benchmark, using `/` as a separator between benchmarks and sub-benchmarks. In Tosca's setup, the first part of the name is the name of the benchmark (e.g. `BenchmarkInc`), the second part the input size (e.g. `10`), and the last part the EVM implementation (e.g. `evmone-basic`) followed by the number of cores used to run the benchmark (e.g. `-12`); the latter is added by Go's benchmark infrastructure and has no effect if the benchmark is not using multiple go routines.
 - the number of times the benchmark was executed
 - the average time per execution
 - the average number of bytes allocated per execution (only when using `-benchmem`)
 - the average number of allocations per execution (only when using `-benchmem`)

### Filtering Benchmarks

The `-bench` flag can be used to filter benchmarks by their name using regex expressions. It's important to note that:
 - tests are selected if the regex matches any part of the benchmark name
 - before the regex expressions are applied, the pattern is devided into sub-patterns for sub-tests using the `/` separator

Thus, the command
```
go test ./go/vm/test -run=NONE -bench=Inc/1/zero
```
is running the benchmarks
```
BenchmarkInc/1/evmzero-12
BenchmarkInc/10/evmzero-12
```
since the individual parts of the benchmark name are matched one-by-one.

### Profiling Benchmarks
Go has an integrated CPU profiler that can be enabled using the `-cpuprofile` flag:
```
go test ./go/vm/test -run=NONE -bench Fib/20/evmzero -cpuprofile cpu.log
```
This command runs the benchmark and collects CPU performance data which can be visualized using
```
go tool pprof -http "localhost:8000" ./cpu.log
```
which starts a web-server at port `8000` hosting an interactive set of charts to investigate the collected profiling data.

If symbols for the C++ libraries are missing, make sure the library is build including those symbols and that in some go file in the `./go/vm/test` package the following import is present:
```
_ "github.com/ianlancetaylor/cgosymbolizer"
```
This package makes C/C++ symbols accessible for the Go profiler. On some systems, however, this may hide Go symbols so the import may have to be added/removed as needed.

### Diffing Benchmarks

To compare the benchmark results of two different code versions, the [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) tool can be utilized. The tool produces reports on the impact of code changes on the performance of benchmarks in the following format:
```
name                          old time/op  new time/op  delta
Fib/20/lfvm-12                21.6ms ± 8%  21.1ms ± 2%    ~     (p=0.417 n=10+7)
Fib/20/lfvm-si-12             21.7ms ± 3%  21.1ms ± 2%  -2.66%  (p=0.004 n=9+8)
Fib/20/lfvm-no-sha-cache-12   21.4ms ± 6%  21.0ms ± 1%    ~     (p=0.165 n=10+10)
Fib/20/lfvm-no-code-cache-12  21.2ms ± 3%  20.7ms ± 3%  -2.17%  (p=0.006 n=9+10)
```
For details on how to use it, please refer to the [Example](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat#hdr-Example) section of the tool's documentation page.

Ideally, pull requests targeting performance improvements include such a report in their description to document its impact.