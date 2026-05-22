package db

import "testing"

func TestDBWALRotationAndRecovery(t *testing.T) {
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

	if err := database.Put([]byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("put k1 v1: %v", err)
	}
	if err := database.Put([]byte("k1"), []byte("v2")); err != nil {
		t.Fatalf("put k1 v2: %v", err)
	}
	if err := database.Put([]byte("k2"), []byte("v3")); err != nil {
		t.Fatalf("put k2 v3: %v", err)
	}

	if err := database.Barrier(); err != nil {
		t.Fatal(err)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := Open(opts)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	value, ok, err := reopened.Get([]byte("k1"))
	if err != nil {
		t.Fatalf("get k1: %v", err)
	}
	if !ok || string(value) != "v2" {
		t.Fatalf("unexpected k1 value: ok=%v value=%q", ok, value)
	}

	value, ok, err = reopened.Get([]byte("k2"))
	if err != nil {
		t.Fatalf("get k2: %v", err)
	}
	if !ok || string(value) != "v3" {
		t.Fatalf("unexpected k2 value: ok=%v value=%q", ok, value)
	}
}