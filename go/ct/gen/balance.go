package gen

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"pgregory.net/rand"
)

type BalanceGenerator struct {
	cfg      []balanceConfigConstraint
	warmCold []warmColdConstraint
}

type balanceConfigConstraint struct {
	key      Variable
	newValue Variable
}

type BalanceCfg int

func (a *balanceConfigConstraint) Less(b *balanceConfigConstraint) bool {
	if a.key != b.key {
		return a.key < b.key
	}
	return a.newValue < b.newValue
}

func NewBalanceGenerator() *BalanceGenerator {
	return &BalanceGenerator{}
}

func (g *BalanceGenerator) Clone() *BalanceGenerator {
	return &BalanceGenerator{
		cfg:      slices.Clone(g.cfg),
		warmCold: slices.Clone(g.warmCold),
	}
}

func (g *BalanceGenerator) Restore(other *BalanceGenerator) {
	if g == other {
		return
	}
	g.cfg = slices.Clone(other.cfg)
	g.warmCold = slices.Clone(other.warmCold)
}

func (g *BalanceGenerator) BindConfiguration(key, newValue Variable) {
	v := balanceConfigConstraint{key, newValue}
	if !slices.Contains(g.cfg, v) {
		g.cfg = append(g.cfg, v)
	}
}

func (g *BalanceGenerator) BindWarm(key Variable) {
	v := warmColdConstraint{key, true}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *BalanceGenerator) BindCold(key Variable) {
	v := warmColdConstraint{key, false}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *BalanceGenerator) String() string {
	var parts []string

	sort.Slice(g.cfg, func(i, j int) bool { return g.cfg[i].Less(&g.cfg[j]) })
	for _, con := range g.cfg {
		parts = append(parts, fmt.Sprintf("cfg[%v]=%v", con.key, con.newValue))
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

func (g *BalanceGenerator) Generate(assignment Assignment, rnd *rand.Rand, accountAddress Address) (*st.Balance, error) {
	// Check for conflicts among balance configuration constraints
	sort.Slice(g.cfg, func(i, j int) bool { return g.cfg[i].Less(&g.cfg[j]) })
	for i := 0; i < len(g.cfg)-1; i++ {
		a, b := g.cfg[i], g.cfg[i+1]
		if a.key == b.key && a.newValue != b.newValue {
			return nil, fmt.Errorf("%w, key %v conflicting balance configuration", ErrUnsatisfiable, a.key)
		}
	}

	// Check for conflicts among warm/cold constraints.
	sort.Slice(g.warmCold, func(i, j int) bool { return g.warmCold[i].Less(&g.warmCold[j]) })
	for i := 0; i < len(g.warmCold)-1; i++ {
		a, b := g.warmCold[i], g.warmCold[i+1]
		if a.key == b.key && a.warm != b.warm {
			return nil, fmt.Errorf("%w, key %v conflicting warm/cold constraints", ErrUnsatisfiable, a.key)
		}
	}

	// When handling unbound variables, we need to generate an unused key for
	// them. We therefore track which keys have already been used.
	keysInUse := map[Address]bool{}
	for _, con := range g.cfg {
		if key, isBound := assignment[con.key]; isBound {
			keysInUse[NewAddress(key)] = true
		}
	}
	for _, con := range g.warmCold {
		if key, isBound := assignment[con.key]; isBound {
			keysInUse[NewAddress(key)] = true
		}
	}
	getUnusedKey := func() Address {
		for {
			key, _ := RandAddress(rnd)

			if _, isPresent := keysInUse[key]; !isPresent {
				keysInUse[key] = true
				return key
			}
		}
	}

	getKey := func(v Variable) Address {
		key, isBound := assignment[v]
		address := NewAddress(key)
		if !isBound {
			address = getUnusedKey()
			assignment[v] = address.ToU256()
		}

		return address
	}

	balance := st.NewBalance()

	// Process balance configuration constraints.
	for _, con := range g.cfg {
		key := getKey(con.key)

		value := assignment[con.newValue]

		balance.Current[key] = value
		balance.MarkWarmCold(key, rnd.Intn(2) == 1)
	}

	// Process warm/cold constraints.
	for _, con := range g.warmCold {
		key := getKey(con.key)
		if _, isPresent := balance.Current[key]; !isPresent {
			balance.Current[key] = RandU256(rnd)
		}
		balance.MarkWarmCold(key, con.warm)
	}

	// Some random entries.
	for i, max := 0, rnd.Intn(5); i < max; i++ {
		key := getUnusedKey()
		balance.Current[key] = RandU256(rnd)
		balance.MarkWarmCold(key, rnd.Intn(2) == 1)
	}

	// Add own account address
	key := accountAddress
	balance.Current[key] = RandU256(rnd)

	return balance, nil
}
