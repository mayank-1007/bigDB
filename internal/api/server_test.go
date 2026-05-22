package api

import (
	"bytes"
	"encoding/json"
	// "io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	"bufio"

	"bigtable/internal/telemetry"
	"bigtable/internal/db"
)

type fakeEngine struct {
	mu       sync.Mutex
	putCount  int
	delCount  int
	compactCount int
	snapshot telemetry.Snapshot
	events   chan telemetry.Event
}

func newFakeEngine() *fakeEngine {
	return &fakeEngine{
		events: make(chan telemetry.Event, 16),
		snapshot: telemetry.Snapshot{
			TotalKeys:   2,
			WALSegments: 1,
			SSTables:    3,
		},
	}
}

func (f *fakeEngine) Put(key, value []byte) error {
	f.mu.Lock()
	f.putCount++
	f.mu.Unlock()
	f.events <- telemetry.Event{Time: time.Now(), Type: telemetry.EventPut, Message: "put", Key: string(key), Value: string(value), Stage: "wal"}
	return nil
}

func (f *fakeEngine) Delete(key []byte) error {
	f.mu.Lock()
	f.delCount++
	f.mu.Unlock()
	f.events <- telemetry.Event{Time: time.Now(), Type: telemetry.EventDelete, Message: "delete", Key: string(key), Stage: "wal"}
	return nil
}

func (f *fakeEngine) Get(key []byte) ([]byte, bool, error) {
	return []byte("value"), true, nil
}

func (f *fakeEngine) Compact() error {
	f.mu.Lock()
	f.compactCount++
	f.mu.Unlock()
	f.events <- telemetry.Event{Time: time.Now(), Type: telemetry.EventCompact, Message: "compact", Stage: "maintenance"}
	return nil
}

func (f *fakeEngine) Snapshot() telemetry.Snapshot {
	return f.snapshot
}

func (f *fakeEngine) Events() <-chan telemetry.Event {
	return f.events
}

func TestStateEndpoint(t *testing.T) {
	srv := NewServer(newFakeEngine())

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var snap telemetry.Snapshot
	if err := json.NewDecoder(rec.Body).Decode(&snap); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if snap.TotalKeys != 2 || snap.SSTables != 3 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestCommandEndpoint(t *testing.T) {
	engine := newFakeEngine()
	srv := NewServer(engine)

	body := bytes.NewBufferString(`{"op":"put","key":"user:1","value":"alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/command", body)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	engine.mu.Lock()
	defer engine.mu.Unlock()

	if engine.putCount != 1 {
		t.Fatalf("expected putCount=1, got %d", engine.putCount)
	}
}

func TestEventsEndpointStreams(t *testing.T) {
	opts := db.DefaultOptions()
	opts.DataDir = t.TempDir()

	engine, err := db.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	handler := NewServer(engine)

	srv := httptest.NewServer(handler.Handler())
	defer srv.Close()

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(srv.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)

		reader := bufio.NewReader(resp.Body)

		_, _ = reader.ReadString('\n')
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("event stream timeout")
	}
}

