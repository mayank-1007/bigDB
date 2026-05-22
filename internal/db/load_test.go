package db

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentWritesAndReads(t *testing.T) {
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
	defer database.Close()

	const writers = 10
	const writesPerWriter = 500

	var wg sync.WaitGroup
	wg.Add(writers)

	for w := 0; w < writers; w++ {
		go func(writerID int) {
			defer wg.Done()

			for i := 0; i < writesPerWriter; i++ {
				key := []byte(fmt.Sprintf("w%d-k%d", writerID, i))
				value := []byte(fmt.Sprintf("v%d", i))
				if err := database.Put(key, value); err != nil {
					t.Errorf("put failed: %v", err)
					return
				}
			}
		}(w)
	}

	wg.Wait()

	for w := 0; w < writers; w++ {
		for i := 0; i < writesPerWriter; i++ {
			key := []byte(fmt.Sprintf("w%d-k%d", w, i))
			expected := fmt.Sprintf("v%d", i)

			value, ok, err := database.Get(key)
			if err != nil {
				t.Fatalf("get failed: %v", err)
			}
			if !ok {
				t.Fatalf("missing key %q", key)
			}
			if string(value) != expected {
				t.Fatalf("unexpected value for %q: got %q want %q", key, value, expected)
			}
		}
	}
}