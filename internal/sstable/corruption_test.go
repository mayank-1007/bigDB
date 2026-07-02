package sstable

import (
	"os"
	"path/filepath"
	"testing"

	"bigdb/internal/record"
)

func TestSSTableChecksumDetectsCorruption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "l0-sst-00000000000000000001.sst")

	snapshot := map[string]record.Record{
		"a": record.NewPut([]byte("a"), []byte("value-a"), 1),
		"b": record.NewPut([]byte("b"), []byte("value-b"), 2),
	}

	if err := WriteSnapshot(path, snapshot, 1); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sstable: %v", err)
	}

	// Corrupt a byte inside the first record area.
	if len(data) < 20 {
		t.Fatal("sstable too small for corruption test")
	}
	data[12] ^= 0xFF

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("rewrite corrupted sstable: %v", err)
	}

	file, err := Open(path)
	if err != nil {
		t.Fatalf("open corrupted sstable: %v", err)
	}
	defer file.Close()

	_, _, err = file.Get([]byte("a"))
	if err == nil {
		t.Fatal("expected corruption error, got nil")
	}
}
