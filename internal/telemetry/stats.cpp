#include "stats.h"

namespace bigdb::telemetry {

Snapshot Stats::snapshot() const {
    Snapshot s;
    s.total_keys = total_keys.load(std::memory_order_relaxed);
    s.wal_segments = wal_segments.load(std::memory_order_relaxed);
    s.sstables = sstables.load(std::memory_order_relaxed);
    s.last_write_ms = last_write_ms.load(std::memory_order_relaxed);
    s.last_read_ns = last_read_ns.load(std::memory_order_relaxed);
    s.last_recovery_ms = last_recovery_ms.load(std::memory_order_relaxed);
    s.compactions = compactions.load(std::memory_order_relaxed);
    s.flushes = flushes.load(std::memory_order_relaxed);
    return s;
}

} // namespace bigdb::telemetry
