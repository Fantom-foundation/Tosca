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

type AccountGenerator struct {
	warmCold []warmColdConstraint
}

func NewAccountGenerator() *AccountGenerator {
	return &AccountGenerator{}
}

func (g *AccountGenerator) Clone() *AccountGenerator {
	return &AccountGenerator{
		warmCold: slices.Clone(g.warmCold),
	}
}

func (g *AccountGenerator) Restore(other *AccountGenerator) {
	if g == other {
		return
	}
	g.warmCold = slices.Clone(other.warmCold)
}

func (g *AccountGenerator) BindWarm(address Variable) {
	v := warmColdConstraint{address, true}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *AccountGenerator) BindCold(address Variable) {
	v := warmColdConstraint{address, false}
	if !slices.Contains(g.warmCold, v) {
		g.warmCold = append(g.warmCold, v)
	}
}

func (g *AccountGenerator) String() string {
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

func randCode(rnd *rand.Rand) []byte {
	size := rnd.Intn(42)
	code := make([]byte, 0, size)
	for i := 0; i < size; i++ {
		var op OpCode
		for {
			op = OpCode(rnd.Intn(256))
			if IsValid(op) {
				break
			}
		}
		code = append(code, byte(op))
	}
	return code
}

func (g *AccountGenerator) Generate(assignment Assignment, rnd *rand.Rand, accountAddress Address) (*st.Account, error) {
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

	account := st.NewAccount()
	// Process warm/cold constraints.
	for _, con := range g.warmCold {
		address := getAddress(con.key)
		if _, isPresent := account.Balance[address]; !isPresent {
			account.Balance[address] = RandU256(rnd)
			account.Code[address] = randCode(rnd)
		}
		account.SetWarm(address, con.warm)
	}

	// Some random entries.
	for i, max := 0, rnd.Intn(5); i < max; i++ {
		address := getUnusedAddress()
		account.Balance[address] = RandU256(rnd)
		if rnd.Intn(2) == 1 {
			account.Code[address] = randCode(rnd)
		}
		account.SetWarm(address, rnd.Intn(2) == 1)
	}

	// Add own account address
	account.Balance[accountAddress] = RandU256(rnd)
	account.Code[accountAddress] = randCode(rnd)

	return account, nil
}
