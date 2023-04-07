# Tosca Build / Development Instructions

## Build Requirements

- Go toolchain, minimum version 1.19
    - Ubuntu/Debian package: `golang-go`
- C/C++ toolchain (+ standard library) supporting C++20, Clang >= 14 recommended
    - Ubuntu/Debian package: `clang`
    - Recommended: install `clang-format`, `clangd`, and `gdb` for development
- [Bazel](https://bazel.build/)
    - Install Bazelisk via Go¹:
      ```
      go install github.com/bazelbuild/bazelisk@latest
      ```
    - Create a symlink `bazel` pointing to the installed `bazelisk` binary:
      ```
      cd $HOME/go/bin
      ln -s bazelisk bazel
      ```
- [mockgen](https://github.com/golang/mock)
    - Install via Go:
      ```
      go install github.com/golang/mock/mockgen@v1.6.0
      ```

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
- Generate `compile_commands.json`: press `F1` → *Generate Compilation Database*

### Build / Run in VSCode

Some tasks are defined in [`task.json`](cpp/.vscode/tasks.json) to ease building and testing during development.
Tasks can be run via `F1` → *Tasks: Run …*

### Debug in VSCode

The debugging targets are defined in [`launch.json`](cpp/.vscode/launch.json).
`gdb` must be installed for this to work.

Right now, this is only used to debug unit tests.
With a unit test file open (e.g. `word_test.cc`), set a break point and press `F5`.

### Build / Run Manually

To build different configurations, invoke Bazel in the `cpp` subdirectory:

```bash
# Debug Configuration
bazel build -c dbg //...

# Address Sanitizer
bazel build -c dbg --config asan //...

# Optimized build
bazel build -c opt //...

# Run all tests
bazel test //...

# Run test in sub-directory
bazel test //common/...

# Run individual test or binary
bazel run //common:word_test
```

> Note: VSCode's multi-root workspace feature does not play nice with these extensions.
