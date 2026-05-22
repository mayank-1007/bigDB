package bloom

import (
	"encoding/binary"
	"hash/fnv"
)

type Filter struct {
	BitCount  uint32
	HashCount uint32
	Bits      []byte
}

func New(bitCount, hashCount uint32) *Filter {
	if bitCount < 8 {
		bitCount = 8
	}
	if hashCount == 0 {
		hashCount = 1
	}
	return &Filter{
		BitCount:  bitCount,
		HashCount: hashCount,
		Bits:      make([]byte, (bitCount+7)/8),
	}
}

func NewForKeys(keyCount int) *Filter {
	// Small but effective default: ~16 bits per key, minimum 1024 bits.
	bitCount := uint32(1024)
	if keyCount > 0 {
		bitCount = uint32(keyCount * 16)
		if bitCount < 1024 {
			bitCount = 1024
		}
	}
	return New(bitCount, 7)
}

func (f *Filter) Add(key []byte) {
	h1, h2 := hashPair(key)

	if h2 == 0 {
		h2 = 1
	}

	for i := uint32(0); i < f.HashCount; i++ {
		pos := (h1 + uint64(i)*h2) % uint64(f.BitCount)
		byteIdx := pos / 8
		bitMask := byte(1 << (pos % 8))
		f.Bits[byteIdx] |= bitMask
	}
}

func (f *Filter) MightContain(key []byte) bool {
	if f == nil || f.BitCount == 0 || len(f.Bits) == 0 {
		return true
	}

	h1, h2 := hashPair(key)
	if h2 == 0 {
		h2 = 1
	}

	for i := uint32(0); i < f.HashCount; i++ {
		pos := (h1 + uint64(i)*h2) % uint64(f.BitCount)
		byteIdx := pos / 8
		bitMask := byte(1 << (pos % 8))
		if (f.Bits[byteIdx] & bitMask) == 0 {
			return false
		}
	}
	return true
}

func (f *Filter) MarshalBinary() []byte {
	out := make([]byte, 8+len(f.Bits))
	binary.BigEndian.PutUint32(out[0:4], f.BitCount)
	binary.BigEndian.PutUint32(out[4:8], f.HashCount)
	copy(out[8:], f.Bits)
	return out
}

func UnmarshalBinary(data []byte) (*Filter, bool) {
	if len(data) < 8 {
		return nil, false
	}

	bitCount := binary.BigEndian.Uint32(data[0:4])
	hashCount := binary.BigEndian.Uint32(data[4:8])
	bits := make([]byte, len(data[8:]))
	copy(bits, data[8:])

	if bitCount == 0 || hashCount == 0 {
		return nil, false
	}

	return &Filter{
		BitCount:  bitCount,
		HashCount: hashCount,
		Bits:      bits,
	}, true
}

func hashPair(key []byte) (uint64, uint64) {
	h1 := fnv.New64a()
	_, _ = h1.Write(key)
	sum1 := h1.Sum64()

	h2 := fnv.New32a()
	_, _ = h2.Write(key)
	sum2 := uint64(h2.Sum32())

	// Avoid zero-step secondary hash.
	if sum2 == 0 {
		sum2 = 0x9e3779b97f4a7c15
	}
	return sum1, sum2
}