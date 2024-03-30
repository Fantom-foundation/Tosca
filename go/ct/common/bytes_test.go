package common

import (
	"encoding/json"
	"testing"
)

func TestBytes_EqualWhenContainingSameContent(t *testing.T) {
	b1 := NewBytes([]byte{1, 2, 3})
	b2 := NewBytes([]byte{1, 2, 3})
	b3 := NewBytes([]byte{3, 2, 1})

	if &b1 == &b2 {
		t.Fatalf("instances are not distinct, got %v and %v", &b1, &b2)
	}

	if b1 != b2 {
		t.Errorf("instances are not equal, got %v and %v", b1, b2)
	}

	if b1 == b3 {
		t.Errorf("instances are equal, got %v and %v", b1, b3)
	}
}

func TestBytes_AssignmentProducesEqualValue(t *testing.T) {
	b1 := NewBytes([]byte{1, 2, 3})
	b2 := b1

	if b1 != b2 {
		t.Errorf("assigned value is not equal: %v vs %v", b1, b2)
	}
}

func TestBytes_CanBeJsonEncoded(t *testing.T) {
	tests := map[string]struct {
		bytes   Bytes
		encoded string
	}{
		"empty": {
			NewBytes(nil), "\"\"",
		},
		"single": {
			NewBytes([]byte{1}), "\"01\"",
		},
		"triple": {
			NewBytes([]byte{1, 2, 3}), "\"010203\"",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(test.bytes)
			if err != nil {
				t.Fatalf("failed to encode into JSON: %v", err)
			}

			if want, got := test.encoded, string(encoded); want != got {
				t.Errorf("unexpected JSON encoding, wanted %v, got %v", want, got)
			}

			var restored Bytes
			if err := json.Unmarshal(encoded, &restored); err != nil {
				t.Fatalf("failed to restore bytes: %v", err)
			}
			if test.bytes != restored {
				t.Errorf("unexpected restored value, wanted %v, got %v", test.bytes, restored)
			}
		})
	}
}

func TestBytes_InvalidJsonFails(t *testing.T) {
	tests := map[string]string{
		"empty":               "",
		"missing start quote": "12\"",
		"missing end quote":   "\"12",
		"uneven length":       "\"123\"",
		"not hex":             "\"xy\"",
	}

	for name, encoded := range tests {
		t.Run(name, func(t *testing.T) {
			var restored Bytes
			err := json.Unmarshal([]byte(encoded), &restored)
			if err == nil {
				t.Errorf("decoding should have failed, but got %v", restored)
			}
		})
	}
}
