#include "wal.h"
#include <gtest/gtest.h>
#include <filesystem>
#include <fstream>
#include "../record/record.h"

using namespace bigdb::wal;
using namespace bigdb::record;
namespace fs = std::filesystem;

class WALChecksumTest : public ::testing::Test {
protected:
    void SetUp() override {
        temp_dir = testing::TempDir() + "/wal_checksum_test_" + std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
        fs::create_directories(temp_dir);
    }
    void TearDown() override {
        fs::remove_all(temp_dir);
    }
    std::string temp_dir;
};

TEST_F(WALChecksumTest, ChecksumDetectsCorruption) {
    std::string path = temp_dir + "/wal.log";
    auto w = WAL::Open(path);
    ASSERT_NE(w, nullptr);

    EXPECT_NO_THROW(w->Append(Record::new_put({'k', '1'}, {'v', '1'}, 1)));
    w->Close();

    std::fstream f(path, std::ios::in | std::ios::out | std::ios::binary);
    ASSERT_TRUE(f.good());
    
    f.seekg(0, std::ios::end);
    size_t size = f.tellg();
    ASSERT_GE(size, 12);

    f.seekg(10, std::ios::beg);
    char byte;
    f.read(&byte, 1);
    
    byte ^= 0xFF;
    
    f.seekp(10, std::ios::beg);
    f.write(&byte, 1);
    f.close();

    auto w2 = WAL::Open(path);
    ASSERT_NE(w2, nullptr);

    EXPECT_THROW(w2->ReplayAll([](const Record&) {}), std::runtime_error);
}
