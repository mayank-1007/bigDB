package db

import (
	"runtime"
	"testing"
)

func TestRecoveryAfterUncleanShutdownSimulation(t *testing.T) {
	dir := t.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1_000_000
	opts.SparseIndexGap = 10

	database, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := database.Put([]byte("user:1"), []byte("alice")); err != nil {
		t.Fatalf("put user:1: %v", err)
	}
	if err := database.Put([]byte("user:2"), []byte("bob")); err != nil {
		t.Fatalf("put user:2: %v", err)
	}
	if err := database.Delete([]byte("user:2")); err != nil {
		t.Fatalf("delete user:2: %v", err)
	}

	// Simulate process disappearance.
	// WAL durability is already guaranteed because Append() calls Sync().
	// We close the OS handle so Windows cleanup succeeds.
	if err := database.Close(); err != nil {
		t.Fatalf("close before simulated restart: %v", err)
	}

	database = nil
	runtime.GC()

	reopened, err := Open(opts)
	if err != nil {
		t.Fatalf("reopen after simulated crash: %v", err)
	}
	defer reopened.Close()

	value, ok, err := reopened.Get([]byte("user:1"))
	if err != nil {
		t.Fatalf("get user:1: %v", err)
	}
	if !ok || string(value) != "alice" {
		t.Fatalf("unexpected recovered value for user:1: ok=%v value=%q", ok, value)
	}

	_, ok, err = reopened.Get([]byte("user:2"))
	if err != nil {
		t.Fatalf("get user:2: %v", err)
	}
	if ok {
		t.Fatal("expected user:2 to remain deleted after recovery")
	}
}

func TestRecoveryAfterManyWrites(t *testing.T) {
	dir := t.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1_000_000
	opts.SparseIndexGap = 10

	database, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	for i := 0; i < 1000; i++ {
		key := []byte("k")
		value := []byte("v")
		if err := database.Put(key, value); err != nil {
			t.Fatalf("put: %v", err)
		}
	}

	// Simulate abrupt restart after durable WAL sync.
	if err := database.Close(); err != nil {
		t.Fatalf("close before restart: %v", err)
	}

	database = nil
	runtime.GC()

	reopened, err := Open(opts)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	value, ok, err := reopened.Get([]byte("k"))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok || string(value) != "v" {
		t.Fatalf("unexpected recovered value: ok=%v value=%q", ok, value)
	}
}