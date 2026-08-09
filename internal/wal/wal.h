#pragma once

#include <cstdint>
#include <string>
#include <vector>

#include "../record/record.h"

namespace bigdb::wal {

class WAL {
public:
    WAL(const std::string& path);
    ~WAL();

    void Append(const record::Record& rec);
    std::string Rotate();
    std::vector<record::Record> Replay() const;
    void Close();

private:
    std::string path_;
    std::string dir_;
    std::string activePath_;
    std::FILE* file_ = nullptr;
    uint64_t nextArchive_ = 1;
};

}  // namespace bigdb::wal
