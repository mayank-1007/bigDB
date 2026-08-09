# BigDB Architecture (C++ Port)

## 1. System Goal

BigDB is a durability-first storage engine designed to demonstrate how an LSM-style database works internally. The implementation is kept intentionally transparent so that each stage can be explained clearly in interviews, design discussions, or demos. This is the C++ port of the original Go implementation.

## 2. High-Level Flow

```text
Client Command
    ↓
WAL Append
    ↓
Memtable Update
    ↓
Background Flush (on threshold)
    ↓
Immutable SSTable Creation
    ↓
Leveled Compaction
    ↓
Crash Recovery via WAL Replay
```
