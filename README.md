# BigDB — Bigtable-Inspired/LSM-tree Storage Engine

BigDB is a Go-based storage engine inspired by Bigtable and LSM-tree design principles. It implements a durable write path, in-memory memtable, immutable SSTables, background flush, leveled compaction, crash recovery, corruption detection, and telemetry.

## Overview

BigDB is designed to demonstrate the core mechanics of a production-style storage engine, including:

- write-ahead logging for durability
- in-memory buffering through a memtable
- immutable SSTable creation on flush
- leveled compaction for efficient space reclamation
- crash recovery through WAL replay
- checksum-based corruption detection
- observability through telemetry events
- benchmarking and profiling for performance analysis

## Architecture

The engine follows a standard LSM-style flow:

`Client write → WAL append → Memtable update → background flush → SSTable creation → leveled compaction → WAL replay on recovery`

### Read Path

Reads are resolved in the following order:

1. Memtable
2. Recently flushed state and cached SSTables
3. SSTables, searched in level order
4. Tombstones to preserve delete semantics

### Write Path

Writes are first appended to the WAL and then applied to the memtable. This ensures durability even if the process exits unexpectedly before a flush occurs.

## Core Features

- **WAL-first durability** with checksum validation
- **Memtable** for fast in-memory reads and writes
- **SSTable caching** to keep opened tables in memory and avoid repeated reloads
- **SSTable flush path** with sparse indexing and Bloom filters
- **Leveled compaction** with duplicate elimination and tombstone handling
- **Crash recovery** through WAL replay
- **Corruption detection** for WAL and SSTable files
- **Telemetry** for runtime visibility
- **Benchmarks and profiling** to measure real performance characteristics

## Tech Stack

- **Go**
- Standard library packages for concurrency, file I/O, timing, and filesystem operations
- Custom packages for WAL, memtable, SSTable, compaction, recovery, and telemetry
- Testing and profiling tools:
  - `go test`
  - `go test -race`
  - `go test -bench`
  - `go tool pprof`

## Repository Layout

- `cmd/db` — CLI entrypoint
- `internal/db` — core engine orchestration
- `internal/wal` — append, replay, rotation, checkpointing, checksum handling
- `internal/memtable` — in-memory state and snapshot support
- `internal/sstable` — SSTable read/write, sparse index, checksum validation
- `internal/compaction` — SSTable merge logic
- `internal/recovery` — WAL replay into memtable
- `internal/telemetry` — event and snapshot types

## Container Delivery

The project can be built and shared as a container image.

```bash
docker build -t bigdb .
docker run --rm -v bigdb-data:/data bigdb -put-key user:1 -put-value alice
```

Published image:

- GHCR: `ghcr.io/mayank-1007/bigdb:latest`
- Package page: `https://github.com/users/mayank-1007/packages/container/bigdb`

Can be directly pulled with:

```bash
docker pull ghcr.io/mayank-1007/bigdb:latest
```

## Usage

### One-shot CLI mode

```bash
go run ./cmd/db -data-dir ./data -put-key user:1 -put-value alice
go run ./cmd/db -data-dir ./data -get-key user:1
go run ./cmd/db -data-dir ./data -del-key user:1
go run ./cmd/db -data-dir ./data -compact
