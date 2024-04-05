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

func RandomBytes(rnd *rand.Rand) Bytes {
	const (
		expectedSize = 200
		maxSize      = 2000
	)
	rand := rnd.ExpFloat64()
	size := int(rand * expectedSize)
	if size > maxSize {
		size = maxSize
	}
	return RandomBytesOfSize(rnd, size)
}

func RandomBytesOfSize(rnd *rand.Rand, size int) Bytes {
	data := make([]byte, size)
	rnd.Read(data)
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
