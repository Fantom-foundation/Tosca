package gen

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/exp/slices"
	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type StorageGenerator struct {
	current  []valueConstraint
	warmCold []warmColdConstraint
}

type valueConstraint struct {
	key   Variable
	value U256
}

func (a *valueConstraint) Less(b *valueConstraint) bool {
	if a.key != b.key {
		return a.key < b.key
	}
	return a.value.Lt(b.value)
}

type warmColdConstraint struct {
	key  Variable
	warm bool
}

func (a *warmColdConstraint) Less(b *warmColdConstraint) bool {
	if a.key != b.key {
		return a.key < b.key
	}
	return a.warm != b.warm && a.warm
}

func NewStorageGenerator() *StorageGenerator {
	return &StorageGenerator{}
}

func (g *StorageGenerator) BindCurrent(key Variable, value U256) {
	v := valueConstraint{key, value}
	if !slices.Contains(g.current, v) {
		g.current = append(g.current, v)
	}
}

func (g *StorageGenerator) BindWarm(key Variable) {
	v := warmColdConstraint{key, true}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *StorageGenerator) BindCold(key Variable) {
	v := warmColdConstraint{key, false}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *StorageGenerator) Generate(assignment Assignment, rnd *rand.Rand) (*st.Storage, error) {
	// Check for conflicts among value and warm/cold constraints.
	sort.Slice(g.current, func(i, j int) bool { return g.current[i].Less(&g.current[j]) })
	for i := 0; i < len(g.current)-1; i++ {
		a, b := g.current[i], g.current[i+1]
		if a.key == b.key && a.value != b.value {
			return nil, fmt.Errorf("%w, key %v has conflicting values", ErrUnsatisfiable, a.key)
		}
	}
	sort.Slice(g.warmCold, func(i, j int) bool { return g.warmCold[i].Less(&g.warmCold[j]) })
	for i := 0; i < len(g.warmCold)-1; i++ {
		a, b := g.warmCold[i], g.warmCold[i+1]
		if a.key == b.key && a.warm != b.warm {
			return nil, fmt.Errorf("%w, key %v warm/cold not satisfiable", ErrUnsatisfiable, a.key)
		}
	}

	// When handling unbound variables, we need to generate an unused key for
	// them. We therefore track which keys have already been used.
	keysInUse := map[U256]bool{}
	for _, con := range g.current {
		if key, isBound := assignment[con.key]; isBound {
			keysInUse[key] = true
		}
	}
	for _, con := range g.warmCold {
		if key, isBound := assignment[con.key]; isBound {
			keysInUse[key] = true
		}
	}
	getUnusedKey := func() U256 {
		for {
			key := RandU256(rnd)
			if _, isPresent := keysInUse[key]; !isPresent {
				keysInUse[key] = true
				return key
			}
		}
	}

	getKey := func(v Variable) U256 {
		key, isBound := assignment[v]
		if !isBound {
			key = getUnusedKey()
			assignment[v] = key // update assignment
		}
		return key
	}
	getValue := func(v Variable) U256 {
		for _, con := range g.current {
			if con.key == v {
				return con.value
			}
		}
		return RandU256(rnd)
	}
	getWarm := func(v Variable) bool {
		for _, con := range g.warmCold {
			if con.key == v {
				return con.warm
			}
		}
		return rnd.Intn(2) == 1
	}

	// Collect all variables.
	variables := map[Variable]bool{}
	for _, con := range g.current {
		variables[con.key] = true
	}
	for _, con := range g.warmCold {
		variables[con.key] = true
	}

	storage := st.NewStorage()

	// Process constraints.
	for v := range variables {
		key := getKey(v)
		storage.Current[key] = getValue(v)
		storage.MarkWarmCold(key, getWarm(v))
	}

	// Also, add some random entries.
	for i, max := 0, rnd.Intn(5); i < max; i++ {
		key := getUnusedKey()
		storage.Current[key] = RandU256(rnd)
		storage.MarkWarmCold(key, rnd.Intn(2) == 1)
	}

	return storage, nil
}

func (g *StorageGenerator) Clone() *StorageGenerator {
	return &StorageGenerator{
		current:  slices.Clone(g.current),
		warmCold: slices.Clone(g.warmCold),
	}
}

func (g *StorageGenerator) Restore(other *StorageGenerator) {
	if g == other {
		return
	}
	g.current = slices.Clone(other.current)
	g.warmCold = slices.Clone(other.warmCold)
}

func (g *StorageGenerator) String() string {
	var parts []string

	sort.Slice(g.current, func(i, j int) bool { return g.current[i].Less(&g.current[j]) })
	for _, con := range g.current {
		parts = append(parts, fmt.Sprintf("[%v]=%v", con.key, con.value))
	}

	sort.Slice(g.warmCold, func(i, j int) bool { return g.warmCold[i].Less(&g.warmCold[j]) })
	for _, con := range g.warmCold {
		if con.warm {
			parts = append(parts, fmt.Sprintf("warm(%v)", con.key))
		} else {
			parts = append(parts, fmt.Sprintf("cold(%v)", con.key))
		}
	}

	return "{" + strings.Join(parts, ",") + "}"
}
