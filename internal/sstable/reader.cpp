#include "reader.h"
#include <filesystem>
#include <algorithm>
#include <iostream>
#include <cstring>

namespace fs = std::filesystem;

namespace bigdb::sstable {

namespace {

uint32_t crc32_ieee(const std::vector<uint8_t>& data) {
    uint32_t crc = 0xFFFFFFFF;
    for (uint8_t byte : data) {
        crc ^= byte;
        for (int i = 0; i < 8; i++) {
            uint32_t mask = -(crc & 1);
            crc = (crc >> 1) ^ (0xEDB88320 & mask);
        }
    }
    return ~crc;
}

std::string filepathBase(const std::string& path) {
    fs::path p(path);
    return p.filename().string();
}

bool parseLevelAndSeq(const std::string& path, uint32_t& level, uint64_t& seq) {
    std::string name = filepathBase(path);

    if (name.rfind("l", 0) == 0 && name.find("-sst-") != std::string::npos && name.length() >= 4 && name.compare(name.length() - 4, 4, ".sst") == 0) {
        size_t pos = name.find("-sst-");
        std::string levelRaw = name.substr(1, pos - 1);
        std::string seqRaw = name.substr(pos + 5, name.length() - pos - 5 - 4);

        try {
            level = std::stoul(levelRaw);
            seq = std::stoull(seqRaw);
            return true;
        } catch (...) {
            return false;
        }
    }

    if (name.rfind("sst-", 0) == 0 && name.length() >= 4 && name.compare(name.length() - 4, 4, ".sst") == 0) {
        std::string seqRaw = name.substr(4, name.length() - 4 - 4);
        try {
            level = 0;
            seq = std::stoull(seqRaw);
            return true;
        } catch (...) {
            return false;
        }
    }

    return false;
}

std::pair<record::Record, uint64_t> readRecordAt(std::ifstream& fh, uint64_t offset, uint64_t limit) {
    if (offset + 4 > limit) {
        throw std::runtime_error("EOF");
    }

    fh.seekg(offset, std::ios::beg);
    uint8_t lenBuf[4];
    if (!fh.read(reinterpret_cast<char*>(lenBuf), 4)) {
        throw std::runtime_error("read record length failed");
    }

    uint32_t rawLen = (static_cast<uint32_t>(lenBuf[0]) << 24) |
                      (static_cast<uint32_t>(lenBuf[1]) << 16) |
                      (static_cast<uint32_t>(lenBuf[2]) << 8) |
                      static_cast<uint32_t>(lenBuf[3]);

    bool hasCRC = (rawLen & ChecksumFlag) != 0;
    uint32_t payloadLen = rawLen & ~ChecksumFlag;

    uint64_t total = 4 + payloadLen;
    if (hasCRC) total += 4;

    if (offset + total > limit) {
        throw CorruptRecordError();
    }

    std::vector<uint8_t> payload(payloadLen);
    if (!fh.read(reinterpret_cast<char*>(payload.data()), payloadLen)) {
        throw std::runtime_error("read record payload failed");
    }

    if (hasCRC) {
        uint8_t crcBuf[4];
        if (!fh.read(reinterpret_cast<char*>(crcBuf), 4)) {
            throw std::runtime_error("read record crc failed");
        }
        uint32_t want = (static_cast<uint32_t>(crcBuf[0]) << 24) |
                        (static_cast<uint32_t>(crcBuf[1]) << 16) |
                        (static_cast<uint32_t>(crcBuf[2]) << 8) |
                        static_cast<uint32_t>(crcBuf[3]);
        uint32_t got = crc32_ieee(payload);
        if (want != got) {
            throw CorruptRecordError();
        }
    }

    record::Record rec = record::Record::decode(payload);
    return {rec, offset + total};
}

std::unique_ptr<bloom::Filter> readBloom(std::ifstream& fh, const Footer& footer) {
    std::vector<uint8_t> data(footer.bloom_size);
    fh.seekg(footer.bloom_offset, std::ios::beg);
    if (!fh.read(reinterpret_cast<char*>(data.data()), footer.bloom_size)) {
        throw std::runtime_error("read bloom filter failed");
    }

    auto filter = bloom::Filter::UnmarshalBinary(data);
    if (!filter) {
        throw CorruptFooterError();
    }
    return filter;
}

std::vector<SparseIndexEntry> readIndex(std::ifstream& fh, const Footer& footer) {
    if (footer.index_count == 0) {
        return {};
    }

    std::vector<SparseIndexEntry> index;
    index.reserve(footer.index_count);
    uint64_t offset = footer.index_offset;

    for (uint32_t i = 0; i < footer.index_count; i++) {
        fh.seekg(offset, std::ios::beg);
        uint8_t keyLenBuf[4];
        if (!fh.read(reinterpret_cast<char*>(keyLenBuf), 4)) {
            throw std::runtime_error("read index key len failed");
        }
        offset += 4;

        uint32_t keyLen = (static_cast<uint32_t>(keyLenBuf[0]) << 24) |
                          (static_cast<uint32_t>(keyLenBuf[1]) << 16) |
                          (static_cast<uint32_t>(keyLenBuf[2]) << 8) |
                          static_cast<uint32_t>(keyLenBuf[3]);

        std::vector<uint8_t> key(keyLen);
        if (!fh.read(reinterpret_cast<char*>(key.data()), keyLen)) {
            throw std::runtime_error("read index key failed");
        }
        offset += keyLen;

        uint8_t offBuf[8];
        if (!fh.read(reinterpret_cast<char*>(offBuf), 8)) {
            throw std::runtime_error("read index offset failed");
        }
        offset += 8;

        uint64_t off = (static_cast<uint64_t>(offBuf[0]) << 56) |
                       (static_cast<uint64_t>(offBuf[1]) << 48) |
                       (static_cast<uint64_t>(offBuf[2]) << 40) |
                       (static_cast<uint64_t>(offBuf[3]) << 32) |
                       (static_cast<uint64_t>(offBuf[4]) << 24) |
                       (static_cast<uint64_t>(offBuf[5]) << 16) |
                       (static_cast<uint64_t>(offBuf[6]) << 8) |
                       static_cast<uint64_t>(offBuf[7]);

        index.push_back({key, off});
    }

    return index;
}

} // namespace

File::File(const std::string& path, std::unique_ptr<std::ifstream> fh, std::vector<SparseIndexEntry> index, Footer footer, std::unique_ptr<bloom::Filter> bloom, uint32_t level, uint64_t seq)
    : path_(path), fh_(std::move(fh)), index_(std::move(index)), footer_(footer), bloom_(std::move(bloom)), level_(level), seq_(seq) {}

File::~File() {
    Close();
}

void File::Close() {
    if (fh_) {
        fh_->close();
        fh_.reset();
    }
}

std::unique_ptr<File> File::Open(const std::string& path) {
    auto fh = std::make_unique<std::ifstream>(path, std::ios::binary);
    if (!*fh) {
        throw std::runtime_error("open sstable failed");
    }

    uint32_t level = 0;
    uint64_t seq = 0;
    if (!parseLevelAndSeq(path, level, seq)) {
        throw CorruptFooterError();
    }

    fh->seekg(0, std::ios::end);
    size_t size = fh->tellg();

    Footer footer;
    if (size >= NewFooterSize) {
        fh->seekg(size - NewFooterSize, std::ios::beg);
        std::vector<uint8_t> footerBytes(NewFooterSize);
        fh->read(reinterpret_cast<char*>(footerBytes.data()), NewFooterSize);

        std::string magic(reinterpret_cast<char*>(footerBytes.data() + 32), 8);
        magic.erase(std::find(magic.begin(), magic.end(), '\0'), magic.end());
        
        if (magic == NewMagicValue) {
            footer.index_offset = (static_cast<uint64_t>(footerBytes[0]) << 56) | (static_cast<uint64_t>(footerBytes[1]) << 48) | (static_cast<uint64_t>(footerBytes[2]) << 40) | (static_cast<uint64_t>(footerBytes[3]) << 32) | (static_cast<uint64_t>(footerBytes[4]) << 24) | (static_cast<uint64_t>(footerBytes[5]) << 16) | (static_cast<uint64_t>(footerBytes[6]) << 8) | static_cast<uint64_t>(footerBytes[7]);
            footer.index_count = (static_cast<uint32_t>(footerBytes[8]) << 24) | (static_cast<uint32_t>(footerBytes[9]) << 16) | (static_cast<uint32_t>(footerBytes[10]) << 8) | static_cast<uint32_t>(footerBytes[11]);
            footer.record_count = (static_cast<uint32_t>(footerBytes[12]) << 24) | (static_cast<uint32_t>(footerBytes[13]) << 16) | (static_cast<uint32_t>(footerBytes[14]) << 8) | static_cast<uint32_t>(footerBytes[15]);
            footer.sparse_gap = (static_cast<uint32_t>(footerBytes[16]) << 24) | (static_cast<uint32_t>(footerBytes[17]) << 16) | (static_cast<uint32_t>(footerBytes[18]) << 8) | static_cast<uint32_t>(footerBytes[19]);
            footer.bloom_offset = (static_cast<uint64_t>(footerBytes[20]) << 56) | (static_cast<uint64_t>(footerBytes[21]) << 48) | (static_cast<uint64_t>(footerBytes[22]) << 40) | (static_cast<uint64_t>(footerBytes[23]) << 32) | (static_cast<uint64_t>(footerBytes[24]) << 24) | (static_cast<uint64_t>(footerBytes[25]) << 16) | (static_cast<uint64_t>(footerBytes[26]) << 8) | static_cast<uint64_t>(footerBytes[27]);
            footer.bloom_size = (static_cast<uint32_t>(footerBytes[28]) << 24) | (static_cast<uint32_t>(footerBytes[29]) << 16) | (static_cast<uint32_t>(footerBytes[30]) << 8) | static_cast<uint32_t>(footerBytes[31]);
            footer.magic = NewMagicValue;
        } else {
            fh->seekg(size - LegacyFooterSize, std::ios::beg);
            std::vector<uint8_t> legacyFooterBytes(LegacyFooterSize);
            fh->read(reinterpret_cast<char*>(legacyFooterBytes.data()), LegacyFooterSize);
            
            std::string legacyMagic(reinterpret_cast<char*>(legacyFooterBytes.data() + 20), 8);
            if (legacyMagic != LegacyMagicValue) {
                throw CorruptFooterError();
            }
            
            footer.index_offset = (static_cast<uint64_t>(legacyFooterBytes[0]) << 56) | (static_cast<uint64_t>(legacyFooterBytes[1]) << 48) | (static_cast<uint64_t>(legacyFooterBytes[2]) << 40) | (static_cast<uint64_t>(legacyFooterBytes[3]) << 32) | (static_cast<uint64_t>(legacyFooterBytes[4]) << 24) | (static_cast<uint64_t>(legacyFooterBytes[5]) << 16) | (static_cast<uint64_t>(legacyFooterBytes[6]) << 8) | static_cast<uint64_t>(legacyFooterBytes[7]);
            footer.index_count = (static_cast<uint32_t>(legacyFooterBytes[8]) << 24) | (static_cast<uint32_t>(legacyFooterBytes[9]) << 16) | (static_cast<uint32_t>(legacyFooterBytes[10]) << 8) | static_cast<uint32_t>(legacyFooterBytes[11]);
            footer.record_count = (static_cast<uint32_t>(legacyFooterBytes[12]) << 24) | (static_cast<uint32_t>(legacyFooterBytes[13]) << 16) | (static_cast<uint32_t>(legacyFooterBytes[14]) << 8) | static_cast<uint32_t>(legacyFooterBytes[15]);
            footer.sparse_gap = (static_cast<uint32_t>(legacyFooterBytes[16]) << 24) | (static_cast<uint32_t>(legacyFooterBytes[17]) << 16) | (static_cast<uint32_t>(legacyFooterBytes[18]) << 8) | static_cast<uint32_t>(legacyFooterBytes[19]);
            footer.magic = LegacyMagicValue;
        }
    } else {
        throw CorruptFooterError();
    }

    std::vector<SparseIndexEntry> index = readIndex(*fh, footer);

    std::unique_ptr<bloom::Filter> bf;
    if (footer.magic == NewMagicValue && footer.bloom_offset > 0 && footer.bloom_size > 0) {
        bf = readBloom(*fh, footer);
    }

    return std::unique_ptr<File>(new File(path, std::move(fh), std::move(index), footer, std::move(bf), level, seq));
}

std::pair<record::Record, bool> File::Get(const std::vector<uint8_t>& key) {
    if (!fh_) {
        throw std::runtime_error("sstable closed");
    }

    if (bloom_ && !bloom_->MightContain(key)) {
        return {record::Record{}, false};
    }

    uint64_t startOffset = 0;
    uint64_t endOffset = footer_.index_offset;
    if (footer_.bloom_offset > 0) {
        endOffset = footer_.bloom_offset;
    }

    if (!index_.empty()) {
        auto it = std::lower_bound(index_.begin(), index_.end(), key, [](const SparseIndexEntry& entry, const std::vector<uint8_t>& k) {
            return entry.key < k;
        });

        if (it == index_.end() || it->key > key) {
            if (it != index_.begin()) {
                --it;
            }
        }

        startOffset = it->offset;
        auto next = it + 1;
        if (next != index_.end()) {
            endOffset = next->offset;
        }
    }

    uint64_t offset = startOffset;
    while (offset < endOffset) {
        try {
            auto [rec, nextOffset] = readRecordAt(*fh_, offset, endOffset);
            if (rec.key == key) {
                return {rec, !rec.is_deleted};
            }
            if (rec.key > key) {
                return {record::Record{}, false};
            }
            offset = nextOffset;
        } catch (const std::exception& e) {
            if (std::string(e.what()) == "EOF") {
                break;
            }
            throw;
        }
    }

    return {record::Record{}, false};
}

std::vector<record::Record> File::All() {
    if (!fh_) {
        throw std::runtime_error("sstable closed");
    }

    std::vector<record::Record> records;
    uint64_t offset = 0;
    uint64_t limit = footer_.index_offset;
    if (footer_.bloom_offset > 0) {
        limit = footer_.bloom_offset;
    }

    while (offset < limit) {
        try {
            auto [rec, nextOffset] = readRecordAt(*fh_, offset, limit);
            records.push_back(rec);
            offset = nextOffset;
        } catch (const std::exception& e) {
            if (std::string(e.what()) == "EOF") {
                break;
            }
            throw;
        }
    }
    return records;
}

} // namespace bigdb::sstable
