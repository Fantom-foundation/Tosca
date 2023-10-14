package cti

//go:generate mockgen -source host.go -destination host_mocks.go -package cti

import "github.com/holiman/uint256"

type Host interface {
	GetStorage(uint256.Int) uint256.Int
	SetStorage(uint256.Int, uint256.Int)

	Call(
		gasSent uint256.Int,
		address uint256.Int,
		value uint256.Int,
		message []byte,
	) (
		success bool,
		gasLeft uint256.Int,
		result []byte,
	)
}
