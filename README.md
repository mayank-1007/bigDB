# AETHER_DB — Bigtable-Inspired Storage Engine

AETHER_DB is a Go storage-engine project inspired by Bigtable and LSM-tree systems. It is not just a CRUD demo: it implements a durable write path, an in-memory memtable, immutable SSTables, background flush, leveled compaction, crash recovery, corruption detection, live telemetry, and an HTTP API that drives a real-time visualizer.

The project was validated with full test, race, stability, benchmark, recovery, compaction, corruption, and API checks. The final validation run passed end-to-end, including `go test ./...`, `go test -race ./...`, `go test ./... -count=10`, the recovery suite, the compaction suite, SSTable corruption tests, API tests, and profiling runs. Representative benchmark results from the latest validation were:

- `BenchmarkDBGetHotKey` at about **239 ns/op**
- `BenchmarkDBConcurrentWrites` at about **529 µs/op**
- `BenchmarkDBRecoveryReplay` at about **120 ms**
- `BenchmarkDBPutSequential` at about **682 µs/op** in one full validation run, with later runs varying slightly as expected under repeated benchmark execution

Those numbers matter because they show the intended trade-offs of the design: hot reads are memory-fast, writes pay the WAL/fsync durability cost, and recovery is bounded by replay volume.

## What this project demonstrates

AETHER_DB covers the core storage-engine concepts that are normally hidden inside production systems:

- **WAL-first durability** with checksum validation
- **Memtable** for hot in-memory reads and writes
- **SSTable flush path** with sparse indexing and Bloom filters
- **Leveled compaction** with duplicate elimination and tombstone preservation
- **Crash recovery** via WAL replay
- **Corruption detection** for both WAL and SSTable files
- **Telemetry and live event streaming** for observability
- **HTTP API** for command execution and visualization
- **Benchmarks and profiling** to expose the actual bottlenecks

## Core architecture

The engine follows the classic LSM-style write path:

**Client command → WAL append → Memtable update → background flush → SSTable creation → leveled compaction → recovery via WAL replay**

Reads are resolved with the standard priority order:

1. Memtable
2. Frozen snapshot / recent flush state
3. SSTables, using level-aware search order
4. Tombstones stop stale values from resurfacing

This is the main reason the project is useful in interviews: it demonstrates durability, write amplification control, read optimization, and crash recovery as a complete system rather than as isolated features.

## Tech stack

### Backend
- **Go**
- `sync`, `time`, `os`, `path/filepath`, and standard concurrency primitives
- custom packages for WAL, memtable, SSTable, compaction, recovery, and telemetry
- `net/http` for the live API
- `go test`, `go test -race`, `go test -bench`, `go tool pprof`

### Frontend
- **React + TypeScript**
- Vite
- Framer Motion
- Recharts
- Lucide icons
- dark, product-style dashboard layout that consumes the live API

## Repository layout

- `cmd/db` — CLI and HTTP server entrypoint
- `internal/db` — core engine orchestration
- `internal/wal` — append, replay, rotation, checkpointing, checksum handling
- `internal/memtable` — in-memory state and snapshot support
- `internal/sstable` — SSTable read/write, sparse index, checksum validation
- `internal/compaction` — SSTable merge logic
- `internal/recovery` — WAL replay into memtable
- `internal/api` — command/state/event API used by the visualizer
- `internal/telemetry` — event and snapshot types
- `bigtable-visualizer` — the frontend dashboard

## Runtime modes

### 1) One-shot CLI mode

Use this mode for quick manual verification of the engine:

```bash
go run ./cmd/db -data-dir ./data -put-key user:1 -put-value alice
go run ./cmd/db -data-dir ./data -get-key user:1
go run ./cmd/db -data-dir ./data -del-key user:1
go run ./cmd/db -data-dir ./data -compact
```

Observed behavior in the project validation:
- `put ok`
- `user:1 => alice`
- `delete ok`
- `compaction ok`

### 2) Live API / visualizer mode

This mode exposes the engine state and event stream for the frontend:

```bash
go run ./cmd/db -serve -data-dir ./data -http :8080 -pprof :6060
```

Available endpoints:
- `GET /api/state` — current snapshot
- `POST /api/command` — run `put`, `get`, `delete`, or `compact`
- `GET /api/events` — live event stream for packet / stage animation
- `GET /debug/pprof/` — runtime profiling when pprof is enabled

## Validation evidence

The engine was checked with a full verification script that exercised the following areas:

- `go test ./...`
- `go test -race ./...`
- `go test ./... -count=10`
- recovery tests for clean and unclean restarts
- compaction tests for duplicate removal and tombstone handling
- SSTable checksum corruption tests
- WAL checksum and replay tests
- API tests for state, command, and event streaming
- repeated benchmarks and profiling

The final consolidated validation run completed successfully and produced:
- a passed test log
- `cpu.out`
- `mem.out`

## Benchmark interpretation

The benchmark results are consistent with the intended architecture:

- **`BenchmarkDBGetHotKey` ~239 ns/op**  
  This shows the hot read path is memory-resident and very fast.

- **`BenchmarkDBConcurrentWrites` ~529 µs/op**  
  This reflects the durability cost of WAL append, synchronization, and background maintenance.

- **`BenchmarkDBRecoveryReplay` ~120 ms**  
  This is the cost of rebuilding state from the WAL after restart.

- **`BenchmarkDBPutSequential` ~682 µs/op** in the validation run  
  This is the steady write cost under repeated sequential inserts.

The CPU profile showed the expected bottleneck: write cost is dominated by WAL append and file sync, which is exactly what a durable engine should reveal. The memory profile showed the largest allocations in `DB.Get`, `Memtable.Put`, `DB.Open`, and WAL replay paths, which gives you real discussion points for optimization.

## Why this project is resume-worthy

This project demonstrates more than CRUD:

- an actual storage-engine write path
- recovery and corruption handling
- concurrency and race safety
- compaction behavior under load
- telemetry and profiling
- a live API for tooling and visualization
- enough depth to discuss trade-offs in an interview

## What the frontend is for

The frontend visualizer is meant to explain the engine as a live system:
- show a command entering the engine
- show the packet moving through WAL, memtable, SSTable, compaction, and recovery
- expose live state from the backend API
- display the real benchmark and telemetry story in a way that is easy to present

## Short command reference

A compact list of the most useful verification commands is below:

```bash
go test ./...
go test -race ./...
go test ./... -count=10
go test -run TestRecovery ./internal/db -v
go test -run Compaction ./internal/db -v
go test -run Corruption ./internal/sstable -v
go test ./internal/api -v
go test ./... -bench=. -benchmem
go test ./... -bench=. -cpuprofile cpu.out ./internal/db
go test ./... -bench=. -memprofile mem.out ./internal/db
```

This repository is now in a state where the backend engine is verified, measurable, and ready for presentation.
