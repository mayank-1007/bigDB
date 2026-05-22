# AETHER_DB Architecture

## 1. System Goal

AETHER_DB is a storage engine designed to demonstrate how a durability-first, LSM-style database works internally. The implementation is intentionally transparent so that each stage can be explained in an interview or live demo.

## 2. High-Level Flow

```text
Client Command
    ↓
WAL Append
    ↓
Memtable Update
    ↓
Background Flush (when threshold is reached)
    ↓
Immutable SSTable Creation
    ↓
Leveled Compaction
    ↓
Crash Recovery via WAL Replay
```

## 3. Core Components

### `cmd/db`
Entry point for:
- one-shot CLI operations (`put`, `get`, `delete`, `compact`)
- live server mode for the visualizer
- optional pprof server for profiling

### `internal/db`
The orchestration layer. It:
- owns the WAL
- owns the active memtable
- manages SSTable caches
- coordinates flush and compaction workers
- exposes a snapshot and event stream for the UI

### `internal/wal`
Handles:
- append-only durable logging
- replay after restart
- rotation and checkpoint-based cleanup
- corruption detection using checksums

### `internal/memtable`
In-memory write path:
- accepts latest records
- supports tombstones
- serves hot reads

### `internal/sstable`
Immutable on-disk tables:
- sorted records
- sparse index
- checksum verification
- corruption-safe open/read path

### `internal/compaction`
Merges SSTables:
- removes duplicates
- keeps the latest timestamp
- preserves tombstones
- writes output to the next level

### `internal/recovery`
Rebuilds the in-memory state from the WAL after restart.

### `internal/api`
Provides:
- `GET /api/state`
- `POST /api/command`
- `GET /api/events`

### `internal/telemetry`
Defines:
- event types
- live snapshot counters for the visualizer

## 4. Concurrency Model

The engine uses multiple locks for distinct concerns:

- `mu` — protects active engine state and general lifecycle changes
- `sstMu` — protects SSTable map and SSTable cache changes
- `flushMu` — protects frozen snapshots waiting for flush completion
- `compactMu` — serializes compaction and checkpoint maintenance
- `wg` — waits for background flush and compaction workers

This separation keeps the engine responsive while preserving correctness.

## 5. Durability Story

A write is durable only after it has been appended to the WAL. That means:
- the WAL is the first persistence boundary
- the memtable is the fast serving layer
- if the process crashes, recovery replays the WAL

## 6. Read Path

Reads try the hottest layers first:
1. memtable
2. frozen snapshots, if any
3. SSTable cache by level
4. tombstones terminate the lookup if present

This keeps common reads fast while remaining correct after flush and compaction.

## 7. Flush Path

When the memtable crosses the configured threshold:
- the active memtable is frozen
- the WAL rotates
- a background worker writes an SSTable
- the newly created SSTable is opened and cached
- the frozen snapshot is removed
- compaction may be launched if needed

## 8. Compaction Path

Compaction is level-driven:
- if a level exceeds fanout, merge it
- keep the latest version of each key
- keep tombstones so deleted keys do not resurrect
- write merged output to the next level
- remove old files after the merged file is installed

## 9. Recovery Path

On startup:
- the WAL is opened
- records are replayed into the memtable
- SSTables are loaded from disk
- archived WALs are cleaned up with checkpoint handling
- live counters are refreshed for the API/visualizer

## 10. Observability

The project includes:
- command events
- stage events
- live counters
- benchmark results
- pprof profiles

This makes the engine suitable for teaching, debugging, and interviews.

## 11. Validation Summary

The validated backend passed:
- `go test ./...`
- `go test -race ./...`
- `go test ./... -count=10`
- recovery tests
- compaction tests
- corruption tests
- API tests
- benchmarks and CPU/memory profiling

## 12. Interview Message

This project is not just CRUD:
- it demonstrates durability
- it handles crash recovery
- it has background maintenance
- it uses checksums
- it exposes real concurrency and performance tradeoffs
- it supports a live visual explanation layer
