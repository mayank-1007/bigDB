package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLeveledCompactionPromotesFiles(t *testing.T) {
	dir := t.TempDir()

	opts := DefaultOptions()
	opts.DataDir = dir
	opts.WALFileName = "wal.log"
	opts.MemtableThreshold = 1
	opts.SparseIndexGap = 1
	opts.LevelCompactionFanout = 2

	database, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := database.Barrier(); err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	for i := 0; i < 4; i++ {
		key := []byte(fmt.Sprintf("k%d", i))
		value := []byte(fmt.Sprintf("v%d", i))
		if err := database.Put(key, value); err != nil {
			t.Fatalf("put %d: %v", i, err)
		}
	}

	deadline := time.Now().Add(4 * time.Second)
	foundHigherLevel := false

	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(filepath.Join(dir, opts.SSTableDirName))
		if err != nil {
			t.Fatalf("read sst dir: %v", err)
		}

		foundHigherLevel = false
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "l1-sst-") || strings.HasPrefix(name, "l2-sst-") {
				foundHigherLevel = true
				break
			}
		}

		if foundHigherLevel {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	if !foundHigherLevel {
		t.Fatal("expected leveled compaction to create a higher-level SSTable")
	}

	for i := 0; i < 4; i++ {
		key := []byte(fmt.Sprintf("k%d", i))
		value, ok, err := database.Get(key)
		if err != nil {
			t.Fatalf("get %d: %v", i, err)
		}
		if !ok || string(value) != fmt.Sprintf("v%d", i) {
			t.Fatalf("unexpected value for %s: ok=%v value=%q", key, ok, value)
		}
	}
}