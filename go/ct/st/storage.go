package st

import (
	"fmt"

	"golang.org/x/exp/maps"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

type Storage struct {
	current  map[U256]U256
	original map[U256]U256
	warm     map[U256]bool
}

func NewStorage() *Storage {
	return &Storage{
		current:  make(map[U256]U256),
		original: make(map[U256]U256),
		warm:     make(map[U256]bool),
	}
}

func (s *Storage) SetCurrent(key U256, value U256) {
	s.current[key] = value
}

func (s *Storage) GetCurrent(key U256) U256 {
	return s.current[key]
}

func (s *Storage) SetOriginal(key U256, value U256) {
	s.original[key] = value
}

func (s *Storage) GetOriginal(key U256) U256 {
	return s.original[key]
}

func (s *Storage) IsInOriginal(key U256) bool {
	_, isIn := s.original[key]
	return isIn
}

func (s *Storage) Clone() *Storage {
	return &Storage{
		current:  maps.Clone(s.current),
		original: maps.Clone(s.original),
		warm:     maps.Clone(s.warm),
	}
}

func (s *Storage) IsWarm(key U256) bool {
	return s.warm[key]
}

func (s *Storage) MarkWarmCold(key U256, warm bool) {
	if warm {
		s.MarkWarm(key)
	} else {
		s.MarkCold(key)
	}
}

func (s *Storage) MarkWarm(key U256) {
	s.warm[key] = true
}

func (s *Storage) MarkCold(key U256) {
	delete(s.warm, key)
}

func mapEqualIgnoringZeroValues[K comparable](a map[K]U256, b map[K]U256) bool {
	for key, valueA := range a {
		valueB, contained := b[key]
		if !contained && valueA != NewU256(0) {
			return false
		} else if valueA != valueB {
			return false
		}
	}
	for key, valueB := range b {
		if _, contained := a[key]; !contained && valueB != NewU256(0) {
			return false
		}
	}
	return true
}

func (a *Storage) Eq(b *Storage) bool {
	return mapEqualIgnoringZeroValues(a.current, b.current) &&
		maps.Equal(a.original, b.original) &&
		maps.Equal(a.warm, b.warm)
}

func (a *Storage) Diff(b *Storage) (res []string) {
	for key, valueA := range a.current {
		valueB, contained := b.current[key]
		if !contained && valueA != NewU256(0) {
			res = append(res, fmt.Sprintf("Different current entry:\n\t[%v]=%v\n\tvs\n\tmissing", key, valueA))
		} else if valueA != valueB {
			res = append(res, fmt.Sprintf("Different current entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", key, valueA, key, valueB))
		}
	}
	for key, valueB := range b.current {
		if _, contained := a.current[key]; !contained && valueB != NewU256(0) {
			res = append(res, fmt.Sprintf("Different current entry:\n\tmissing\n\tvs\n\t[%v]=%v", key, valueB))
		}
	}

	for key, valueA := range a.original {
		valueB, contained := b.original[key]
		if !contained {
			res = append(res, fmt.Sprintf("Different original entry:\n\t[%v]=%v\n\tvs\n\tmissing", key, valueA))
		} else if valueA != valueB {
			res = append(res, fmt.Sprintf("Different original entry:\n\t[%v]=%v\n\tvs\n\t[%v]=%v", key, valueA, key, valueB))
		}
	}
	for key, valueB := range b.original {
		if _, contained := a.original[key]; !contained {
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
