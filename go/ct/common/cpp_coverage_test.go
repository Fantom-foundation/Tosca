package common

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings" // Add this line to import the "strings" package
	"testing"
)

func TestDumpCppCoverageData(t *testing.T) {

	if !isCppCoverageEnabled() {
		t.Skip("C++ coverage is not enabled")
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
		found = strings.HasSuffix(s, ".gcda")
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk directory %s: %v", tempDir, err)
	}

	if !found {
		t.Fatalf("Failed to find gcda files")
	}
}
