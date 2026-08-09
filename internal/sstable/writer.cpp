#include "writer.h"
#include <fstream>
#include <filesystem>
#include <algorithm>
#include <vector>
#include <stdexcept>
#include <cstring>
#include <iostream>

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

void writeFull(std::ofstream& f, const std::vector<uint8_t>& data) {
    f.write(reinterpret_cast<const char*>(data.data()), data.size());
    if (f.fail()) {
        throw std::runtime_error("write failed");
    }
}

} // namespace

void WriteSnapshot(const std::string& path, const std::map<std::string, record::Record>& snapshot, int sparse_gap) {
    if (sparse_gap <= 0) {
        sparse_gap = 10;
    }

    std::string tmpPath = path + ".tmp";
    std::ofstream f(tmpPath, std::ios::binary | std::ios::trunc);
    if (!f) {
        throw std::runtime_error("create sstable tmp failed");
    }

    std::vector<std::string> keys;
    keys.reserve(snapshot.size());
    for (const auto& [k, _] : snapshot) {
        keys.push_back(k);
    }
    // std::map is already sorted, but let's be sure.
    std::sort(keys.begin(), keys.end());

    std::vector<SparseIndexEntry> indexEntries;

    for (size_t i = 0; i < keys.size(); i++) {
        const auto& k = keys[i];
        const auto& rec = snapshot.at(k);

        uint64_t offset = f.tellp();

        if (i % sparse_gap == 0) {
            indexEntries.push_back({rec.key, offset});
        }

        std::vector<uint8_t> payload = rec.encode();

        std::vector<uint8_t> lenBuf(4);
        uint32_t lenWithFlag = static_cast<uint32_t>(payload.size()) | ChecksumFlag;
        lenBuf[0] = (lenWithFlag >> 24) & 0xFF;
        lenBuf[1] = (lenWithFlag >> 16) & 0xFF;
        lenBuf[2] = (lenWithFlag >> 8) & 0xFF;
        lenBuf[3] = lenWithFlag & 0xFF;

        std::vector<uint8_t> crcBuf(4);
        uint32_t crc = crc32_ieee(payload);
        crcBuf[0] = (crc >> 24) & 0xFF;
        crcBuf[1] = (crc >> 16) & 0xFF;
        crcBuf[2] = (crc >> 8) & 0xFF;
        crcBuf[3] = crc & 0xFF;

        writeFull(f, lenBuf);
        writeFull(f, payload);
        writeFull(f, crcBuf);
    }

    auto bf = bloom::Filter::NewForKeys(keys.size());
    for (const auto& k : keys) {
        std::vector<uint8_t> keyBytes(k.begin(), k.end());
        bf->Add(keyBytes);
    }

    uint64_t bloomOffset = f.tellp();
    std::vector<uint8_t> bloomBytes = bf->MarshalBinary();
    writeFull(f, bloomBytes);

    uint64_t indexOffset = f.tellp();

    for (const auto& entry : indexEntries) {
        std::vector<uint8_t> keyLenBuf(4);
        uint32_t keyLen = entry.key.size();
        keyLenBuf[0] = (keyLen >> 24) & 0xFF;
        keyLenBuf[1] = (keyLen >> 16) & 0xFF;
        keyLenBuf[2] = (keyLen >> 8) & 0xFF;
        keyLenBuf[3] = keyLen & 0xFF;

        std::vector<uint8_t> offsetBuf(8);
        offsetBuf[0] = (entry.offset >> 56) & 0xFF;
        offsetBuf[1] = (entry.offset >> 48) & 0xFF;
        offsetBuf[2] = (entry.offset >> 40) & 0xFF;
        offsetBuf[3] = (entry.offset >> 32) & 0xFF;
        offsetBuf[4] = (entry.offset >> 24) & 0xFF;
        offsetBuf[5] = (entry.offset >> 16) & 0xFF;
        offsetBuf[6] = (entry.offset >> 8) & 0xFF;
        offsetBuf[7] = entry.offset & 0xFF;

        writeFull(f, keyLenBuf);
        writeFull(f, entry.key);
        writeFull(f, offsetBuf);
    }

    std::vector<uint8_t> footer(NewFooterSize, 0);
    // index_offset
    footer[0] = (indexOffset >> 56) & 0xFF; footer[1] = (indexOffset >> 48) & 0xFF;
    footer[2] = (indexOffset >> 40) & 0xFF; footer[3] = (indexOffset >> 32) & 0xFF;
    footer[4] = (indexOffset >> 24) & 0xFF; footer[5] = (indexOffset >> 16) & 0xFF;
    footer[6] = (indexOffset >> 8) & 0xFF;  footer[7] = indexOffset & 0xFF;
    
    // index_count
    uint32_t indexCount = indexEntries.size();
    footer[8] = (indexCount >> 24) & 0xFF; footer[9] = (indexCount >> 16) & 0xFF;
    footer[10] = (indexCount >> 8) & 0xFF; footer[11] = indexCount & 0xFF;
    
    // record_count
    uint32_t recordCount = keys.size();
    footer[12] = (recordCount >> 24) & 0xFF; footer[13] = (recordCount >> 16) & 0xFF;
    footer[14] = (recordCount >> 8) & 0xFF;  footer[15] = recordCount & 0xFF;
    
    // sparse_gap
    footer[16] = (sparse_gap >> 24) & 0xFF; footer[17] = (sparse_gap >> 16) & 0xFF;
    footer[18] = (sparse_gap >> 8) & 0xFF;  footer[19] = sparse_gap & 0xFF;
    
    // bloom_offset
    footer[20] = (bloomOffset >> 56) & 0xFF; footer[21] = (bloomOffset >> 48) & 0xFF;
    footer[22] = (bloomOffset >> 40) & 0xFF; footer[23] = (bloomOffset >> 32) & 0xFF;
    footer[24] = (bloomOffset >> 24) & 0xFF; footer[25] = (bloomOffset >> 16) & 0xFF;
    footer[26] = (bloomOffset >> 8) & 0xFF;  footer[27] = bloomOffset & 0xFF;
    
    // bloom_size
    uint32_t bloomSize = bloomBytes.size();
    footer[28] = (bloomSize >> 24) & 0xFF; footer[29] = (bloomSize >> 16) & 0xFF;
    footer[30] = (bloomSize >> 8) & 0xFF;  footer[31] = bloomSize & 0xFF;
    
    // magic
    std::memcpy(footer.data() + 32, NewMagicValue, std::strlen(NewMagicValue));

    writeFull(f, footer);
    f.flush();
    f.close();

    fs::rename(tmpPath, path);
}

} // namespace bigdb::sstable
