package bloom

import "testing"

func TestBloomFilterMightContainForInsertedKey(t *testing.T) {
	f := NewForKeys(10)
	f.Add([]byte("alpha"))
	f.Add([]byte("beta"))

	if !f.MightContain([]byte("alpha")) {
		t.Fatal("expected inserted key to be present")
	}
	if !f.MightContain([]byte("beta")) {
		t.Fatal("expected inserted key to be present")
	}
}