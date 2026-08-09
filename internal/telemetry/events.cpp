#include "events.h"

namespace bigdb::telemetry {

std::string to_string(EventType type) {
    switch (type) {
        case EventType::Put: return "put";
        case EventType::Get: return "get";
        case EventType::Delete: return "delete";
        case EventType::Flush: return "flush";
        case EventType::Compact: return "compact";
        case EventType::Recover: return "recover";
        case EventType::WAL: return "wal";
        case EventType::Memtable: return "memtable";
        case EventType::SSTable: return "sstable";
        default: return "unknown";
    }
}

} // namespace bigdb::telemetry
