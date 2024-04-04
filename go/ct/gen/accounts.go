package gen

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

type AccountsGenerator struct {
	warmCold []warmColdConstraint
}

func NewAccountGenerator() *AccountsGenerator {
	return &AccountsGenerator{}
}

func (g *AccountsGenerator) Clone() *AccountsGenerator {
	return &AccountsGenerator{
		warmCold: slices.Clone(g.warmCold),
	}
}

func (g *AccountsGenerator) Restore(other *AccountsGenerator) {
	if g == other {
		return
	}
	g.warmCold = slices.Clone(other.warmCold)
}

func (g *AccountsGenerator) BindWarm(address Variable) {
	v := warmColdConstraint{address, true}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *AccountsGenerator) BindCold(address Variable) {
	v := warmColdConstraint{address, false}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *AccountsGenerator) String() string {
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

// Code saved in accounts is never executed,
// to keep overhead and complexity to a minimum,
// it is just random data with random length
func randCode(rnd *rand.Rand) []byte {
	size := rnd.Intn(42)
	code := make([]byte, 0, size)
	rnd.Read(code)
	return code
}

func (g *AccountsGenerator) Generate(assignment Assignment, rnd *rand.Rand, accountAddress vm.Address) (*st.Accounts, error) {
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
	addressesInUse := map[vm.Address]bool{}
	addressesInUse[accountAddress] = true
	for _, con := range g.warmCold {
		if pos, isBound := assignment[con.key]; isBound {
			addressesInUse[NewAddress(pos)] = true
		}
	}
	getUnusedAddress := func() vm.Address {
		for {
			address, _ := RandAddress(rnd)

			if _, isPresent := addressesInUse[address]; !isPresent {
				addressesInUse[address] = true
				return address
			}
		}
	}

	getAddress := func(v Variable) vm.Address {
		key, isBound := assignment[v]
		address := NewAddress(key)
		if !isBound {
			address = getUnusedAddress()
			assignment[v] = AddressToU256(address)
		}

		return address
	}

	accountsBuilder := st.NewAccountsBuilder()

	// Process warm/cold constraints.
	for _, con := range g.warmCold {
		address := getAddress(con.key)
		// TODO: Not every warm address requires balance or code
		if !accountsBuilder.Exists(address) {
			accountsBuilder.SetBalance(address, RandU256(rnd))
			accountsBuilder.SetCode(address, randCode(rnd))
		}
		if con.warm {
			accountsBuilder.SetWarm(address)
		}
	}

	// Some random entries.
	for i, max := 0, rnd.Intn(5); i < max; i++ {
		address := getUnusedAddress()
		accountsBuilder.SetBalance(address, RandU256(rnd))
		if rnd.Intn(2) == 1 {
			accountsBuilder.SetCode(address, randCode(rnd))
		}
		if rnd.Intn(2) == 1 {
			accountsBuilder.SetWarm(address)
		}
	}

	// Add own account address
	accountsBuilder.SetBalance(accountAddress, RandU256(rnd))
	accountsBuilder.SetCode(accountAddress, randCode(rnd))

	return accountsBuilder.Build(), nil
}
