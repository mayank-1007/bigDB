#pragma once

#include "format.h"
#include "../record/record.h"
#include <memory>
#include <fstream>
#include <utility>

namespace bigdb::sstable {

class File {
public:
    static std::unique_ptr<File> Open(const std::string& path);
    ~File();

    std::string Path() const { return path_; }
    uint32_t Level() const { return level_; }
    uint64_t Sequence() const { return seq_; }

    std::pair<record::Record, bool> Get(const std::vector<uint8_t>& key);
    std::vector<record::Record> All();
    void Close();

private:
    File(const std::string& path, std::unique_ptr<std::ifstream> fh, std::vector<SparseIndexEntry> index, Footer footer, std::unique_ptr<bloom::Filter> bloom, uint32_t level, uint64_t seq);

    std::string path_;
    std::unique_ptr<std::ifstream> fh_;
    std::vector<SparseIndexEntry> index_;
    Footer footer_;
    std::unique_ptr<bloom::Filter> bloom_;
    uint32_t level_;
    uint64_t seq_;
};

} // namespace bigdb::sstable
