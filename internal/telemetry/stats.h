#pragma once

#include "events.h"
#include <atomic>

namespace bigdb::telemetry {

struct Stats {
    std::atomic<int64_t> total_keys{0};
    std::atomic<int64_t> wal_segments{0};
    std::atomic<int64_t> sstables{0};
    std::atomic<int64_t> last_write_ms{0};
    std::atomic<int64_t> last_read_ns{0};
    std::atomic<int64_t> last_recovery_ms{0};
    std::atomic<int64_t> compactions{0};
    std::atomic<int64_t> flushes{0};

    Snapshot snapshot() const;
};

} // namespace bigdb::telemetry
