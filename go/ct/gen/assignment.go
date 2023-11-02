package gen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Fantom-foundation/Tosca/go/ct"
)

type Variable string

func (v Variable) String() string {
	return "$" + string(v)
}

type Assignment map[Variable]ct.U256

func (a Assignment) String() string {
	if a == nil {
		return "{}"
	}
	keys := make([]Variable, 0, len(a))
	for key := range a {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	entries := make([]string, 0, len(a))
	for _, key := range keys {
		entries = append(entries, fmt.Sprintf("%s->%v", string(key), a[key]))
	}
	return "{" + strings.Join(entries, ",") + "}"
}
