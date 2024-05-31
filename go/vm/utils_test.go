package vm

import (
	"math/rand"
	"testing"

	"github.com/holiman/uint256"
)

func randomUint256(rnd *rand.Rand) *uint256.Int {
	var value uint256.Int
	value[0] = rnd.Uint64()
	value[1] = rnd.Uint64()
	value[2] = rnd.Uint64()
	value[3] = rnd.Uint64()
	return &value
}

func TestUint256ToValue(t *testing.T) {
	rnd := rand.New(rand.NewSource(0))
	value := randomUint256(rnd)
	result := Uint256ToValue(value)
	expected := value.Bytes32()
	if result != expected {
		t.Errorf("incorrect result, got: %v, want: %v", result, expected)
	}
}

func TestUint256ToValue_nil(t *testing.T) {
	result := Uint256ToValue(nil)
	expected := [32]byte{}
	if result != expected {
		t.Errorf("incorrect result, got: %v, want: %v", result, expected)
	}
}

func TestValueToUint256(t *testing.T) {
	rnd := rand.New(rand.NewSource(0))
	expected := randomUint256(rnd)
	result := ValueToUint256(expected.Bytes32())
	if result.Cmp(expected) != 0 {
		t.Errorf("incorrect result, got: %v, want: %v", result, expected)
	}
}
