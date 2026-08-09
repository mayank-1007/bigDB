#include "reader.h"
#include "writer.h"
#include <gtest/gtest.h>
#include <filesystem>

using namespace bigdb::sstable;
using namespace bigdb::record;
namespace fs = std::filesystem;

class SSTableTest : public ::testing::Test {
protected:
    void SetUp() override {
        temp_dir = testing::TempDir() + "/sstable_test_" + std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
        fs::create_directories(temp_dir);
    }
    void TearDown() override {
        fs::remove_all(temp_dir);
    }
    std::string temp_dir;
};

TEST_F(SSTableTest, ChecksumDetectsCorruption) {
    std::string path = temp_dir + "/l0-sst-00000000000000000001.sst";

    std::map<std::string, Record> snapshot = {
        {"a", Record::new_put({'a'}, {'v','a','l','u','e','-','a'}, 1)},
        {"b", Record::new_put({'b'}, {'v','a','l','u','e','-','b'}, 2)}
    };

    EXPECT_NO_THROW(WriteSnapshot(path, snapshot, 1));

    std::fstream f(path, std::ios::in | std::ios::out | std::ios::binary);
    ASSERT_TRUE(f.good());
    
    f.seekg(0, std::ios::end);
    size_t size = f.tellg();
    ASSERT_GE(size, 20);

    f.seekg(12, std::ios::beg);
    char byte;
    f.read(&byte, 1);
    byte ^= 0xFF;
    f.seekp(12, std::ios::beg);
    f.write(&byte, 1);
    f.close();

    auto file = File::Open(path);
    ASSERT_NE(file, nullptr);

    EXPECT_THROW(file->Get({'a'}), CorruptRecordError);
}
