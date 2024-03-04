module github.com/Fantom-foundation/Tosca

go 1.21

require (
	github.com/ethereum/evmc/v10 v10.0.0
	github.com/ethereum/go-ethereum v1.10.25
	github.com/golang/mock v1.6.0
	github.com/holiman/uint256 v1.2.0
	github.com/urfave/cli/v2 v2.10.2
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d
	pgregory.net/rand v1.0.2
)

require (
	github.com/Fantom-foundation/Substate v0.0.0-20230224090651-4c8c024214f4 // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/VictoriaMetrics/fastcache v1.6.0 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.uber.org/mock v0.4.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum-substate v1.1.1-0.20240227132411-c08de2b3341f

replace github.com/ethereum/evmc/v10 => ./third_party/evmc
