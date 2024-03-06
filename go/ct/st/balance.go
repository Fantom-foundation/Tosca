package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"golang.org/x/exp/maps"
)

type Balance struct {
	Current map[Address]U256
	warm    map[Address]struct{}
}

func NewBalance() *Balance {
	return &Balance{
		Current: make(map[Address]U256),
		warm:    make(map[Address]struct{}),
	}
}

func (b *Balance) Clone() *Balance {
	return &Balance{
		Current: maps.Clone(b.Current),
		warm:    maps.Clone(b.warm),
	}
}

func (a *Balance) Eq(b *Balance) bool {
	return maps.Equal(a.Current, b.Current) &&
		maps.Equal(a.warm, b.warm)
}

func (a *Balance) Diff(b *Balance) (res []string) {
	for key, valueA := range a.Current {
		valueB, contained := b.Current[key]
		if !contained {
			res = append(res, fmt.Sprintf("Different current entry:\n\t[%v]=%v\n\tvs\n\tmissing", key, valueA))
		} else if valueA != valueB {
			res = append(res, fmt.Sprintf("Different current entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", key, valueA, key, valueB))
		}
	}
	for key, valueB := range b.Current {
		if _, contained := a.Current[key]; !contained {
			res = append(res, fmt.Sprintf("Different current entry:\n\tmissing\n\tvs\n\t[%v]=%v", key, valueB))
		}
	}

	for key := range a.warm {
		if _, contained := b.warm[key]; !contained {
			res = append(res, fmt.Sprintf("Different warm entry: %v vs missing", key))
		}
	}
	for key := range b.warm {
		if _, contained := a.warm[key]; !contained {
			res = append(res, fmt.Sprintf("Different warm entry: missing vs %v", key))
		}
	}

	return
}

func (b *Balance) IsWarm(key Address) bool {
	_, contains := b.warm[key]
	return contains
}

func (b *Balance) IsCold(key Address) bool {
	_, contains := b.warm[key]
	return !contains
}

func (b *Balance) MarkWarm(key Address) {
	b.warm[key] = struct{}{}
}

func (b *Balance) MarkCold(key Address) {
	delete(b.warm, key)
}

func (b *Balance) SetWarm(key Address, warm bool) {
	if warm {
		b.MarkWarm(key)
	} else {
		b.MarkCold(key)
	}
}
