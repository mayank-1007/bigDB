# AETHER_DB Architecture

## 1. System Goal

AETHER_DB is a durability-first storage engine designed to demonstrate how an LSM-style database works internally. The implementation is kept intentionally transparent so that each stage can be explained clearly in interviews, design discussions, or demos.

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