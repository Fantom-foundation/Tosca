package st

import (
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"golang.org/x/exp/maps"
)

type Transient struct {
	storage map[U256]U256
}

func (t *Transient) SetStorage(key U256, value U256) {
	if t.storage == nil {
		t.storage = make(map[U256]U256)
	} else {
		t.storage[key] = value
	}
}

func (t *Transient) GetStorage(key U256) U256 {
	return t.storage[key]
}

func (t *Transient) IsInStorage(key U256) bool {
	_, isIn := t.storage[key]
	return isIn
}

func (t *Transient) DeleteStorage(key U256) {
	delete(t.storage, key)
}

func (t *Transient) Clone() *Transient {
	return &Transient{maps.Clone(t.storage)}
}

func (t *Transient) Eq(other *Transient) bool {
	return mapEqualIgnoringZeroValues(t.storage, other.storage)
}

func (t *Transient) Diff(other *Transient) (res []string) {
	res = append(res, mapDiffIgnoringZeroValues(t.storage, other.storage)...)
	return
}
