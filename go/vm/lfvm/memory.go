package lfvm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

type Memory struct {
	store             []byte
	total_memory_cost uint64
}

func NewMemory() *Memory {
	return &Memory{}
}

func toValidMemorySize(size uint64) uint64 {
	// Target size seems to need to be a multiple of 32
	return ((size + 31) / 32) * 32
}

func (m *Memory) ExpansionCosts(size uint64) uint64 {
	if m.Len() >= size {
		return 0
	}
	size = toValidMemorySize(size)
	memory_size_word := uint64((size + 31) / 32)
	new_costs := (memory_size_word*memory_size_word)/512 + (3 * memory_size_word)
	fee := new_costs - m.total_memory_cost
	return fee
}

func (m *Memory) EnsureCapacity(offset, size uint64, c *context) error {
	if size <= 0 {
		return nil
	}
	needed := offset + size
	if m.Len() < needed {
		needed = toValidMemorySize(needed)
		fee := m.ExpansionCosts(needed)
		if !c.UseGas(fee) {
			return vm.ErrOutOfGas
		}
		m.total_memory_cost += fee
		m.store = append(m.store, make([]byte, needed-m.Len())...)
	}
	return nil
}

func (m *Memory) EnsureCapacityWithoutGas(size uint64, c *context) {
	if size <= 0 {
		return
	}
	if m.Len() < size {
		size = toValidMemorySize(size)
		fee := m.ExpansionCosts(size)
		m.total_memory_cost += fee
		m.store = append(m.store, make([]byte, size-m.Len())...)
	}
}

func (m *Memory) Len() uint64 {
	return uint64(len(m.store))
}

func (m *Memory) SetByte(offset uint64, value byte) {
	if m.Len() < offset+1 {
		panic(fmt.Sprintf("memory to small, size %d, attempted to write at position %d", m.Len(), offset))
	}
	m.store[offset] = value
}

func (m *Memory) SetWord(offset uint64, value *uint256.Int) {
	if m.Len() < offset+32 {
		panic(fmt.Sprintf("memory to small, size %d, attempted to write 32 byte at position %d", m.Len(), offset))
	}

	// Inlining and unrolling value.WriteToSlice(..) lead to a 7x speedup
	dest := m.store[offset : offset+32]
	dest[31] = byte(value[0])
	dest[30] = byte(value[0] >> 8)
	dest[29] = byte(value[0] >> 16)
	dest[28] = byte(value[0] >> 24)
	dest[27] = byte(value[0] >> 32)
	dest[26] = byte(value[0] >> 40)
	dest[25] = byte(value[0] >> 48)
	dest[24] = byte(value[0] >> 56)

	dest[23] = byte(value[1])
	dest[22] = byte(value[1] >> 8)
	dest[21] = byte(value[1] >> 16)
	dest[20] = byte(value[1] >> 24)
	dest[19] = byte(value[1] >> 32)
	dest[18] = byte(value[1] >> 40)
	dest[17] = byte(value[1] >> 48)
	dest[16] = byte(value[1] >> 56)

	dest[15] = byte(value[2])
	dest[14] = byte(value[2] >> 8)
	dest[13] = byte(value[2] >> 16)
	dest[12] = byte(value[2] >> 24)
	dest[11] = byte(value[2] >> 32)
	dest[10] = byte(value[2] >> 40)
	dest[9] = byte(value[2] >> 48)
	dest[8] = byte(value[2] >> 56)

	dest[7] = byte(value[3])
	dest[6] = byte(value[3] >> 8)
	dest[5] = byte(value[3] >> 16)
	dest[4] = byte(value[3] >> 24)
	dest[3] = byte(value[3] >> 32)
	dest[2] = byte(value[3] >> 40)
	dest[1] = byte(value[3] >> 48)
	dest[0] = byte(value[3] >> 56)
}

func (m *Memory) Set(offset, size uint64, value []byte) {
	if size > 0 {
		if offset+size > m.Len() {
			panic(fmt.Sprintf("memory to small, size %d, attempted to write %d bytes at %d", m.Len(), size, offset))
		}
		copy(m.store[offset:offset+size], value)
	}
}

func (m *Memory) CopyWord(offset uint64, trg *uint256.Int) {
	if m.Len() < offset+32 {
		panic(fmt.Sprintf("memory to small, size %d, attempted to read 32 byte at position %d", m.Len(), offset))
	}
	trg.SetBytes32(m.store[offset : offset+32])
}

// Copies data from the memory to the given slice.
func (m *Memory) CopyData(offset uint64, trg []byte) {
	if m.Len() < offset {
		copy(trg, make([]byte, len(trg)))
		return
	}

	// Copy what is available.
	covered := copy(trg, m.store[offset:])

	// Pad the rest
	if covered < len(trg) {
		copy(trg[covered:], make([]byte, len(trg)-covered))
	}
}

func (m *Memory) GetSlice(offset, size uint64) []byte {
	if size == 0 {
		return nil
	}

	if m.Len() > offset {
		return m.store[offset : offset+size]
	}

	return nil
}

func (m *Memory) Data() []byte {
	return m.store
}

func (m *Memory) Print() {
	fmt.Printf("### mem %d bytes ###\n", len(m.store))
	if len(m.store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(m.store); i += 32 {
			fmt.Printf("%03d: % x\n", addr, m.store[i:i+32])
			addr++
		}
		if len(m.store)%32 != 0 {
			fmt.Printf("%03d: % x\n", addr, m.store[len(m.store)/32*32:])
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("####################")
}
