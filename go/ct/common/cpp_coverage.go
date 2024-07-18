package common

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../cpp/build/common/coverage -ltosca_collect_coverage
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../cpp/build/common/coverage

int IsCoverageEnabled();
void DumpCoverageData();
*/
import "C"

// isCppCoverageEnabled returns true if C++ has been compiled with coverage enabled.
// this assumes that every C++ library loaded at runtime has been compiled with coverage enabled.
func isCppCoverageEnabled() bool {
	return C.IsCoverageEnabled() != 0
}

// DumpCppCoverageData triggers the C++ code to dump coverage data.
// Not calling this function will result in no coverage data being reported
// for runtime loaded C and C++ libraries.
func DumpCppCoverageData() {
	C.DumpCoverageData()
}
