package common

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"pgregory.net/rand"
)

func TestHash_GetRandomHash(t *testing.T) {
	rnd := rand.New()
	hash := GetRandomHash(rnd)
	if hash == (tosca.Hash{}) {
		t.Errorf("GetRandomHash() = %v, want non-zero", hash)
	}
}
