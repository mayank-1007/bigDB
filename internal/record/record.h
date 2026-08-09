#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace bigdb::record {

struct Record {
    std::string key;
    std::string value;
    int64_t timestamp = 0;
    bool isDeleted = false;
};

Record NewPut(const std::string& key, const std::string& value, int64_t timestamp);
Record NewDelete(const std::string& key, int64_t timestamp);
std::vector<unsigned char> Encode(const Record& record);
Record Decode(const std::vector<unsigned char>& data);

}  // namespace bigdb::record
