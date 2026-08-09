#pragma once

#include <string>

namespace bigdb::db {

struct Options {
    std::string DataDir = "./data";
    std::string WALFileName = "wal.log";
    std::string SSTableDirName = "sst";
    int MemtableThreshold = 2000;
    int SparseIndexGap = 10;
    int LevelCompactionFanout = 4;
    int CompactionIntervalMS = 0; // 0 means no background compaction timer, only triggered by flush
};

Options DefaultOptions();

} // namespace bigdb::db
