#pragma once

#include <vector>
#include <cstdint>
#include <memory>
#include <optional>

namespace bigdb::bloom {

struct Filter {
    uint32_t bit_count;
    uint32_t hash_count;
    std::vector<uint8_t> bits;

    static std::unique_ptr<Filter> New(uint32_t bit_count, uint32_t hash_count);
    static std::unique_ptr<Filter> NewForKeys(int key_count);

    void Add(const std::vector<uint8_t>& key);
    bool MightContain(const std::vector<uint8_t>& key) const;

    std::vector<uint8_t> MarshalBinary() const;
    static std::unique_ptr<Filter> UnmarshalBinary(const std::vector<uint8_t>& data);
};

} // namespace bigdb::bloom
