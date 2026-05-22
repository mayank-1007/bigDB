package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackgroundFlushDoesNotBreakReads(t *testing.T) {
	dir := t.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 2
	opts.SparseIndexGap = 1

	database, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := database.Put([]byte("a"), []byte("1")); err != nil {
		t.Fatalf("put a: %v", err)
	}
	if err := database.Put([]byte("b"), []byte("2")); err != nil {
		t.Fatalf("put b: %v", err)
	}
	if err := database.Put([]byte("c"), []byte("3")); err != nil {
		t.Fatalf("put c: %v", err)
	}

	value, ok, err := database.Get([]byte("a"))
	if err != nil {
		t.Fatalf("get a: %v", err)
	}
	if !ok || string(value) != "1" {
		t.Fatalf("unexpected read for a: ok=%v value=%q", ok, value)
	}

	sstDir := filepath.Join(dir, opts.SSTableDirName)
	found := false
	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(sstDir)
		if err != nil {
			t.Fatalf("read sst dir: %v", err)
		}
		if len(entries) > 0 {
			found = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !found {
		t.Fatal("expected background flush to create an sstable")
	}
}