
#if defined(TOSCA_COVERAGE)
extern void __gcov_dump();
#endif

/// Reports whenever the library (and therefore the complete c++ project)
/// was compiled with coverage flags.
int IsCoverageEnabled() {
#if defined(TOSCA_COVERAGE)
  return 1;
#else
  return 0;
#endif
}

/// When using Gcov, Shared libraries do collect coverage data,
/// but it is not automatically written into a file. Usually, GCC would
/// add a corresponding call to the end of a `main` function, but it
/// can not do this for a Go application. Thus, the dumping of
/// coverage data of C++ library code needs to be triggered explicitly
/// by calling this function at the end of an application.
/// Calling this function will dump coverage data for all loaded C++ libraries
/// compiled with coverage flags. If coverage data collection is disabled,
/// this function is a no-op.
void DumpCoverageData() {
#if defined(TOSCA_COVERAGE)
  // Since Gcc 11, before __gcov_flush()
  __gcov_dump();
#endif
}
