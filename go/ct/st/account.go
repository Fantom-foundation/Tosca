package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"golang.org/x/exp/maps"
)

type Account struct {
	Balance map[Address]U256
	warm    map[Address]struct{}
}

func NewAccount() *Account {
	return &Account{
		Balance: make(map[Address]U256),
		warm:    make(map[Address]struct{}),
	}
}

func (b *Account) Clone() *Account {
	return &Account{
		Balance: maps.Clone(b.Balance),
		warm:    maps.Clone(b.warm),
	}
}

func (a *Account) Eq(b *Account) bool {
	return maps.Equal(a.Balance, b.Balance) &&
		maps.Equal(a.warm, b.warm)
}

func (a *Account) Diff(b *Account) (res []string) {
	for key, valueA := range a.Balance {
		valueB, contained := b.Balance[key]
		if !contained {
			res = append(res, fmt.Sprintf("Different balance entry:\n\t[%v]=%v\n\tvs\n\tmissing", key, valueA))
		} else if valueA != valueB {
			res = append(res, fmt.Sprintf("Different balance entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", key, valueA, key, valueB))
		}
	}
	for key, valueB := range b.Balance {
		if _, contained := a.Balance[key]; !contained {
			res = append(res, fmt.Sprintf("Different balance entry:\n\tmissing\n\tvs\n\t[%v]=%v", key, valueB))
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

func (b *Account) IsWarm(key Address) bool {
	_, contains := b.warm[key]
	return contains
}

func (b *Account) IsCold(key Address) bool {
	_, contains := b.warm[key]
	return !contains
}

func (b *Account) MarkWarm(key Address) {
	b.warm[key] = struct{}{}
}

func (b *Account) MarkCold(key Address) {
	delete(b.warm, key)
}

func (b *Account) SetWarm(key Address, warm bool) {
	if warm {
		b.MarkWarm(key)
	} else {
		b.MarkCold(key)
	}
}
