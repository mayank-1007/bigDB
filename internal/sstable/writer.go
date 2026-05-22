package sstable

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"

	"bigtable/internal/bloom"
	"bigtable/internal/record"
)

func WriteSnapshot(path string, snapshot map[string]record.Record, sparseGap int) error {
	if sparseGap <= 0 {
		sparseGap = 10
	}

	tmpPath := path + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create sstable tmp: %w", err)
	}

	keys := make([]string, 0, len(snapshot))
	for k := range snapshot {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	indexEntries := make([]SparseIndexEntry, 0, len(keys))

	// Write every record exactly as it exists in the snapshot.
	// Tombstones are first-class records and must be persisted too.
	for i, k := range keys {
		rec := snapshot[k]

		offset, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("get write offset: %w", err)
		}

		if i%sparseGap == 0 {
			keyCopy := append([]byte(nil), rec.Key...)
			indexEntries = append(indexEntries, SparseIndexEntry{
				Key:    keyCopy,
				Offset: uint64(offset),
			})
		}

		payload, err := rec.Encode()
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("encode record: %w", err)
		}

		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payload))|checksumFlag)

		var crcBuf [4]byte
		binary.BigEndian.PutUint32(crcBuf[:], crc32.ChecksumIEEE(payload))

		if err := writeFull(f, lenBuf[:]); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write record length: %w", err)
		}
		if err := writeFull(f, payload); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write record payload: %w", err)
		}
		if err := writeFull(f, crcBuf[:]); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write record crc: %w", err)
		}
	}

	bf := bloom.NewForKeys(len(keys))
	for _, k := range keys {
		bf.Add([]byte(k))
	}

	bloomOffset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("get bloom offset: %w", err)
	}

	bloomBytes := bf.MarshalBinary()
	if err := writeFull(f, bloomBytes); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write bloom filter: %w", err)
	}

	indexOffset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("get index offset: %w", err)
	}

	for _, entry := range indexEntries {
		var keyLenBuf [4]byte
		binary.BigEndian.PutUint32(keyLenBuf[:], uint32(len(entry.Key)))

		var offsetBuf [8]byte
		binary.BigEndian.PutUint64(offsetBuf[:], entry.Offset)

		if err := writeFull(f, keyLenBuf[:]); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write index key len: %w", err)
		}
		if err := writeFull(f, entry.Key); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write index key: %w", err)
		}
		if err := writeFull(f, offsetBuf[:]); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write index offset: %w", err)
		}
	}

	footer := make([]byte, newFooterSize)
	binary.BigEndian.PutUint64(footer[0:8], uint64(indexOffset))
	binary.BigEndian.PutUint32(footer[8:12], uint32(len(indexEntries)))
	binary.BigEndian.PutUint32(footer[12:16], uint32(len(keys)))
	binary.BigEndian.PutUint32(footer[16:20], uint32(sparseGap))
	binary.BigEndian.PutUint64(footer[20:28], uint64(bloomOffset))
	binary.BigEndian.PutUint32(footer[28:32], uint32(len(bloomBytes)))
	copy(footer[32:40], []byte(newMagicValue))

	if err := writeFull(f, footer); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write footer: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sync sstable: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close sstable tmp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename sstable tmp: %w", err)
	}

	return nil
}

func writeFull(f *os.File, data []byte) error {
	for len(data) > 0 {
		n, err := f.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}