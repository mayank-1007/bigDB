#include "filter.h"
#include <gtest/gtest.h>
#include <string>

using namespace bigdb::bloom;

TEST(BloomFilterTest, MightContainForInsertedKey) {
    auto f = Filter::NewForKeys(10);
    
    std::string key1_str = "alpha";
    std::string key2_str = "beta";
    std::vector<uint8_t> key1(key1_str.begin(), key1_str.end());
    std::vector<uint8_t> key2(key2_str.begin(), key2_str.end());
    
    f->Add(key1);
    f->Add(key2);

    EXPECT_TRUE(f->MightContain(key1)) << "expected inserted key to be present";
    EXPECT_TRUE(f->MightContain(key2)) << "expected inserted key to be present";
    
    std::string key3_str = "gamma";
    std::vector<uint8_t> key3(key3_str.begin(), key3_str.end());
    // gamma might technically collide, but very unlikely for 10 keys and 1024 bits.
    EXPECT_FALSE(f->MightContain(key3));
}
