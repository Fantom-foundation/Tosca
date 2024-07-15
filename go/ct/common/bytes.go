// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package common

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"pgregory.net/rand"
)

// Bytes is an immutable slice of bytes that can be trivially cloned.
type Bytes struct {
	data string
}

func NewBytes(data []byte) Bytes {
	return Bytes{data: string(data)}
}

func RandomBytes(rnd *rand.Rand, maxSize int) Bytes {
	const expectedSize = 200
	rand := rnd.ExpFloat64()
	size := int(rand * expectedSize)
	if size > maxSize {
		size = maxSize
	}
	return RandomBytesOfSize(rnd, size)
}

func RandomBytesOfSize(rnd *rand.Rand, size int) Bytes {
	data := make([]byte, size)
	_, _ = rnd.Read(data) // rnd.Read never returns an error
	return NewBytes(data)
}

func (b Bytes) ToBytes() []byte {
	return []byte(b.data)
}

func (b Bytes) String() string {
	return fmt.Sprintf("0x%x", b.data)
}

func (b Bytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%x", b.ToBytes()))
}

func (b *Bytes) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	data, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	b.data = string(data)
	return nil
}

func (b Bytes) Length() int {
	return len(b.data)
}

func (b Bytes) Get(start, end uint64) []byte {
	return []byte(b.data)[start:end]
}
