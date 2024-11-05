// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package rust

import (
	"flag"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

var stateImpl = flag.Bool("expect-coverage", false, "enable if the unit test is expecting a coverage build")

func TestDumpRustCoverageData(t *testing.T) {

	// write coverage data into tempDir directory
	tempDir := t.TempDir()
	llvmProfileFile := tempDir + "/rust-%p-%m.profraw"

	// run dump routine
	DumpRustCoverageData(llvmProfileFile)

	expectEnabled := *stateImpl
	enabled := isRustCoverageEnabled()

	if !enabled {
		if expectEnabled {
			t.Fatalf("Failed, rust coverage is not enabled")
		} else {
			t.Skip("Skipping test, rust coverage disabled and not expected to be enabled")
		}
	} else if !expectEnabled {
		t.Fatalf("Failed, rust coverage is enabled, but it was expected disabled")
	}

	// check that at least one file is generated
	found := false
	err := filepath.WalkDir(tempDir, func(s string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		found = found || strings.HasSuffix(s, ".profraw")
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk directory %s: %v", tempDir, err)
	}

	if !found {
		t.Fatalf("Failed, test generated no coverage data files")
	}
}
