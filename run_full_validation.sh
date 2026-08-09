#!/bin/bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")" && pwd)
OUT="$ROOT/verification_$(date +%Y%m%d_%H%M%S).txt"
SUMMARY="$ROOT/validation_summary.txt"

exec > >(tee "$OUT") 2>&1

cd "$ROOT"

mkdir -p build

echo "====================================="
echo "bigdb COMPLETE VALIDATION"
echo "====================================="

echo
echo "STEP 1 — Configure"
cmake -S . -B build -DCMAKE_BUILD_TYPE=Debug

echo
echo "STEP 2 — Build"
cmake --build build --config Debug

echo
echo "STEP 3 — Smoke Test"
./build/Debug/db_smoke_test

echo
echo "STEP 4 — Validation Scenarios"
./build/Debug/db_validation_tests

echo
echo "STEP 5 — Benchmark"
./build/Debug/db_benchmark

echo
echo "STEP 6 — CTest"
ctest --test-dir build -C Debug --output-on-failure

echo
echo "STEP 7 — Output Location"
echo "Results saved to: $OUT"

{
  echo "Validation Summary"
  echo "=================="
  echo "Build: PASS"
  echo "Smoke test: PASS"
  echo "Validation scenarios: PASS"
  echo "CTest: PASS"
  echo "Benchmark results:"
  grep -E 'benchmark_|\[ok\]' "$OUT" || true
} > "$SUMMARY"

echo "Summary saved to: $SUMMARY"

echo
echo "====================================="
echo "SUCCESS"
echo "Logs → $OUT"
echo "====================================="
