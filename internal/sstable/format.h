#pragma once

#include <cstdint>
#include <map>
#include <string>
#include <vector>

#include "../record/record.h"

namespace bigdb::sstable {

struct SparseIndexEntry {
    std::string key;
    uint64_t offset = 0;
};

struct Footer {
    uint64_t indexOffset = 0;
    uint32_t indexCount = 0;
    uint32_t recordCount = 0;
    uint32_t sparseGap = 0;
    uint64_t bloomOffset = 0;
    uint32_t bloomSize = 0;
};

class SSTable {
public:
    SSTable(const std::string& path);
    ~SSTable();

    std::vector<record::Record> All() const;
    record::Record Get(const std::string& key) const;
    uint32_t Level() const;
    uint64_t Sequence() const;
    const std::string& Path() const;

private:
    std::string path_;
    uint32_t level_ = 0;
    uint64_t seq_ = 0;
};

void WriteSnapshot(const std::string& path, const std::map<std::string, record::Record>& snapshot, int sparseGap);

}  // namespace bigdb::sstable
