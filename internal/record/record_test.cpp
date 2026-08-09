#include "record.h"
#include <gtest/gtest.h>
#include <string>
#include <vector>

using namespace bigdb::record;

TEST(RecordTest, EncodeDecodeRoundTrip) {
    std::string key_str = "user:1";
    std::string value_str = "alice";
    std::vector<uint8_t> key(key_str.begin(), key_str.end());
    std::vector<uint8_t> value(value_str.begin(), value_str.end());
    int64_t timestamp = 123456789;

    Record original = Record::new_put(key, value, timestamp);

    std::vector<uint8_t> encoded = original.encode();
    Record decoded = Record::decode(encoded);

    EXPECT_EQ(decoded.key, original.key);
    EXPECT_EQ(decoded.value, original.value);
    EXPECT_EQ(decoded.timestamp, original.timestamp);
    EXPECT_EQ(decoded.is_deleted, original.is_deleted);
}

TEST(RecordTest, DeleteRecordRoundTrip) {
    std::string key_str = "user:2";
    std::vector<uint8_t> key(key_str.begin(), key_str.end());
    int64_t timestamp = 999;

    Record original = Record::new_delete(key, timestamp);

    std::vector<uint8_t> encoded = original.encode();
    Record decoded = Record::decode(encoded);

    EXPECT_TRUE(decoded.is_deleted);
    EXPECT_EQ(decoded.key, original.key);
    EXPECT_TRUE(decoded.value.empty());
    EXPECT_EQ(decoded.timestamp, original.timestamp);
}
