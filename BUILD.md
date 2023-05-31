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
