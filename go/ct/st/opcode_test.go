package st

import (
	"regexp"
	"testing"
)

func TestOpCode_CanBePrinted(t *testing.T) {
	validName := regexp.MustCompile(`^\?|([A-Z0-9]+)$`)
	for i := 0; i < 256; i++ {
		print := OpCode(i).String()
		if !validName.Match([]byte(print)) {
			t.Errorf("Invalid print for op %v (%d)", print, i)
		}
	}
}
