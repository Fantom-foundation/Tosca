package vm

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func (a Address) MarshalJSON() ([]byte, error) {
	res := fmt.Sprintf("%x", a)
	return json.Marshal(res)
}

func (a *Address) UnmarshalJSON(data []byte) error {
	return jsonToBytes(a[:], data)
}

func (v Value) MarshalJSON() ([]byte, error) {
	return bytesToJSON(v[:])
}

func (v *Value) UnmarshalJSON(data []byte) error {
	return jsonToBytes(v[:], data)
}

func bytesToJSON(data []byte) ([]byte, error) {
	res := fmt.Sprintf("%x", data)
	return json.Marshal(res)
}

func jsonToBytes(trg []byte, data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	data, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	if want, got := len(trg), len(data); want != got {
		return fmt.Errorf("invalid format, wanted %d bytes, got %d", want, got)
	}
	copy(trg[:], data)
	return nil
}

func (k CallKind) String() string {
	switch k {
	case Call:
		return "call"
	case StaticCall:
		return "static_call"
	case DelegateCall:
		return "delegate_call"
	case CallCode:
		return "call_code"
	case Create:
		return "create"
	case Create2:
		return "create2"
	default:
		return "unknown"
	}
}

func (k CallKind) MarshalJSON() ([]byte, error) {
	var res string
	switch k {
	case Call:
		res = "call"
	case StaticCall:
		res = "static_call"
	case DelegateCall:
		res = "delegate_call"
	case CallCode:
		res = "call_code"
	case Create:
		res = "create"
	case Create2:
		res = "create2"
	default:
		return nil, fmt.Errorf("invalid call kind: %v", k)
	}
	return json.Marshal(res)
}

func (k *CallKind) UnmarshalJSON(data []byte) error {
	var kind string
	if err := json.Unmarshal(data, &kind); err != nil {
		return err
	}
	switch strings.ToLower(kind) {
	case "call":
		*k = Call
	case "static_call":
		*k = StaticCall
	case "delegate_call":
		*k = DelegateCall
	case "call_code":
		*k = CallCode
	case "create":
		*k = Create
	case "create2":
		*k = Create2
	default:
		return fmt.Errorf("unknown call kind: %s", kind)
	}
	return nil
}
