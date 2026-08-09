#include "record.h"

#include <stdexcept>
#include <cstring>

namespace bigdb::record {

namespace {
void PutUint32(std::vector<unsigned char>& out, uint32_t value) {
    out.push_back(static_cast<unsigned char>((value >> 24) & 0xff));
    out.push_back(static_cast<unsigned char>((value >> 16) & 0xff));
    out.push_back(static_cast<unsigned char>((value >> 8) & 0xff));
    out.push_back(static_cast<unsigned char>(value & 0xff));
}

uint32_t ReadUint32(const std::vector<unsigned char>& data, size_t offset) {
    return ((uint32_t(data[offset]) << 24) |
            (uint32_t(data[offset + 1]) << 16) |
            (uint32_t(data[offset + 2]) << 8) |
            uint32_t(data[offset + 3]));
}

void PutUint64(std::vector<unsigned char>& out, uint64_t value) {
    for (int i = 7; i >= 0; --i) {
        out.push_back(static_cast<unsigned char>((value >> (i * 8)) & 0xff));
    }
}

uint64_t ReadUint64(const std::vector<unsigned char>& data, size_t offset) {
    uint64_t value = 0;
    for (size_t i = 0; i < 8; ++i) {
        value = (value << 8) | data[offset + i];
    }
    return value;
}
}  // namespace

Record NewPut(const std::string& key, const std::string& value, int64_t timestamp) {
    return Record{key, value, timestamp, false};
}

Record NewDelete(const std::string& key, int64_t timestamp) {
    return Record{key, "", timestamp, true};
}

std::vector<unsigned char> Encode(const Record& record) {
    std::vector<unsigned char> out;
    out.reserve(4 + 4 + 8 + 1 + record.key.size() + record.value.size());

    PutUint32(out, static_cast<uint32_t>(record.key.size()));
    PutUint32(out, static_cast<uint32_t>(record.value.size()));
    PutUint64(out, static_cast<uint64_t>(record.timestamp));
    out.push_back(record.isDeleted ? 1 : 0);
    out.insert(out.end(), record.key.begin(), record.key.end());
    out.insert(out.end(), record.value.begin(), record.value.end());
    return out;
}

Record Decode(const std::vector<unsigned char>& data) {
    if (data.size() < 17) {
        throw std::runtime_error("invalid record encoding");
    }

    const uint32_t keyLen = ReadUint32(data, 0);
    const uint32_t valueLen = ReadUint32(data, 4);
    const uint64_t timestamp = ReadUint64(data, 8);
    const bool deleted = data[16] != 0;

    const size_t expected = 17 + keyLen + valueLen;
    if (data.size() != expected) {
        throw std::runtime_error("invalid record encoding length");
    }

    Record rec;
    rec.timestamp = static_cast<int64_t>(timestamp);
    rec.isDeleted = deleted;
    rec.key.assign(data.begin() + 17, data.begin() + 17 + keyLen);
    rec.value.assign(data.begin() + 17 + keyLen, data.end());
    return rec;
}

}  // namespace bigdb::record
