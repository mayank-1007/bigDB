package memtable

import (
	"testing"

	"bigdb/internal/record"
)

func TestMemtablePutGetDelete(t *testing.T) {
	m := New()

	put := record.NewPut([]byte("k1"), []byte("v1"), 1)
	m.Put(put)

	got, ok := m.Get([]byte("k1"))
	if !ok {
		t.Fatal("expected key to exist")
	}
	if string(got.Value) != "v1" {
		t.Fatalf("unexpected value: %s", got.Value)
	}

	del := record.NewDelete([]byte("k1"), 2)
	m.Delete(del)

	_, ok = m.Get([]byte("k1"))
	if ok {
		t.Fatal("expected deleted key to be hidden")
	}
}
