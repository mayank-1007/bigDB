package memtable

import (
	"bytes"
	"sync"

	"bigdb/internal/record"
)

type Memtable struct {
	mu   sync.RWMutex
	data map[string]record.Record
}

func New() *Memtable {
	return &Memtable{
		data: make(map[string]record.Record),
	}
}

func (m *Memtable) Put(rec record.Record) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := string(rec.Key)

	existing, ok := m.data[key]
	if ok && existing.Timestamp > rec.Timestamp {
		return
	}

	m.data[key] = rec
}

func (m *Memtable) Delete(rec record.Record) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := string(rec.Key)

	existing, ok := m.data[key]
	if ok && existing.Timestamp > rec.Timestamp {
		return
	}

	m.data[key] = rec
}

func (m *Memtable) Get(key []byte) (record.Record, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, ok := m.data[string(key)]
	if !ok {
		return record.Record{}, false
	}

	if rec.IsDeleted {
		return record.Record{}, false
	}

	if !bytes.Equal(rec.Key, key) {
		return record.Record{}, false
	}

	return rec, true
}

func (m *Memtable) Snapshot() map[string]record.Record {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string]record.Record, len(m.data))
	for k, v := range m.data {
		out[k] = v
	}
	return out
}

func (m *Memtable) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}
