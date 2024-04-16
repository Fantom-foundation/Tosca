//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

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

type StorageBuilder struct {
	s Storage
}

func NewStorageBuilder() *StorageBuilder {
	return &StorageBuilder{}
}

func (s *StorageBuilder) Build() *Storage {
	res := s.s
	s.s = Storage{}

	return &res
}

func (s *StorageBuilder) SetCurrent(key, value U256) *StorageBuilder {
	if s.s.current == nil {
		s.s.current = make(map[U256]U256)
	}
	s.s.current[key] = value
	return s
}

func (s *StorageBuilder) SetOriginal(key, value U256) *StorageBuilder {
	if s.s.original == nil {
		s.s.original = make(map[U256]U256)
	}
	s.s.original[key] = value
	return s
}

func (s *StorageBuilder) SetWarm(key U256, value bool) *StorageBuilder {
	if value {
		if s.s.warm == nil {
			s.s.warm = make(map[U256]bool)
		}
		s.s.warm[key] = value
	}
	return s
}

func (s *StorageBuilder) IsInOriginal(key U256) bool {
	_, isIn := s.s.original[key]
	return isIn
}

func (s *Storage) SetCurrent(key U256, value U256) {
	if s.current == nil {
		s.current = make(map[U256]U256)
	} else {
		s.current = maps.Clone(s.current)
	}
	s.current[key] = value
}

func (s *Storage) GetCurrent(key U256) U256 {
	return s.current[key]
}

func (s *Storage) RemoveCurrent(key U256) {
	if s.current != nil {
		s.current = maps.Clone(s.current)
	}
	delete(s.current, key)
}

func (s *Storage) SetOriginal(key U256, value U256) {
	if s.original == nil {
		s.original = make(map[U256]U256)
	} else {
		s.original = maps.Clone(s.original)
	}
	s.original[key] = value
}

func (s *Storage) GetOriginal(key U256) U256 {
	return s.original[key]
}

func (s *Storage) RemoveOriginal(key U256) {
	if s.original != nil {
		s.original = maps.Clone(s.original)
	}
	delete(s.original, key)
}

func (s *Storage) IsWarm(key U256) bool {
	return s.warm[key]
}

func (s *Storage) MarkWarm(key U256) {
	if s.warm == nil {
		s.warm = make(map[U256]bool)
	} else {
		s.warm = maps.Clone(s.warm)
	}
	s.warm[key] = true
}

func (s *Storage) MarkCold(key U256) {
	if s.warm != nil {
		s.warm = maps.Clone(s.warm)
	}
	delete(s.warm, key)
}

func (s *Storage) Clone() *Storage {
	return &Storage{
		current:  s.current,
		original: s.original,
		warm:     s.warm,
	}
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
