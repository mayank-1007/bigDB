#pragma once

#include <atomic>
#include <cstdint>
#include <string>

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
    SSTable,
};

struct Event {
    EventType type = EventType::WAL;
    std::string message;
    std::string key;
    std::string value;
    std::string stage;
    int64_t durationMs = 0;
};

struct Snapshot {
    int64_t totalKeys = 0;
    int64_t walSegments = 0;
    int64_t sstables = 0;
    int64_t lastWriteMS = 0;
    int64_t lastReadNS = 0;
    int64_t lastRecoveryMS = 0;
    int64_t compactions = 0;
    int64_t flushes = 0;
};

class Stats {
public:
    Snapshot Snapshot() const;

    std::atomic<int64_t> totalKeys{0};
    std::atomic<int64_t> walSegments{0};
    std::atomic<int64_t> sstables{0};
    std::atomic<int64_t> lastWriteMS{0};
    std::atomic<int64_t> lastReadNS{0};
    std::atomic<int64_t> lastRecoveryMS{0};
    std::atomic<int64_t> compactions{0};
    std::atomic<int64_t> flushes{0};
};

}  // namespace bigdb::telemetry
