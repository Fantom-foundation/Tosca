package vm

// These are the officially exported EVM variants provided
// by this package. Other VM implementations may be present
// in this repository, but they are not (yet) intended for
// external use.

import (
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)
