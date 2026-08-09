#pragma once

#include <string>
#include <chrono>

namespace bigdb::telemetry {

enum class EventType {
    Put,
    Get,
    Delete,
    Flush,
    Compact,
    Recover,
    WAL,
    Memtable,
    SSTable
};

std::string to_string(EventType type);

struct Event {
    std::chrono::system_clock::time_point time;
    EventType type;
    std::string message;
    std::string key;
    std::string value;
    std::string stage;
    int64_t duration_ms{0};
};

struct Snapshot {
    int64_t total_keys{0};
    int64_t wal_segments{0};
    int64_t sstables{0};
    int64_t last_write_ms{0};
    int64_t last_read_ns{0};
    int64_t last_recovery_ms{0};
    int64_t compactions{0};
    int64_t flushes{0};
};

} // namespace bigdb::telemetry
