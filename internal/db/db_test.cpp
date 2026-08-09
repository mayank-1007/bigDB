#include "db.h"
#include <gtest/gtest.h>
#include <filesystem>
#include <thread>
#include <chrono>

using namespace bigdb::db;
namespace fs = std::filesystem;

class DBTest : public ::testing::Test {
protected:
    void SetUp() override {
        temp_dir = testing::TempDir() + "/db_test_" + std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
        fs::create_directories(temp_dir);
    }
    void TearDown() override {
        fs::remove_all(temp_dir);
    }
    std::string temp_dir;
};

TEST_F(DBTest, PutGetRestartRecovery) {
    Options opts = DefaultOptions();
    opts.DataDir = temp_dir;
    opts.WALFileName = "wal.log";

    auto db = DB::Open(opts);
    ASSERT_NE(db, nullptr);

    EXPECT_NO_THROW(db->Put({'n','a','m','e'}, {'m','a','y','a','n','k'}));
    
    db->Close();

    auto reopened = DB::Open(opts);
    ASSERT_NE(reopened, nullptr);

    auto [value, ok] = reopened->Get({'n','a','m','e'});
    ASSERT_TRUE(ok) << "expected recovered key";
    
    std::string valStr(value.begin(), value.end());
    EXPECT_EQ(valStr, "mayank") << "unexpected value";

    reopened->Close();
}

TEST_F(DBTest, FlushToSSTableAndReadBack) {
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

    auto [valueA, okA] = db->Get({'a'});
    ASSERT_TRUE(okA);
    std::string valA(valueA.begin(), valueA.end());
    EXPECT_EQ(valA, "1");

    auto [valueC, okC] = db->Get({'c'});
    ASSERT_TRUE(okC);
    std::string valC(valueC.begin(), valueC.end());
    EXPECT_EQ(valC, "3");

    db->Close();
}
