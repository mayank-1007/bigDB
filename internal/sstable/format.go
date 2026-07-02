package sstable

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"bigdb/internal/bloom"
)

var (
	ErrCorruptFooter = errors.New("corrupt sstable footer")
	ErrCorruptIndex  = errors.New("corrupt sstable index")
	ErrCorruptRecord = errors.New("corrupt sstable record")
)

const (
	legacyMagicValue = "BTSSTB1"
	newMagicValue    = "BTSSTB2X"
	legacyFooterSize = 28
	newFooterSize    = 40

	checksumFlag = 1 << 31
)

type SparseIndexEntry struct {
	Key    []byte
	Offset uint64
}

type Footer struct {
	IndexOffset uint64
	IndexCount  uint32
	RecordCount uint32
	SparseGap   uint32

	BloomOffset uint64
	BloomSize   uint32

	Magic string
}

type File struct {
	path   string
	fh     *os.File
	index  []SparseIndexEntry
	footer Footer
	bloom  *bloom.Filter

	level uint32
	seq   uint64
}

func (f *File) Path() string {
	if f == nil {
		return ""
	}
	return f.path
}

func (f *File) Level() uint32 {
	if f == nil {
		return 0
	}
	return f.level
}

func (f *File) Sequence() uint64 {
	if f == nil {
		return 0
	}
	return f.seq
}

func (f *File) Close() error {
	if f == nil || f.fh == nil {
		return nil
	}
	err := f.fh.Close()
	f.fh = nil
	return err
}

func parseLevelAndSeq(path string) (uint32, uint64, bool) {
	name := filepathBase(path)

	if strings.HasPrefix(name, "l") && strings.Contains(name, "-sst-") && strings.HasSuffix(name, ".sst") {
		parts := strings.SplitN(name, "-sst-", 2)
		if len(parts) != 2 {
			return 0, 0, false
		}

		levelRaw := strings.TrimPrefix(parts[0], "l")
		seqRaw := strings.TrimSuffix(parts[1], ".sst")

		level64, err := strconv.ParseUint(levelRaw, 10, 32)
		if err != nil {
			return 0, 0, false
		}
		seq, err := strconv.ParseUint(seqRaw, 10, 64)
		if err != nil {
			return 0, 0, false
		}

		return uint32(level64), seq, true
	}

	// Legacy format: sst-<seq>.sst
	if strings.HasPrefix(name, "sst-") && strings.HasSuffix(name, ".sst") {
		raw := strings.TrimSuffix(strings.TrimPrefix(name, "sst-"), ".sst")
		seq, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		return 0, seq, true
	}

	return 0, 0, false
}

func filepathBase(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i >= 0 {
		return path[i+1:]
	}
	return path
}
