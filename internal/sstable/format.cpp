#include "format.h"

#include <algorithm>
#include <filesystem>
#include <fstream>
#include <map>
#include <stdexcept>
#include <vector>

namespace fs = std::filesystem;

namespace bigdb::sstable {
namespace {
void WriteUint32(std::ofstream& out, uint32_t value) {
    unsigned char buf[4] = {
        static_cast<unsigned char>((value >> 24) & 0xff),
        static_cast<unsigned char>((value >> 16) & 0xff),
        static_cast<unsigned char>((value >> 8) & 0xff),
        static_cast<unsigned char>(value & 0xff),
    };
    out.write(reinterpret_cast<const char*>(buf), 4);
}

void WriteUint64(std::ofstream& out, uint64_t value) {
    unsigned char buf[8];
    for (int i = 7; i >= 0; --i) {
        buf[7 - i] = static_cast<unsigned char>((value >> (i * 8)) & 0xff);
    }
    out.write(reinterpret_cast<const char*>(buf), 8);
}

uint32_t Checksum(const std::vector<unsigned char>& payload) {
    uint32_t crc = 0xFFFFFFFFu;
    for (unsigned char b : payload) {
        crc ^= b;
        for (int i = 0; i < 8; ++i) {
            crc = (crc >> 1) ^ ((crc & 1u) ? 0xEDB88320u : 0u);
        }
    }
    return ~crc;
}
}  // namespace

SSTable::SSTable(const std::string& path) : path_(path) {
    fs::path p(path);
    const std::string filename = p.stem().string();
    const auto pos = filename.find("-sst-");
    if (pos == std::string::npos) {
        throw std::runtime_error("invalid sstable name");
    }
    const std::string levelStr = filename.substr(1, pos - 1);
    const std::string seqStr = filename.substr(pos + 5);
    level_ = static_cast<uint32_t>(std::stoul(levelStr));
    seq_ = std::stoull(seqStr);
}

SSTable::~SSTable() = default;

std::vector<record::Record> SSTable::All() const {
    std::ifstream in(path_, std::ios::binary);
    if (!in) {
        throw std::runtime_error("failed to open sstable");
    }

    std::vector<record::Record> out;
    while (in) {
        uint32_t len = 0;
        in.read(reinterpret_cast<char*>(&len), sizeof(len));
        if (!in) {
            break;
        }
        const uint32_t payloadLen = len & 0x7FFFFFFFu;
        std::vector<unsigned char> payload(payloadLen);
        in.read(reinterpret_cast<char*>(payload.data()), payloadLen);
        if (!in) {
            break;
        }
        uint32_t crc = 0;
        in.read(reinterpret_cast<char*>(&crc), sizeof(crc));
        if (!in) {
            break;
        }
        if (crc != Checksum(payload)) {
            throw std::runtime_error("sstable crc mismatch");
        }
        out.push_back(record::Decode(payload));
    }
    return out;
}

record::Record SSTable::Get(const std::string& key) const {
    for (const auto& rec : All()) {
        if (rec.key == key) {
            return rec;
        }
    }
    return {};
}

uint32_t SSTable::Level() const { return level_; }
uint64_t SSTable::Sequence() const { return seq_; }
const std::string& SSTable::Path() const { return path_; }

void WriteSnapshot(const std::string& path, const std::map<std::string, record::Record>& snapshot, int sparseGap) {
    std::ofstream out(path, std::ios::binary | std::ios::trunc);
    if (!out) {
        throw std::runtime_error("failed to create sstable");
    }

    std::vector<std::string> keys;
    keys.reserve(snapshot.size());
    for (const auto& [k, _] : snapshot) {
        keys.push_back(k);
    }
    std::sort(keys.begin(), keys.end());

    std::vector<SparseIndexEntry> index;
    for (size_t i = 0; i < keys.size(); ++i) {
        const auto& key = keys[i];
        const auto& rec = snapshot.at(key);
        if (i % std::max(1, sparseGap) == 0) {
            index.push_back(SparseIndexEntry{key, static_cast<uint64_t>(out.tellp())});
        }
        const auto payload = record::Encode(rec);
        WriteUint32(out, static_cast<uint32_t>(payload.size()));
        out.write(reinterpret_cast<const char*>(payload.data()), payload.size());
        uint32_t crc = Checksum(payload);
        WriteUint32(out, crc);
    }

    out.close();
}

}  // namespace bigdb::sstable
