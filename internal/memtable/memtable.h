#pragma once

#include <cstdint>
#include <map>
#include <string>

#include "../record/record.h"

namespace bigdb::memtable {

class Memtable {
public:
    void Put(const record::Record& rec);
    void Delete(const record::Record& rec);
    record::Record Get(const std::string& key) const;
    bool Contains(const std::string& key) const;
    std::map<std::string, record::Record> Snapshot() const;
    size_t Size() const;

private:
    std::map<std::string, record::Record> data_;
};

}  // namespace bigdb::memtable
