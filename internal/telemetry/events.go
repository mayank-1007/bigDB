package telemetry

import "time"

type EventType string

const (
	EventPut      EventType = "put"
	EventGet      EventType = "get"
	EventDelete   EventType = "delete"
	EventFlush    EventType = "flush"
	EventCompact  EventType = "compact"
	EventRecover  EventType = "recover"
	EventWAL      EventType = "wal"
	EventMemtable EventType = "memtable"
	EventSSTable  EventType = "sstable"
)

type Event struct {
	Time     time.Time `json:"time"`
	Type     EventType `json:"type"`
	Message  string    `json:"message"`
	Key      string    `json:"key,omitempty"`
	Value    string    `json:"value,omitempty"`
	Stage    string    `json:"stage"`
	Duration  int64    `json:"duration_ms,omitempty"`
}

type Snapshot struct {
	TotalKeys      int64 `json:"total_keys"`
	WALSegments    int64 `json:"wal_segments"`
	SSTables       int64 `json:"sstables"`
	LastWriteMS    int64 `json:"last_write_ms"`
	LastReadNS     int64 `json:"last_read_ns"`
	LastRecoveryMS int64 `json:"last_recovery_ms"`
	Compactions    int64 `json:"compactions"`
	Flushes        int64 `json:"flushes"`
}