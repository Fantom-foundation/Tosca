package gen

import (
	"fmt"
	"slices"
	"strings"

	"pgregory.net/rand"
)

type ReturnDateGenerator struct {
	sizes []int
}

func NewReturnDateGenerator() *ReturnDateGenerator {
	return &ReturnDateGenerator{}
}

func (r *ReturnDateGenerator) Generate(rnd *rand.Rand) ([]byte, error) {

	// Pick a size
	size := 0
	if len(r.sizes) > 1 {
		return nil, fmt.Errorf("%w, multiple conflicting sizes defined: %v", ErrUnsatisfiable, r.sizes)
	} else if len(r.sizes) == 1 {
		size = r.sizes[0]
		if size < 0 {
			return nil, fmt.Errorf("%w, can not produce return data with negative size %d", ErrUnsatisfiable, size)
		}
	} else {
		size = rnd.Intn(50)
	}

	resultReturnData := make([]byte, size)
	_, err := rnd.Read(resultReturnData)
	if err != nil {
		return nil, err
	}
	return resultReturnData, nil
}

func (r *ReturnDateGenerator) Clone() *ReturnDateGenerator {
	newGen := NewReturnDateGenerator()
	newGen.sizes = slices.Clone(r.sizes)
	return newGen
}

func (r *ReturnDateGenerator) Restore(other *ReturnDateGenerator) {
	if r == other {
		return
	}
	r.sizes = slices.Clone(other.sizes)
}

func (r *ReturnDateGenerator) String() string {
	var sizes []string

	for _, size := range r.sizes {
		sizes = append(sizes, fmt.Sprint(size))
	}
	str := "{sizes: " + strings.Join(sizes, ", ") + "}"
	return str
}

func (r *ReturnDateGenerator) SetSize(size int) {
	if !slices.Contains(r.sizes, size) {
		r.sizes = append(r.sizes, size)
	}
}
