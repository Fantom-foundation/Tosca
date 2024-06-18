// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vm

import (
	"encoding/json"
	"testing"
)

func TestAddress_NewAddress(t *testing.T) {
	address := Address{}

	if address != [20]byte{} {
		t.Errorf("New address must be default value.")
	}
}

func TestAddress_JSON_Encoding(t *testing.T) {
	tests := []struct {
		address Address
		json    string
	}{
		{Address{}, "\"0x0000000000000000000000000000000000000000\""},
		{Address{1}, "\"0x0100000000000000000000000000000000000000\""},
		{Address{0xAB}, "\"0xab00000000000000000000000000000000000000\""},
		{
			Address{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			"\"0x000102030405060708090a0b0c0d0e0f10111213\"",
		},
	}

	for _, test := range tests {
		encoded, err := json.Marshal(test.address)
		if err != nil {
			t.Fatalf("failed to encode into JSON: %v", err)
		}

		if want, got := test.json, string(encoded); want != got {
			t.Errorf("unexpected JSON encoding, wanted %v, got %v", want, got)
		}

		var restored Address
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatalf("failed to restore address: %v", err)
		}
		if test.address != restored {
			t.Errorf("unexpected restored value, wanted %v, got %v", test.address, restored)
		}

	}
}

func TestAddress_JSON_InvalidValueDecodingFails(t *testing.T) {
	tests := map[string]string{
		"empty":                 "\"\"",
		"empty with hex prefix": "\"0x\"",
		"no hex prefix":         "\"0000000000000000000000000000000000000000\"",
		"too short":             "\"0x00000000000000000000000000000000000000\"",
		"just too short":        "\"0x000102030405060708090a0b0c0d0e0f1011121\"",
		"just too long":         "\"0x000102030405060708090a0b0c0d0e0f101112131\"",
		"too long":              "\"0x000000000000000000000000000000000000000000\"",
		"invalid hex":           "\"0x0g00000000000000000000000000000000000000\"",
		"not hex":               "\"hello, this is a test with 20 characters\"",
		"not a JSON string":     "0x000102030405060708090a0b0c0d0e0f10111213",
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			var address Address
			if json.Unmarshal([]byte(data), &address) == nil {
				t.Errorf("expected decoding to fail, but instead it produced %v", address)
			}
		})
	}
}

func TestValue_JSON_Encoding(t *testing.T) {
	tests := []struct {
		value Value
		json  string
	}{
		{Value{}, "\"0x0000000000000000000000000000000000000000000000000000000000000000\""},
		{Value{1}, "\"0x0100000000000000000000000000000000000000000000000000000000000000\""},
		{Value{0xAB}, "\"0xab00000000000000000000000000000000000000000000000000000000000000\""},
		{
			Value{
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
				10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
				20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
				30, 31,
			},
			"\"0x000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f\"",
		},
	}

	for _, test := range tests {
		encoded, err := json.Marshal(test.value)
		if err != nil {
			t.Fatalf("failed to encode into JSON: %v", err)
		}

		if want, got := test.json, string(encoded); want != got {
			t.Errorf("unexpected JSON encoding, wanted %v, got %v", want, got)
		}

		var restored Value
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatalf("failed to restore address: %v", err)
		}
		if test.value != restored {
			t.Errorf("unexpected restored value, wanted %v, got %v", test.value, restored)
		}

	}
}

func TestCallKind_JSON_Encoding(t *testing.T) {
	tests := []struct {
		kind CallKind
		json string
	}{
		{Call, "\"call\""},
		{StaticCall, "\"static_call\""},
		{DelegateCall, "\"delegate_call\""},
		{CallCode, "\"call_code\""},
		{Create, "\"create\""},
		{Create2, "\"create2\""},
	}

	for _, test := range tests {
		encoded, err := json.Marshal(test.kind)
		if err != nil {
			t.Fatalf("failed to encode into JSON: %v", err)
		}

		if want, got := test.json, string(encoded); want != got {
			t.Errorf("unexpected JSON encoding, wanted %v, got %v", want, got)
		}

		var restored CallKind
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatalf("failed to restore address: %v", err)
		}
		if test.kind != restored {
			t.Errorf("unexpected restored value, wanted %v, got %v", test.kind, restored)
		}

	}
}

func TestCallKind_JSON_InvalidValueEncodingFails(t *testing.T) {
	if _, err := json.Marshal(CallKind(99)); err == nil {
		t.Errorf("expected encoding to fail")
	}
}
