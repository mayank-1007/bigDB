#include "filter.h"
#include <cstring>
#include <stdexcept>

namespace bigdb::bloom {

namespace {

std::pair<uint64_t, uint64_t> HashPair(const std::vector<uint8_t>& key) {
    // FNV-1a 64-bit
    uint64_t h1 = 14695981039346656037ULL;
    for (uint8_t b : key) {
        h1 ^= b;
        h1 *= 1099511628211ULL;
    }

    // FNV-1a 32-bit
    uint32_t h2_32 = 2166136261U;
    for (uint8_t b : key) {
        h2_32 ^= b;
        h2_32 *= 16777619U;
    }
    
    uint64_t h2 = h2_32;
    if (h2 == 0) {
        h2 = 0x9e3779b97f4a7c15ULL;
    }

    return {h1, h2};
}

} // namespace

std::unique_ptr<Filter> Filter::New(uint32_t bit_count, uint32_t hash_count) {
    if (bit_count < 8) {
        bit_count = 8;
    }
    if (hash_count == 0) {
        hash_count = 1;
    }

    auto f = std::make_unique<Filter>();
    f->bit_count = bit_count;
    f->hash_count = hash_count;
    f->bits.resize((bit_count + 7) / 8, 0);
    return f;
}

std::unique_ptr<Filter> Filter::NewForKeys(int key_count) {
    uint32_t bit_count = 1024;
    if (key_count > 0) {
        bit_count = static_cast<uint32_t>(key_count * 16);
        if (bit_count < 1024) {
            bit_count = 1024;
        }
    }
    return New(bit_count, 7);
}

void Filter::Add(const std::vector<uint8_t>& key) {
    auto [h1, h2] = HashPair(key);
    if (h2 == 0) {
        h2 = 1;
    }

    for (uint32_t i = 0; i < hash_count; i++) {
        uint64_t pos = (h1 + static_cast<uint64_t>(i) * h2) % bit_count;
        uint64_t byteIdx = pos / 8;
        uint8_t bitMask = 1 << (pos % 8);
        bits[byteIdx] |= bitMask;
    }
}

bool Filter::MightContain(const std::vector<uint8_t>& key) const {
    if (bit_count == 0 || bits.empty()) {
        return true;
    }

    auto [h1, h2] = HashPair(key);
    if (h2 == 0) {
        h2 = 1;
    }

    for (uint32_t i = 0; i < hash_count; i++) {
        uint64_t pos = (h1 + static_cast<uint64_t>(i) * h2) % bit_count;
        uint64_t byteIdx = pos / 8;
        uint8_t bitMask = 1 << (pos % 8);
        if ((bits[byteIdx] & bitMask) == 0) {
            return false;
        }
    }
    return true;
}

std::vector<uint8_t> Filter::MarshalBinary() const {
    std::vector<uint8_t> out(8 + bits.size());
    out[0] = (bit_count >> 24) & 0xFF;
    out[1] = (bit_count >> 16) & 0xFF;
    out[2] = (bit_count >> 8) & 0xFF;
    out[3] = bit_count & 0xFF;

    out[4] = (hash_count >> 24) & 0xFF;
    out[5] = (hash_count >> 16) & 0xFF;
    out[6] = (hash_count >> 8) & 0xFF;
    out[7] = hash_count & 0xFF;

    std::memcpy(out.data() + 8, bits.data(), bits.size());
    return out;
}

std::unique_ptr<Filter> Filter::UnmarshalBinary(const std::vector<uint8_t>& data) {
    if (data.size() < 8) {
        return nullptr;
    }

    uint32_t bit_count = (static_cast<uint32_t>(data[0]) << 24) |
                         (static_cast<uint32_t>(data[1]) << 16) |
                         (static_cast<uint32_t>(data[2]) << 8) |
                         static_cast<uint32_t>(data[3]);

    uint32_t hash_count = (static_cast<uint32_t>(data[4]) << 24) |
                          (static_cast<uint32_t>(data[5]) << 16) |
                          (static_cast<uint32_t>(data[6]) << 8) |
                          static_cast<uint32_t>(data[7]);

    if (bit_count == 0 || hash_count == 0) {
        return nullptr;
    }

    auto f = std::make_unique<Filter>();
    f->bit_count = bit_count;
    f->hash_count = hash_count;
    f->bits.resize(data.size() - 8);
    std::memcpy(f->bits.data(), data.data() + 8, data.size() - 8);
    
    return f;
}

} // namespace bigdb::bloom
