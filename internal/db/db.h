#pragma once

#include <atomic>
#include <cstdint>
#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

#include "../compaction/compaction.h"
#include "../memtable/memtable.h"
#include "../record/record.h"
#include "../sstable/format.h"
#include "../telemetry/telemetry.h"
#include "../wal/wal.h"

namespace bigdb::db {

struct Options {
    std::string DataDir = "./data";
    std::string WALFileName = "wal.log";
    std::string SSTableDirName = "sst";
    int MemtableThreshold = 2000;
    int SparseIndexGap = 10;
    int LevelCompactionFanout = 4;
};

Options DefaultOptions();

class DB {
public:
    explicit DB(const Options& opts);
    ~DB();

    void Put(const std::string& key, const std::string& value);
    void Delete(const std::string& key);
    std::pair<std::string, bool> Get(const std::string& key) const;
    void Compact();
    void Close();

    telemetry::Stats& Stats();

private:
    void FreezeActive();
    void FlushSnapshotAsync(uint64_t id, const std::string& path, const std::map<std::string, record::Record>& snapshot, const std::string& walArchive);
    void MarkFlushComplete(uint64_t id, const std::string& path);
    void RefreshCounts();
    std::string SSTablePath(uint32_t level, uint64_t seq) const;
    void LaunchCompaction();
    void CleanupArchivedWALs();

    Options opts_;
    std::unique_ptr<wal::WAL> wal_;
    mutable std::mutex mu_;
    mutable std::mutex sstMu_;
    mutable std::mutex flushMu_;
    mutable std::mutex compactMu_;
    std::shared_ptr<memtable::Memtable> active_;
    std::vector<std::pair<uint64_t, std::map<std::string, record::Record>>> frozen_;
    std::map<uint32_t, std::vector<std::shared_ptr<sstable::SSTable>>> sstables_;
    uint64_t nextSSTable_ = 0;
    bool closed_ = false;
    bool compactionRunning_ = false;
    std::thread flushThread_;
    std::thread compactionThread_;
    telemetry::Stats stats_;
};

}  // namespace bigdb::db
