package cti

//go:generate mockgen -source host.go -destination host_mocks.go -package cti

import "github.com/holiman/uint256"

type Host interface {
	GetStorage(uint256.Int) uint256.Int
	SetStorage(uint256.Int, uint256.Int)
}
