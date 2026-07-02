package wal

import (
	"path/filepath"
	"testing"

	"bigdb/internal/record"
)

func TestWALAppendAndReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal.log")

	w, err := Open(path)
	if err != nil {
		t.Fatalf("open wal: %v", err)
	}
	defer w.Close()

	inputs := []record.Record{
		record.NewPut([]byte("a"), []byte("1"), 1),
		record.NewPut([]byte("b"), []byte("2"), 2),
		record.NewDelete([]byte("a"), 3),
	}

	for _, rec := range inputs {
		if err := w.Append(rec); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}

	var got []record.Record
	if err := w.Replay(func(r record.Record) error {
		got = append(got, r)
		return nil
	}); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	if len(got) != len(inputs) {
		t.Fatalf("replay length mismatch: got %d want %d", len(got), len(inputs))
	}

	for i := range inputs {
		if string(got[i].Key) != string(inputs[i].Key) {
			t.Fatalf("key mismatch at %d: got %q want %q", i, got[i].Key, inputs[i].Key)
		}
		if string(got[i].Value) != string(inputs[i].Value) {
			t.Fatalf("value mismatch at %d: got %q want %q", i, got[i].Value, inputs[i].Value)
		}
		if got[i].Timestamp != inputs[i].Timestamp {
			t.Fatalf("timestamp mismatch at %d: got %d want %d", i, got[i].Timestamp, inputs[i].Timestamp)
		}
		if got[i].IsDeleted != inputs[i].IsDeleted {
			t.Fatalf("deleted mismatch at %d: got %v want %v", i, got[i].IsDeleted, inputs[i].IsDeleted)
		}
	}
}
