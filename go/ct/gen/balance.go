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
	warmCold []warmColdConstraint
}

func NewBalanceGenerator() *BalanceGenerator {
	return &BalanceGenerator{}
}

func (g *BalanceGenerator) Clone() *BalanceGenerator {
	return &BalanceGenerator{
		warmCold: slices.Clone(g.warmCold),
	}
}

func (g *BalanceGenerator) Restore(other *BalanceGenerator) {
	if g == other {
		return
	}
	g.warmCold = slices.Clone(other.warmCold)
}

func (g *BalanceGenerator) BindWarm(address Variable) {
	v := warmColdConstraint{address, true}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *BalanceGenerator) BindCold(address Variable) {
	v := warmColdConstraint{address, false}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *BalanceGenerator) String() string {
	var parts []string

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
	// Check for conflicts among warm/cold constraints.
	sort.Slice(g.warmCold, func(i, j int) bool { return g.warmCold[i].Less(&g.warmCold[j]) })
	for i := 0; i < len(g.warmCold)-1; i++ {
		a, b := g.warmCold[i], g.warmCold[i+1]
		if a.key == b.key && a.warm != b.warm {
			return nil, fmt.Errorf("%w, address %v conflicting warm/cold constraints", ErrUnsatisfiable, a.key)
		}
	}

	// When handling unbound variables, we need to generate an unused address for
	// them. We therefore track which addresses have already been used.
	addressesInUse := map[Address]bool{}
	addressesInUse[accountAddress] = true
	for _, con := range g.warmCold {
		if pos, isBound := assignment[con.key]; isBound {
			addressesInUse[NewAddress(pos)] = true
		}
	}
	getUnusedAddress := func() Address {
		for {
			address, _ := RandAddress(rnd)

			if _, isPresent := addressesInUse[address]; !isPresent {
				addressesInUse[address] = true
				return address
			}
		}
	}

	getAddress := func(v Variable) Address {
		key, isBound := assignment[v]
		address := NewAddress(key)
		if !isBound {
			address = getUnusedAddress()
			assignment[v] = address.ToU256()
		}

		return address
	}

	balance := st.NewBalance()
	// Process warm/cold constraints.
	for _, con := range g.warmCold {
		address := getAddress(con.key)
		if _, isPresent := balance.Current[address]; !isPresent {
			balance.Current[address] = RandU256(rnd)
		}
		balance.SetWarm(address, con.warm)
	}

	// Some random entries.
	for i, max := 0, rnd.Intn(5); i < max; i++ {
		address := getUnusedAddress()
		balance.Current[address] = RandU256(rnd)
		balance.SetWarm(address, rnd.Intn(2) == 1)
	}

	// Add own account address
	address := accountAddress
	balance.Current[address] = RandU256(rnd)

	return balance, nil
}
