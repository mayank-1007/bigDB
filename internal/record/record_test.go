package record

import "testing"

func TestRecordEncodeDecodeRoundTrip(t *testing.T) {
	original := Record{
		Key:       []byte("user:1"),
		Value:     []byte("alice"),
		Timestamp: 123456789,
		IsDeleted: false,
	}

	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if string(decoded.Key) != string(original.Key) {
		t.Fatalf("key mismatch: got %q want %q", decoded.Key, original.Key)
	}
	if string(decoded.Value) != string(original.Value) {
		t.Fatalf("value mismatch: got %q want %q", decoded.Value, original.Value)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Fatalf("timestamp mismatch: got %d want %d", decoded.Timestamp, original.Timestamp)
	}
	if decoded.IsDeleted != original.IsDeleted {
		t.Fatalf("deleted flag mismatch: got %v want %v", decoded.IsDeleted, original.IsDeleted)
	}
}

func TestDeleteRecordRoundTrip(t *testing.T) {
	original := NewDelete([]byte("user:2"), 999)

	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if !decoded.IsDeleted {
		t.Fatalf("expected tombstone record")
	}
	if string(decoded.Key) != "user:2" {
		t.Fatalf("key mismatch: got %q", decoded.Key)
	}
}