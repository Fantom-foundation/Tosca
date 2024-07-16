# Tosca Build / Development Instructions

Be sure to initialize all submodules

```sh
git submodule update --init --recursive
```

## Build Requirements

- Go toolchain, minimum version 1.21
    - Ubuntu/Debian package: `golang-go`
    - Snap package: `go`
- C/C++ toolchain, Clang >= 16 or Gcc >= 11.4
    - Ubuntu/Debian package: `clang`
    - Recommended: install `clang-format`, `clangd`, and `gdb` for development
- [mockgen](https://github.com/golang/mock)
    - Install via Go:
      ```sh
      go install github.com/golang/mock/mockgen@v1.6.0
      ```
- [CMake](https://cmake.org/)
    - Ubuntu/Debian package: `cmake`
    - Snap package: `cmake`

### Go Setup Remarks

Note that in some package managers (e.g. apt) the default Go package is an outdated version which is not sufficient for Tosca. Newer versions can be installed by explicitly specifying the Go version, such as `golang-1.21`. Check your currently installed version with the command `go version`. Depending on your installation process it might be required to set `GOROOT`, this can be done in your `.bashrc`.

If no packages are available, follow the instructions on the official [go website](https://go.dev/doc/install) to download and install the newest version.

Go installs programs into `$GOPATH/bin`, where `GOPATH` defaults to `$HOME/go`, add this `bin` directory to your `PATH`. 

## Building

Use the provided Makefile to build and test the project.

```sh
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

```sh
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

```sh
# Run specific test binary directly
./build/vm/evmzero/uint256_test

# Build and run specific test binary directly
cmake --build build --parallel --target uint256_test && ./build/vm/evmzero/uint256_test
```

> Note: Invoking ctest does **not** trigger compilation.
> You have to invoke the build process beforehand.
> 
> ```sh
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

```sh
go test ./go/vm/test -run=NONE -bench=.
```

The path `./go/vm/test` points to the Go package containing the benchmarks. The flag `-run=NONE` disables the execution of unit tests, and `-bench=<regex>` enables the execution of benchmarks.

Optionally, to include memory usage metrics, the `-benchmem` flag can be added:

```sh
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

```sh
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

```sh
go test ./go/vm/test -run=NONE -bench Fib/20/evmzero -cpuprofile cpu.log
```

This command runs the benchmark and collects CPU performance data which can be shown in tabular form using

```sh
go tool pprof -text -nodecount=10 cpu.log
```

or can be visualized by the following command (requires the graphviz libary)

```sh
go tool pprof -http "localhost:8000" ./cpu.log
```

which starts a web-server at port `8000` hosting an interactive set of charts to investigate the collected profiling data.

If symbols for the C++ libraries are missing, make sure the library is built including those symbols and that in some go file in the `./go/vm/test` package the following import is present:

```
_ "github.com/ianlancetaylor/cgosymbolizer"
```

This package makes C/C++ symbols accessible to the Go profiler. On some systems, however, this may hide Go symbols so the import may have to be added/removed as needed.

### Diffing Benchmarks

To compare the benchmark results of two different code versions, the [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) tool can be utilized. To install the tool run

```sh
go install golang.org/x/perf/cmd/benchstat@latest
```

which places the binary `benchstat` into your `$GOPATH/bin` directory.

The tool produces reports on the impact of code changes on the performance of benchmarks in the following format:

```
name                          old time/op  new time/op  delta
Fib/20/lfvm-12                21.6ms ± 8%  21.1ms ± 2%    ~     (p=0.417 n=10+7)
Fib/20/lfvm-si-12             21.7ms ± 3%  21.1ms ± 2%  -2.66%  (p=0.004 n=9+8)
Fib/20/lfvm-no-sha-cache-12   21.4ms ± 6%  21.0ms ± 1%    ~     (p=0.165 n=10+10)
Fib/20/lfvm-no-code-cache-12  21.2ms ± 3%  20.7ms ± 3%  -2.17%  (p=0.006 n=9+10)
```

For details on how to use it, please refer to the [Example](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat#hdr-Example) section of the tool's documentation page.

Ideally, pull requests targeting performance improvements should include such a report in their description to document its impact.

## Code Coverage

The Tosca project allows to collect coverage reports for unit tests and CT runs. 
More about coverage in [the docs folder](docs/coverage.md)

### CT Driver Code coverage

To run the the CT driver code coverage simply run `make ct-coverage-lfvm` or `make ct-coverage-geth`, these commands will create the folder `Tosca/go/build/coverage/` and a sub folder with the corresponding interpreter's name. In this directory an instrumented version of the driver will be built and then run. The reports of the coverage of this run will also be in this directory, where lastly an HTML version of the report will be added as well. As more interpreter implementations are added to the Tosca project, they should also be added to the makefile.


### C++ Code Unit Test Coverage

Current code coverage infrastructure uses GNU gcov system. This is supported in both Gcc and Clang, although Clang target gcov version may not be the installed one. For this reason the current infrastructure is currently enabled for Gcc only. 

Code coverage depends on the following tools:
- lcov
- genhtml
Both commands can be installed in Ubuntu with the command: `apt install lcov`

Code coverage can be triggered by using the following command:
```sh
make clean # required if already configured using another compiler Toolchain
CC=gcc CXX=g++ make test-cpp-coverage
```
The project will compile an instrumented debug version, to run the C++ unit tests, and to finally print a coverage report of the C++ code. Such report looks something like:
```
...
Processing file vm/evmzero/stack_test.cc
  lines=57 hit=57 functions=28 hit=28
Overall coverage rate:
  lines......: 85.8% (3380 of 3938 lines)
  functions......: 82.8% (2057 of 2483 functions)
```

The report will be generated in HTML form to alow visualization of each line of code coverage for each C++ file in the project. The entry HTML page to the report is located at the C++ build folder: `cpp/build/coverage/index.html`

In VSCode, line by line coverage can be visualized using the extension [gcov-viewer](https://marketplace.visualstudio.com/items?itemName=JacquesLucke.gcov-viewer)


## Fuzzing

Go provides fuzzing support through its standard library. Fuzzing attempts to find bugs by mutating a test input data set and measuring test coverage. This complementary technique aims to identify stability issues.			 

### Fuzzing Evm.StepN interface

The ct Evm.StepN interface is used to evaluate N instructions in different EVM implementations. There are 2 Fuzzer tests against this interface:

- Crash test: FuzzLfvm will execute instructions one at the time looking for a panic. ```make fuzz-lfvm```
- Differential tests; execute instruction in two VM implementations and compare state results in addition to panic.
   - FuzzDifferentialLfvmVsGeth: ```make fuzz-lfvm-diff```
   - FuzzDifferentialEvmzeroVsGeth: ```make fuzz-evmzero-diff``` (disabled, issue #549)
