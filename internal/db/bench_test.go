package db

import (
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkDBPutSequential(b *testing.B) {
	dir := b.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1_000_000
	opts.SparseIndexGap = 10

	database, err := Open(opts)
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer database.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := []byte("key:" + strconv.Itoa(i))
		value := []byte("value:" + strconv.Itoa(i))
		if err := database.Put(key, value); err != nil {
			b.Fatalf("put: %v", err)
		}
	}
}

func BenchmarkDBGetHotKey(b *testing.B) {
	dir := b.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1_000_000
	opts.SparseIndexGap = 10

	database, err := Open(opts)
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := database.Put([]byte("hot"), []byte("value")); err != nil {
		b.Fatalf("seed put: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value, ok, err := database.Get([]byte("hot"))
		if err != nil {
			b.Fatalf("get: %v", err)
		}
		if !ok || string(value) != "value" {
			b.Fatalf("unexpected get result: ok=%v value=%q", ok, value)
		}
	}
}

func BenchmarkDBConcurrentWrites(b *testing.B) {
	dir := b.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1_000_000
	opts.SparseIndexGap = 10

	database, err := Open(opts)
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer database.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		local := 0
		for pb.Next() {
			key := []byte(fmt.Sprintf("k-%d", local))
			value := []byte("v")
			if err := database.Put(key, value); err != nil {
				b.Fatalf("put: %v", err)
			}
			local++
		}
	})
}

func BenchmarkDBRecoveryReplay(b *testing.B) {
	dir := b.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1_000_000
	opts.SparseIndexGap = 10

	seed, err := Open(opts)
	if err != nil {
		b.Fatalf("open seed db: %v", err)
	}

	for i := 0; i < 5000; i++ {
		key := []byte("key:" + strconv.Itoa(i))
		value := []byte("value:" + strconv.Itoa(i))
		if err := seed.Put(key, value); err != nil {
			b.Fatalf("seed put: %v", err)
		}
	}
	if err := seed.Close(); err != nil {
		b.Fatalf("close seed db: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reopened, err := Open(opts)
		if err != nil {
			b.Fatalf("reopen: %v", err)
		}
		if err := reopened.Close(); err != nil {
			b.Fatalf("close reopened db: %v", err)
		}
	}
}