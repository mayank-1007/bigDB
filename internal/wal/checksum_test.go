package wal

import (
	"os"
	"path/filepath"
	"testing"

	"bigtable/internal/record"
)

func TestWALChecksumDetectsCorruption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal.log")

	w, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := w.Append(record.NewPut([]byte("k1"), []byte("v1"), 1)); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if len(data) < 12 {
		t.Fatal("wal too small for corruption test")
	}

	data[10] ^= 0xFF

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("rewrite corrupt wal: %v", err)
	}

	w2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer w2.Close()

	err = w2.ReplayAll(func(r record.Record) error { return nil })
	if err == nil {
		t.Fatal("expected checksum error, got nil")
	}
}