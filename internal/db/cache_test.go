package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDBCachesAndClosesSSTables(t *testing.T) {
	dir := t.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1
	opts.SparseIndexGap = 1

	database, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := database.Put([]byte("a"), []byte("1")); err != nil {
		t.Fatalf("put a: %v", err)
	}
	if err := database.Put([]byte("b"), []byte("2")); err != nil {
		t.Fatalf("put b: %v", err)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("remove data dir after close: %v", err)
	}
}

func TestDBReadsFromCachedSSTables(t *testing.T) {
	dir := t.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1
	opts.SparseIndexGap = 1

	database, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := database.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := database.Put([]byte("key2"), []byte("value2")); err != nil {
		t.Fatalf("put: %v", err)
	}

	for i := 0; i < 20; i++ {
		v, ok, err := database.Get([]byte("key1"))
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if !ok || string(v) != "value1" {
			t.Fatalf("unexpected result: ok=%v value=%q", ok, v)
		}
	}

	_ = filepath.Join
}