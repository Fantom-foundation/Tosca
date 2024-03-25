package vm

import (
	"encoding/json"
	"testing"
)

func TestAddress_JSON_Encoding(t *testing.T) {
	tests := []struct {
		address Address
		json    string
	}{
		{Address{}, "\"0000000000000000000000000000000000000000\""},
		{Address{1}, "\"0100000000000000000000000000000000000000\""},
		{Address{0xAB}, "\"ab00000000000000000000000000000000000000\""},
		{
			Address{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			"\"000102030405060708090a0b0c0d0e0f10111213\"",
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
		"way to short":      "\"123\"",
		"just to short":     "\"000102030405060708090a0b0c0d0e0f1011121\"",
		"just to long":      "\"000102030405060708090a0b0c0d0e0f101112131\"",
		"not hex":           "\"hello, this is a test with 20 characters\"",
		"not a JSON string": "000102030405060708090a0b0c0d0e0f10111213",
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			var address Address
			if json.Unmarshal([]byte(data), &address) == nil {
				t.Errorf("expected decoding to fail, but instead it produced a result")
			}
		})
	}
}

func TestValue_JSON_Encoding(t *testing.T) {
	tests := []struct {
		value Value
		json  string
	}{
		{Value{}, "\"0000000000000000000000000000000000000000000000000000000000000000\""},
		{Value{1}, "\"0100000000000000000000000000000000000000000000000000000000000000\""},
		{Value{0xAB}, "\"ab00000000000000000000000000000000000000000000000000000000000000\""},
		{
			Value{
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
				10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
				20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
				30, 31,
			},
			"\"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f\"",
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
