package db

import "testing"

func TestDBPutGetRestartRecovery(t *testing.T) {
	dir := t.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"

	database, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := database.Put([]byte("name"), []byte("mayank")); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := Open(opts)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	value, ok, err := reopened.Get([]byte("name"))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected recovered key")
	}
	if string(value) != "mayank" {
		t.Fatalf("unexpected value: %s", value)
	}
}

func TestDBFlushToSSTableAndReadBack(t *testing.T) {
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
		t.Fatalf("unexpected value for a: %q ok=%v", value, ok)
	}

	value, ok, err = database.Get([]byte("c"))
	if err != nil {
		t.Fatalf("get c: %v", err)
	}
	if !ok || string(value) != "3" {
		t.Fatalf("unexpected value for c: %q ok=%v", value, ok)
	}
}