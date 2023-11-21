package st

import (
	"fmt"

	"golang.org/x/exp/maps"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

type Storage struct {
	Current  map[U256]U256
	Original map[U256]U256
	warm     map[U256]bool
}

func NewStorage() *Storage {
	return &Storage{
		Current:  make(map[U256]U256),
		Original: make(map[U256]U256),
		warm:     make(map[U256]bool),
	}
}

func (s *Storage) Clone() *Storage {
	return &Storage{
		Current:  maps.Clone(s.Current),
		Original: maps.Clone(s.Original),
		warm:     maps.Clone(s.warm),
	}
}

func (s *Storage) IsWarm(key U256) bool {
	return s.warm[key]
}

func (s *Storage) MarkWarm(key U256) {
	s.warm[key] = true
}

func (s *Storage) MarkCold(key U256) {
	delete(s.warm, key)
}

func (a *Storage) Eq(b *Storage) bool {
	return maps.Equal(a.Current, b.Current) &&
		maps.Equal(a.Original, b.Original) &&
		maps.Equal(a.warm, b.warm)
}

func (a *Storage) Diff(b *Storage) (res []string) {
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

	for key, valueA := range a.Original {
		valueB, contained := b.Original[key]
		if !contained {
			res = append(res, fmt.Sprintf("Different original entry:\n\t[%v]=%v\n\tvs\n\tmissing", key, valueA))
		} else if valueA != valueB {
			res = append(res, fmt.Sprintf("Different original entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", key, valueA, key, valueB))
		}
	}
	for key, valueB := range b.Original {
		if _, contained := a.Original[key]; !contained {
			res = append(res, fmt.Sprintf("Different original entry:\n\tmissing\n\tvs\n\t[%v]=%v", key, valueB))
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
