package record

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidEncoding = errors.New("invalid record encoding")
)

type Record struct {
	Key       []byte
	Value     []byte
	Timestamp int64
	IsDeleted bool
}

func NewPut(key, value []byte, timestamp int64) Record {
	return Record{
		Key:       append([]byte(nil), key...),
		Value:     append([]byte(nil), value...),
		Timestamp: timestamp,
		IsDeleted: false,
	}
}

func NewDelete(key []byte, timestamp int64) Record {
	return Record{
		Key:       append([]byte(nil), key...),
		Value:     nil,
		Timestamp: timestamp,
		IsDeleted: true,
	}
}

// Payload layout:
// [keyLen uint32][valueLen uint32][timestamp int64][isDeleted byte][key bytes][value bytes]
func (r Record) Encode() ([]byte, error) {
	keyLen := len(r.Key)
	valueLen := len(r.Value)

	total := 4 + 4 + 8 + 1 + keyLen + valueLen
	out := make([]byte, total)

	binary.BigEndian.PutUint32(out[0:4], uint32(keyLen))
	binary.BigEndian.PutUint32(out[4:8], uint32(valueLen))
	binary.BigEndian.PutUint64(out[8:16], uint64(r.Timestamp))

	if r.IsDeleted {
		out[16] = 1
	} else {
		out[16] = 0
	}

	copy(out[17:17+keyLen], r.Key)
	copy(out[17+keyLen:], r.Value)

	return out, nil
}

func Decode(data []byte) (Record, error) {
	if len(data) < 17 {
		return Record{}, fmt.Errorf("%w: too short", ErrInvalidEncoding)
	}

	keyLen := binary.BigEndian.Uint32(data[0:4])
	valueLen := binary.BigEndian.Uint32(data[4:8])
	timestamp := int64(binary.BigEndian.Uint64(data[8:16]))
	isDeleted := data[16] == 1

	expected := 17 + int(keyLen) + int(valueLen)
	if len(data) != expected {
		return Record{}, fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidEncoding, expected, len(data))
	}

	key := make([]byte, int(keyLen))
	copy(key, data[17:17+int(keyLen)])

	value := make([]byte, int(valueLen))
	copy(value, data[17+int(keyLen):])

	return Record{
		Key:       key,
		Value:     value,
		Timestamp: timestamp,
		IsDeleted: isDeleted,
	}, nil
}