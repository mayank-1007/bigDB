#include "db.h"
#include <gtest/gtest.h>
#include <filesystem>
#include <thread>
#include <chrono>

using namespace bigdb::db;
namespace fs = std::filesystem;

class FlushTest : public ::testing::Test {
protected:
    void SetUp() override {
        temp_dir = testing::TempDir() + "/flush_test_" + std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
        fs::create_directories(temp_dir);
    }
    void TearDown() override {
        fs::remove_all(temp_dir);
    }
    std::string temp_dir;
};

TEST_F(FlushTest, BackgroundFlushDoesNotBreakReads) {
    Options opts = DefaultOptions();
    opts.DataDir = temp_dir;
    opts.WALFileName = "wal.log";
    opts.MemtableThreshold = 2;
    opts.SparseIndexGap = 1;

    auto db = DB::Open(opts);
    ASSERT_NE(db, nullptr);

    EXPECT_NO_THROW(db->Put({'a'}, {'1'}));
    EXPECT_NO_THROW(db->Put({'b'}, {'2'}));
    EXPECT_NO_THROW(db->Put({'c'}, {'3'}));

    auto [value, ok] = db->Get({'a'});
    ASSERT_TRUE(ok);
    std::string valA(value.begin(), value.end());
    EXPECT_EQ(valA, "1");

    std::string sstDir = (fs::path(temp_dir) / opts.SSTableDirName).string();
    bool found = false;
    
    auto deadline = std::chrono::system_clock::now() + std::chrono::seconds(2);
    while (std::chrono::system_clock::now() < deadline) {
        if (fs::exists(sstDir)) {
            for (const auto& entry : fs::directory_iterator(sstDir)) {
                if (entry.is_regular_file()) {
                    found = true;
                    break;
                }
            }
        }
        if (found) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
    }

    EXPECT_TRUE(found) << "expected background flush to create an sstable";
    db->Close();
}
