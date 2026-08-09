#include "memtable.h"

namespace bigdb::memtable {

void Memtable::Put(const record::Record& rec) {
    auto it = data_.find(rec.key);
    if (it != data_.end() && it->second.timestamp > rec.timestamp) {
        return;
    }
    data_[rec.key] = rec;
}

void Memtable::Delete(const record::Record& rec) {
    auto it = data_.find(rec.key);
    if (it != data_.end() && it->second.timestamp > rec.timestamp) {
        return;
    }
    data_[rec.key] = rec;
}

record::Record Memtable::Get(const std::string& key) const {
    auto it = data_.find(key);
    if (it == data_.end()) {
        return {};
    }
    return it->second;
}

bool Memtable::Contains(const std::string& key) const {
    return data_.find(key) != data_.end();
}

std::map<std::string, record::Record> Memtable::Snapshot() const {
    return data_;
}

size_t Memtable::Size() const {
    return data_.size();
}

}  // namespace bigdb::memtable
