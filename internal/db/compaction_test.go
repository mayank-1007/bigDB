package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDBCompactionRemovesDuplicatesAndTombstones(t *testing.T) {
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
	if err := database.Barrier(); err != nil {
		t.Fatal(err)
	}

	defer database.Close()

	if err := database.Put([]byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("put k1 v1: %v", err)
	}
	if err := database.Put([]byte("k1"), []byte("v2")); err != nil {
		t.Fatalf("put k1 v2: %v", err)
	}
	if err := database.Put([]byte("k2"), []byte("v3")); err != nil {
		t.Fatalf("put k2 v3: %v", err)
	}
	if err := database.Delete([]byte("k2")); err != nil {
		t.Fatalf("delete k2: %v", err)
	}

	if err := database.Compact(); err != nil {
		t.Fatalf("compact: %v", err)
	}

	database.wg.Wait()

	value, ok, err := database.Get([]byte("k1"))
	if err != nil {
		t.Fatalf("get k1: %v", err)
	}
	if !ok || string(value) != "v2" {
		t.Fatalf("unexpected k1 value: ok=%v value=%q", ok, value)
	}

	_, ok, err = database.Get([]byte("k2"))
	if err != nil {
		t.Fatalf("get k2: %v", err)
	}
	if ok {
		t.Fatal("expected k2 to be deleted")
	}

	sstDir := filepath.Join(dir, opts.SSTableDirName)
	entries, err := os.ReadDir(sstDir)
	if err != nil {
		t.Fatalf("read sst dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 SSTable after compaction, got %d", len(entries))
	}
}