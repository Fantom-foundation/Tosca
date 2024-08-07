# Code Coverage

The Tosca project collects coverage data using different technologies for the C++ and Go code bases. 

## Coverage in Go

The official [documentation](https://go.dev/doc/build-cover) explains how to use it.
Both Unit tests and CT driver can produce coverage reports.

By default Go coverage will only include the source of included files, not dependencies. 
The argument `-coverpkg=./go/...,github.com/project/dependency-name/` will include coverage reports from included code.

## Coverage in C++

C and C++ can be instrumented for coverage reports using both Gcc and Clang. However, the actual procedure to do so is slightly different:
- Gcc uses [gcov](https://gcc.gnu.org/onlinedocs/gcc/Gcov.html); there is an ecosystem of tools around gcov, to filter, modify and produce reports: [lcov](https://wiki.documentfoundation.org/Development/Lcov)
- [Clang](https://clang.llvm.org/docs/SourceBasedCodeCoverage.html) implements its [own coverage technology](https://clang.llvm.org/docs/SanitizerCoverage.html), but in theory they provide support for gcov. This later follows a different paradigm, as the coverage engine will only record data for the module indicated at runtime. 

Tosca uses *gcov* to collect coverage data from the C++ tests.
At the time this document was written, Clang compilation support for Gcov would require some extra work at the compilation level.

By default all available source files collect coverage data. Third party and system headers coverage is filtered out after collection using lcov.
```bash
lcov --remove coverage.info /usr/include/* /usr/lib/*' */third_party/* --output-file filtered.info
```
- Text report is generated and can be found in `cpp/build/coverage.txt`
- Html report is generated and can be found in `cpp/build/coverage/index.html` or compressed as `cpp/build/coverage.tar.gz`


## Collect C++ coverage when running CT evmzero 

Gcov collects coverage data during runtime and only at the end of the process it is saved into a file. This mechanism would be called automatically on the C++ main process, but has to be called manually when using evmzero from a Go process.
It is possible to call the Gcov dump routine in runtime, by calling __gcov_dump. A mechanism to invoke this routine from Go has been implemented, so that the CT can generate the required data files. 

To retrieve the coverage report execute the following commands:
```bash
make tosca-cpp-coverage
go run ./go/ct/driver run evmzero
make cpp-coverage-report
```

When replacing `make tosca-cpp-coverage` with `make cpp-test-coverage`, unit test coverage will be combined into the report.

Results can be found in the same folder as described before.

