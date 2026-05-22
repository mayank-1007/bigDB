#!/bin/bash

set -e

OUT="verification_$(date +%Y%m%d_%H%M%S).txt"

exec > >(tee "$OUT") 2>&1

echo "====================================="
echo "BIGTABLE COMPLETE VALIDATION"
echo "====================================="

echo
echo "STEP 0 — Clean"
go clean -testcache

echo
echo "STEP 1 — Build"
go build ./...

echo
echo "STEP 2 — All Tests"
go test ./...

echo
echo "STEP 3 — Race Detection"
go test -race ./...

echo
echo "STEP 4 — Stability (10x)"
go test ./... -count=10

echo
echo "STEP 5 — Benchmarks"
go test ./... -bench=. -benchmem

echo
echo "STEP 6 — Recovery"
go test -run TestRecovery -v ./internal/db

echo
echo "STEP 7 — Compaction"
go test -run Compaction -v ./internal/db

echo
echo "STEP 8 — Corruption"
go test -run Corruption -v ./internal/sstable

echo
echo "STEP 9 — API"
go test -v ./internal/api

echo
echo "STEP 10 — WAL"
go test -v ./internal/wal

echo
echo "STEP 11 — Memtable"
go test -v ./internal/memtable

echo
echo "STEP 12 — DB"
go test -v ./internal/db

echo
echo "STEP 13 — CPU Profile"
go test -bench=. -cpuprofile cpu.out ./internal/db

echo
echo "STEP 14 — Memory Profile"
go test -bench=. -memprofile mem.out ./internal/db

echo
echo "STEP 15 — Build Server"
go build -o bigtable.exe ./cmd/db

echo
echo "====================================="
echo "SUCCESS"
echo "Logs → $OUT"
echo "cpu.out generated"
echo "mem.out generated"
echo "====================================="