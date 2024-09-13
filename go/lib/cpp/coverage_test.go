// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package cpp

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var stateImpl = flag.Bool("expect-coverage", false, "enable if the unit test is expecting a coverage build")

func TestDumpCppCoverageData(t *testing.T) {

	// write coverage data into tempDir directory
	tempDir := t.TempDir()
	os.Setenv("GCOV_PREFIX", tempDir)

	// run dump routine
	DumpCppCoverageData()

	expectEnabled := *stateImpl
	enabled := isCppCoverageEnabled()

	if !enabled {
		if expectEnabled {
			t.Fatalf("Failed, cpp coverage is not enabled")
		} else {
			t.Skip("Skipping test, cpp coverage disabled and not expected to be enabled")
		}
	} else if !expectEnabled {
		t.Fatalf("Failed, cpp coverage is enabled, but it was expected disabled")
	}

	// check that at least one file is generated
	found := false
	err := filepath.WalkDir(tempDir, func(s string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// .gcno are generated at compile time, with source code locations and other meta
		// .gcda are generated at runtime with the actual coverage
		found = found || strings.HasSuffix(s, ".gcda")
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk directory %s: %v", tempDir, err)
	}

	if !found {
		t.Fatalf("Failed, test generated no coverage data files")
	}
}
