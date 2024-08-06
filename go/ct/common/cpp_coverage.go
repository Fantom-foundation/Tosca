// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package common

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../cpp/build/common/coverage -ltosca_collect_coverage
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../cpp/build/common/coverage

int IsCoverageEnabled();
void DumpCoverageData();
*/
import "C"

// isCppCoverageEnabled returns true if C++ has been compiled with coverage enabled.
// This assumes that every C++ library loaded at runtime for which coverage data should
// be collected has been compiled with coverage enabled.
func isCppCoverageEnabled() bool {
	return C.IsCoverageEnabled() != 0
}

// DumpCppCoverageData triggers the C++ code to dump coverage data.
// Not calling this function will result in no coverage data being reported
// for runtime loaded C and C++ libraries.
// If coverage data collection is not enabled, this function is a no-op.
func DumpCppCoverageData() { // coverage-ignore nothing to test
	C.DumpCoverageData()
}
