# We've Moved!

This repository has found a new home! 

You can now find everything at [https://github.com/0xsoniclabs/tosca](https://github.com/0xsoniclabs/tosca). 

Please update your bookmarks and any links you have to this repo.

# Sonic-VM (Tosca)

Sonic-VM (Tosca) is licensed under the [Business Source License (BSL)](LICENSE). The BSL prohibits Sonic-VM (Tosca) from being used in production by any other project. Anyone can view or use the licensed code for internal or testing purposes. Still, commercial use is limited to Fantom Foundation and Fantom users operating on Fantom's mainnet and/or testnet.

Tosca includes a high-performance C++ and a Go implementation of the Ethereum Virtual Machine (EVM) as well as testing infrastructure. This project aims to create fast and fully tested virtual machines using Conformance Testing. Conformance Testing checks whether an EVM specification expressed as constraints/quantor-free predict logic complies with the latest go-ethereum implementation. Tosca's Go version translates the original contract bytecode into its long format instruction set for faster execution. The C++ implementation is a high-performance/highly-tuned implementation of the EVM.

For detailed information regarding requirements, building, testing and profiling please have a look at the provided [BUILD](BUILD.md) file. 
