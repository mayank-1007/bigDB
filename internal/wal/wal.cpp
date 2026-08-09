#include "wal.h"

#include <cstdio>
#include <filesystem>
#include <fstream>
#include <stdexcept>
#include <vector>

namespace fs = std::filesystem;

namespace bigdb::wal {
namespace {
constexpr uint32_t kChecksumFlag = 1u << 31;

std::string ArchivePath(const std::string& path, uint64_t seq) {
    fs::path p(path);
    return (p.parent_path() / (p.stem().string() + "-" + std::to_string(seq) + p.extension().string())).string();
}

void WriteUint32(std::FILE* file, uint32_t value) {
    unsigned char buf[4] = {
        static_cast<unsigned char>((value >> 24) & 0xff),
        static_cast<unsigned char>((value >> 16) & 0xff),
        static_cast<unsigned char>((value >> 8) & 0xff),
        static_cast<unsigned char>(value & 0xff),
    };
    if (std::fwrite(buf, 1, 4, file) != 4) {
        throw std::runtime_error("failed to write wal entry");
    }
}

bool ReadUint32(std::istream& input, uint32_t& out) {
    unsigned char buf[4] = {};
    if (!input.read(reinterpret_cast<char*>(buf), sizeof(buf))) {
        return false;
    }
    out = (static_cast<uint32_t>(buf[0]) << 24) |
          (static_cast<uint32_t>(buf[1]) << 16) |
          (static_cast<uint32_t>(buf[2]) << 8) |
          static_cast<uint32_t>(buf[3]);
    return true;
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

WAL::WAL(const std::string& path) : path_(path), dir_(fs::path(path).parent_path().string()), activePath_(path) {
    fs::create_directories(dir_);
    file_ = std::fopen(path.c_str(), "ab+");
    if (!file_) {
        throw std::runtime_error("failed to open wal file");
    }
    std::fflush(file_);
    std::fseek(file_, 0, SEEK_END);
}

WAL::~WAL() {
    Close();
}

void WAL::Append(const record::Record& rec) {
    if (!file_) {
        throw std::runtime_error("wal is closed");
    }

    auto payload = record::Encode(rec);
    std::vector<unsigned char> entry;
    entry.reserve(4 + payload.size() + 4);
    uint32_t len = static_cast<uint32_t>(payload.size()) | kChecksumFlag;
    WriteUint32(file_, len);
    if (std::fwrite(payload.data(), 1, payload.size(), file_) != payload.size()) {
        throw std::runtime_error("failed to write wal payload");
    }
    uint32_t crc = Checksum(payload);
    WriteUint32(file_, crc);
    std::fflush(file_);
}

std::string WAL::Rotate() {
    if (!file_) {
        throw std::runtime_error("wal is closed");
    }
    std::fflush(file_);
    std::fclose(file_);
    file_ = nullptr;

    const std::string archive = ArchivePath(path_, nextArchive_++);
    std::rename(path_.c_str(), archive.c_str());
    file_ = std::fopen(path_.c_str(), "wb+");
    if (!file_) {
        throw std::runtime_error("failed to recreate wal file");
    }
    return archive;
}

std::vector<record::Record> WAL::Replay() const {
    std::vector<record::Record> out;
    std::ifstream in(path_, std::ios::binary);
    if (!in) {
        return out;
    }

    while (in) {
        uint32_t len = 0;
        if (!ReadUint32(in, len)) {
            break;
        }
        const bool hasCRC = (len & kChecksumFlag) != 0;
        const uint32_t payloadLen = len & ~kChecksumFlag;
        if (payloadLen == 0) {
            break;
        }
        std::vector<unsigned char> payload(payloadLen);
        if (!in.read(reinterpret_cast<char*>(payload.data()), payloadLen)) {
            break;
        }
        if (hasCRC) {
            uint32_t crc = 0;
            if (!ReadUint32(in, crc)) {
                break;
            }
            if (crc != Checksum(payload)) {
                throw std::runtime_error("wal crc mismatch");
            }
        }
        out.push_back(record::Decode(payload));
    }

    return out;
}

void WAL::Close() {
    if (file_) {
        std::fclose(file_);
        file_ = nullptr;
    }
}

}  // namespace bigdb::wal
