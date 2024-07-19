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

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestDumpCppCoverageData(t *testing.T) {

	testArguments := parseCustomArguments(os.Args)
	expectEnabled := slices.Contains(testArguments, "--expect-coverage")
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

	// write coverage data into tempDir directory
	tempDir := t.TempDir()
	os.Setenv("GCOV_PREFIX", tempDir)

	// run dump routine
	DumpCppCoverageData()

	// check that at least one file is generated
	found := false
	err := filepath.WalkDir(tempDir, func(s string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// .gcno are generated at compile time, with source code locations and other meta
		// .gcda are generated at runtime with the actual coverage
		found = strings.HasSuffix(s, ".gcda")
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk directory %s: %v", tempDir, err)
	}

	if !found {
		t.Fatalf("Failed, test generated no coverage data files")
	}
}

func TestParseCustomArguments(t *testing.T) {

	tests := map[string]struct {
		name     string
		args     []string
		expected []string
	}{
		"empty": {
			args:     []string{},
			expected: nil,
		},
		"No custom arguments": {
			args:     []string{"program", "arg1", "arg2"},
			expected: nil,
		},
		"Custom arguments present": {
			args:     []string{"program", "--", "custom1", "custom2"},
			expected: []string{"custom1", "custom2"},
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			result := parseCustomArguments(test.args)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Unexpected result. Got: %v, Expected: %v", result, test.expected)
			}
		})
	}
}

// parseCustomArguments is a helper function to parse custom arguments found
// after the "--" separator in the command line arguments.
func parseCustomArguments(args []string) []string {
	var afterDash []string
	foundDash := false

	for _, arg := range args {
		if foundDash {
			afterDash = append(afterDash, arg)
		} else if arg == "--" {
			foundDash = true
		}
	}
	return afterDash
}
