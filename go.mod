module github.com/Fantom-foundation/Tosca

go 1.21

require (
	github.com/dsnet/golib/unitconv v1.0.2
	github.com/ethereum/evmc/v10 v10.0.0
	github.com/ethereum/go-ethereum v1.10.25
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/holiman/uint256 v1.2.4
	github.com/urfave/cli/v2 v2.25.7
	go.uber.org/mock v0.4.0
	golang.org/x/crypto v0.22.0
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa
	pgregory.net/rand v1.0.2
)

require (
	github.com/bits-and-blooms/bitset v1.10.0 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/crate-crypto/go-kzg-4844 v1.0.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/supranational/blst v0.3.11 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum-sonic v0.0.0-20240529085303-2400937cc3b1

replace github.com/ethereum/evmc/v10 => ./third_party/evmc
