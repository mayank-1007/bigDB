package sstable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"

	"bigtable/internal/bloom"
	"bigtable/internal/record"
)

func Open(path string) (*File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open sstable: %w", err)
	}

	level, seq, ok := parseLevelAndSeq(path)
	if !ok {
		_ = fh.Close()
		return nil, fmt.Errorf("parse sstable name: %w", ErrCorruptFooter)
	}

	info, err := fh.Stat()
	if err != nil {
		_ = fh.Close()
		return nil, fmt.Errorf("stat sstable: %w", err)
	}

	var footer Footer
	switch {
	case info.Size() >= newFooterSize:
		footerBytes := make([]byte, newFooterSize)
		_, err = fh.ReadAt(footerBytes, info.Size()-newFooterSize)
		if err != nil {
			_ = fh.Close()
			return nil, fmt.Errorf("read footer: %w", err)
		}

		magic := string(bytes.TrimRight(footerBytes[32:40], "\x00"))
		if magic == newMagicValue {
			footer = Footer{
				IndexOffset: binary.BigEndian.Uint64(footerBytes[0:8]),
				IndexCount:  binary.BigEndian.Uint32(footerBytes[8:12]),
				RecordCount: binary.BigEndian.Uint32(footerBytes[12:16]),
				SparseGap:   binary.BigEndian.Uint32(footerBytes[16:20]),
				BloomOffset: binary.BigEndian.Uint64(footerBytes[20:28]),
				BloomSize:   binary.BigEndian.Uint32(footerBytes[28:32]),
				Magic:       newMagicValue,
			}
		} else {
			legacyFooterBytes := make([]byte, legacyFooterSize)
			_, err = fh.ReadAt(legacyFooterBytes, info.Size()-legacyFooterSize)
			if err != nil {
				_ = fh.Close()
				return nil, fmt.Errorf("read legacy footer: %w", err)
			}
			if string(legacyFooterBytes[20:28]) != legacyMagicValue {
				_ = fh.Close()
				return nil, ErrCorruptFooter
			}
			footer = Footer{
				IndexOffset: binary.BigEndian.Uint64(legacyFooterBytes[0:8]),
				IndexCount:  binary.BigEndian.Uint32(legacyFooterBytes[8:12]),
				RecordCount: binary.BigEndian.Uint32(legacyFooterBytes[12:16]),
				SparseGap:   binary.BigEndian.Uint32(legacyFooterBytes[16:20]),
				Magic:       legacyMagicValue,
			}
		}
	default:
		_ = fh.Close()
		return nil, ErrCorruptFooter
	}

	index, err := readIndex(fh, footer)
	if err != nil {
		_ = fh.Close()
		return nil, err
	}

	var bf *bloom.Filter
	if footer.Magic == newMagicValue && footer.BloomOffset > 0 && footer.BloomSize > 0 {
		bf, err = readBloom(fh, footer)
		if err != nil {
			_ = fh.Close()
			return nil, err
		}
	}

	return &File{
		path:   path,
		fh:     fh,
		index:  index,
		footer: footer,
		bloom:  bf,
		level:  level,
		seq:    seq,
	}, nil
}

func (f *File) Get(key []byte) (record.Record, bool, error) {
	if f == nil || f.fh == nil {
		return record.Record{}, false, fmt.Errorf("sstable closed")
	}

	if f.bloom != nil && !f.bloom.MightContain(key) {
		return record.Record{}, false, nil
	}

	startOffset := uint64(0)
	endOffset := f.footer.IndexOffset
	if f.footer.BloomOffset > 0 {
		endOffset = f.footer.BloomOffset
	}

	if len(f.index) > 0 {
		pos := sort.Search(len(f.index), func(i int) bool {
			return bytes.Compare(f.index[i].Key, key) > 0
		})

		candidate := pos - 1
		if candidate >= 0 {
			startOffset = f.index[candidate].Offset
			if candidate+1 < len(f.index) {
				endOffset = f.index[candidate+1].Offset
			}
		} else {
			startOffset = 0
			endOffset = f.index[0].Offset
		}
	}

	offset := startOffset
	for offset < endOffset {
		rec, nextOffset, err := readRecordAt(f.fh, offset, endOffset)
		if err != nil {
			return record.Record{}, false, err
		}

		cmp := bytes.Compare(rec.Key, key)
		if cmp == 0 {
			if rec.IsDeleted {
				return rec, true, nil
			}
			return rec, true, nil
		}
		if cmp > 0 {
			return record.Record{}, false, nil
		}

		offset = nextOffset
	}

	return record.Record{}, false, nil
}

func (f *File) All() ([]record.Record, error) {
	if f == nil || f.fh == nil {
		return nil, fmt.Errorf("sstable closed")
	}

	records := make([]record.Record, 0)
	offset := uint64(0)
	limit := f.footer.IndexOffset
	if f.footer.BloomOffset > 0 {
		limit = f.footer.BloomOffset
	}

	for offset < limit {
		rec, nextOffset, err := readRecordAt(f.fh, offset, limit)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		records = append(records, rec)
		offset = nextOffset
	}

	return records, nil
}

func readBloom(fh *os.File, footer Footer) (*bloom.Filter, error) {
	data := make([]byte, footer.BloomSize)

	_, err := fh.ReadAt(data, int64(footer.BloomOffset))
	if err != nil {
		return nil, fmt.Errorf("read bloom filter: %w", err)
	}

	filter, ok := bloom.UnmarshalBinary(data)
	if !ok {
		return nil, ErrCorruptFooter
	}

	return filter, nil
}

func readIndex(fh *os.File, footer Footer) ([]SparseIndexEntry, error) {
	if footer.IndexCount == 0 {
		return nil, nil
	}

	index := make([]SparseIndexEntry, 0, footer.IndexCount)
	offset := int64(footer.IndexOffset)

	for i := uint32(0); i < footer.IndexCount; i++ {
		var keyLenBuf [4]byte
		_, err := fh.ReadAt(keyLenBuf[:], offset)
		if err != nil {
			return nil, fmt.Errorf("read index key len: %w", err)
		}
		offset += 4

		keyLen := binary.BigEndian.Uint32(keyLenBuf[:])
		key := make([]byte, keyLen)
		_, err = fh.ReadAt(key, offset)
		if err != nil {
			return nil, fmt.Errorf("read index key: %w", err)
		}
		offset += int64(keyLen)

		var offBuf [8]byte
		_, err = fh.ReadAt(offBuf[:], offset)
		if err != nil {
			return nil, fmt.Errorf("read index offset: %w", err)
		}
		offset += 8

		index = append(index, SparseIndexEntry{
			Key:    key,
			Offset: binary.BigEndian.Uint64(offBuf[:]),
		})
	}

	return index, nil
}

func readRecordAt(fh *os.File, offset uint64, limit uint64) (record.Record, uint64, error) {
	if offset+4 > limit {
		return record.Record{}, 0, io.EOF
	}

	var lenBuf [4]byte
	_, err := fh.ReadAt(lenBuf[:], int64(offset))
	if err != nil {
		return record.Record{}, 0, fmt.Errorf("read record length: %w", err)
	}

	rawLen := binary.BigEndian.Uint32(lenBuf[:])
	hasCRC := rawLen&checksumFlag != 0
	payloadLen := rawLen &^ checksumFlag

	total := uint64(4) + uint64(payloadLen)
	if hasCRC {
		total += 4
	}
	if offset+total > limit {
		return record.Record{}, 0, ErrCorruptRecord
	}

	payload := make([]byte, payloadLen)
	_, err = fh.ReadAt(payload, int64(offset+4))
	if err != nil {
		return record.Record{}, 0, fmt.Errorf("read record payload: %w", err)
	}

	if hasCRC {
		var crcBuf [4]byte
		_, err = fh.ReadAt(crcBuf[:], int64(offset+4+uint64(payloadLen)))
		if err != nil {
			return record.Record{}, 0, fmt.Errorf("read record crc: %w", err)
		}

		want := binary.BigEndian.Uint32(crcBuf[:])
		got := crc32.ChecksumIEEE(payload)
		if want != got {
			return record.Record{}, 0, ErrCorruptRecord
		}
	}

	rec, err := record.Decode(payload)
	if err != nil {
		return record.Record{}, 0, fmt.Errorf("decode record: %w", err)
	}

	return rec, offset + total, nil
}