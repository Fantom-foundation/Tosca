// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package gen

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"golang.org/x/exp/maps"
	"pgregory.net/rand"
)

type AccountsGenerator struct {
	existingAccounts []existenceConstraint
	minBalance       []balanceConstraint
	maxBalance       []balanceConstraint
	warmCold         []warmColdConstraint
}

func NewAccountGenerator() *AccountsGenerator {
	return &AccountsGenerator{}
}

func (g *AccountsGenerator) Clone() *AccountsGenerator {
	return &AccountsGenerator{
		existingAccounts: slices.Clone(g.existingAccounts),
		warmCold:         slices.Clone(g.warmCold),
		minBalance:       slices.Clone(g.minBalance),
		maxBalance:       slices.Clone(g.maxBalance),
	}
}

func (g *AccountsGenerator) Restore(other *AccountsGenerator) {
	if g == other {
		return
	}
	g.existingAccounts = slices.Clone(other.existingAccounts)
	g.warmCold = slices.Clone(other.warmCold)
	g.minBalance = slices.Clone(other.minBalance)
	g.maxBalance = slices.Clone(other.maxBalance)
}

func (g *AccountsGenerator) BindToAddressOfExistingAccount(address Variable) {
	c := existenceConstraint{address, true}
	if !slices.Contains(g.existingAccounts, c) {
		g.existingAccounts = append(g.existingAccounts, c)
	}
}

func (g *AccountsGenerator) BindToAddressOfNonExistingAccount(address Variable) {
	c := existenceConstraint{address, false}
	if !slices.Contains(g.existingAccounts, c) {
		g.existingAccounts = append(g.existingAccounts, c)
	}
}

func (g *AccountsGenerator) AddBalanceLowerBound(address Variable, value U256) {
	c := balanceConstraint{address, value}
	if !slices.Contains(g.minBalance, c) {
		g.minBalance = append(g.minBalance, c)
	}
}

func (g *AccountsGenerator) AddBalanceUpperBound(address Variable, value U256) {
	c := balanceConstraint{address, value}
	if !slices.Contains(g.maxBalance, c) {
		g.maxBalance = append(g.maxBalance, c)
	}
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

	sort.Slice(g.existingAccounts, func(i, j int) bool {
		return g.existingAccounts[i].Less(&g.existingAccounts[j])
	})
	for _, con := range g.existingAccounts {
		if con.exists {
			parts = append(parts, fmt.Sprintf("exists(%v)", con.address))
		} else {
			parts = append(parts, fmt.Sprintf("!exists(%v)", con.address))
		}
	}

	sort.Slice(g.minBalance, func(i, j int) bool {
		return g.minBalance[i].Less(&g.minBalance[j])
	})
	for _, con := range g.minBalance {
		parts = append(parts, fmt.Sprintf("balance(%v) ≥ %v", con.address, con.value.DecimalString()))
	}

	sort.Slice(g.maxBalance, func(i, j int) bool {
		return g.maxBalance[i].Less(&g.maxBalance[j])
	})
	for _, con := range g.maxBalance {
		parts = append(parts, fmt.Sprintf("balance(%v) ≤ %v", con.address, con.value.DecimalString()))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

func (g *AccountsGenerator) Generate(assignment Assignment, rnd *rand.Rand, accountAddress tosca.Address) (*st.Accounts, error) {
	// Check for conflicts among existence constraints.
	sort.Slice(g.existingAccounts, func(i, j int) bool {
		return g.existingAccounts[i].Less(&g.existingAccounts[j])
	})
	for i := 0; i < len(g.existingAccounts)-1; i++ {
		a, b := g.existingAccounts[i], g.existingAccounts[i+1]
		if a.address == b.address && a.exists != b.exists {
			return nil, fmt.Errorf("%w, address %v conflicting existence constraints", ErrUnsatisfiable, a.address)
		}
	}

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
	addressesInUse := map[tosca.Address]bool{}
	addressesInUse[accountAddress] = true
	for _, con := range g.warmCold {
		if pos, isBound := assignment[con.key]; isBound {
			addressesInUse[NewAddress(pos)] = true
		}
	}
	getUnusedAddress := func() tosca.Address {
		for {
			address := RandomAddress(rnd)

			if _, isPresent := addressesInUse[address]; !isPresent {
				addressesInUse[address] = true
				return address
			}
		}
	}

	getBoundOrBindNewAddress := func(v Variable) tosca.Address {
		value, isBound := assignment[v]
		address := NewAddress(value)
		if !isBound {
			address = getUnusedAddress()
			assignment[v] = AddressToU256(address)
		}

		return address
	}

	accountsBuilder := st.NewAccountsBuilder()

	// Process existence constraints.
	for _, con := range g.existingAccounts {
		address := getBoundOrBindNewAddress(con.address)
		if con.exists {
			accountsBuilder.MarkExisting(address)
		}
	}

	// Process warm/cold constraints.
	for _, con := range g.warmCold {
		address := getBoundOrBindNewAddress(con.key)
		if con.warm {
			accountsBuilder.SetWarm(address)
		}
	}

	// Apply balance constraints.
	if err := g.processBalanceConstraints(getBoundOrBindNewAddress, rnd, accountsBuilder); err != nil {
		return nil, err
	}

	// Some random entries.
	for i, max := 0, rnd.Intn(5); i < max; i++ {
		address := getUnusedAddress()
		accountsBuilder.SetBalance(address, RandU256(rnd))
		if rnd.Intn(2) == 1 {
			accountsBuilder.SetCode(address, RandomBytesOfSize(rnd, 42))
		}
		if rnd.Intn(2) == 1 {
			accountsBuilder.SetWarm(address)
		}
	}

	// Add balance and code for the account executing the code.
	if !accountsBuilder.Exists(accountAddress) {
		accountsBuilder.SetBalance(accountAddress, RandU256(rnd))
	}
	accountsBuilder.SetCode(accountAddress, RandomBytesOfSize(rnd, 42))

	return accountsBuilder.Build(), nil
}

func (g *AccountsGenerator) processBalanceConstraints(
	getBoundOrBindNewAddress func(v Variable) tosca.Address,
	rnd *rand.Rand,
	result *st.AccountsBuilder,
) error {

	// Convert variable based constraints into lower and upper bounds for
	// concrete addresses by fetching variable bindings or assigning new addresses.
	lowerBounds := map[tosca.Address]U256{}
	for _, con := range g.minBalance {
		address := getBoundOrBindNewAddress(con.address)
		current := lowerBounds[address]
		if current.Lt(con.value) {
			lowerBounds[address] = con.value
		}
	}

	upperBounds := map[tosca.Address]U256{}
	for _, con := range g.maxBalance {
		address := getBoundOrBindNewAddress(con.address)
		current, found := upperBounds[address]
		if !found || current.Gt(con.value) {
			upperBounds[address] = con.value
		}
	}

	// Get list of all accounts to be resolved.
	accounts := maps.Keys(lowerBounds)
	for _, address := range maps.Keys(upperBounds) {
		if _, isPresent := lowerBounds[address]; !isPresent {
			accounts = append(accounts, address)
		}
	}

	// Apply constraints to the accounts to obtain balances.
	for _, address := range accounts {
		lower, hasLower := lowerBounds[address]
		upper, hasUpper := upperBounds[address]

		if !hasLower {
			lower = NewU256()
		}
		if !hasUpper {
			upper = MaxU256()
		}

		if lower.Gt(upper) {
			return fmt.Errorf("%w, conflicting balance constraints", ErrUnsatisfiable)
		}

		result.SetBalance(address, RandU256Between(rnd, lower, upper))
	}

	return nil
}

type existenceConstraint struct {
	address Variable
	exists  bool
}

func (a *existenceConstraint) Less(b *existenceConstraint) bool {
	if a.address != b.address {
		return a.address < b.address
	}
	return a.exists != b.exists && a.exists
}

type balanceConstraint struct {
	address Variable
	value   U256
}

func (a *balanceConstraint) Less(b *balanceConstraint) bool {
	if a.address != b.address {
		return a.address < b.address
	}
	return a.value.Lt(b.value)
}
