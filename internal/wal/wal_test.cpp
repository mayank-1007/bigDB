#include "wal.h"
#include <gtest/gtest.h>
#include <filesystem>
#include "../record/record.h"

using namespace bigdb::wal;
using namespace bigdb::record;
namespace fs = std::filesystem;

class WALTest : public ::testing::Test {
protected:
    void SetUp() override {
        temp_dir = testing::TempDir() + "/wal_test_" + std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
        fs::create_directories(temp_dir);
    }
    void TearDown() override {
        fs::remove_all(temp_dir);
    }
    std::string temp_dir;
};

TEST_F(WALTest, AppendAndReplay) {
    std::string path = temp_dir + "/wal.log";
    auto w = WAL::Open(path);
    ASSERT_NE(w, nullptr);

    std::vector<Record> inputs = {
        Record::new_put({'a'}, {'1'}, 1),
        Record::new_put({'b'}, {'2'}, 2),
        Record::new_delete({'a'}, 3)
    };

    for (const auto& rec : inputs) {
        EXPECT_NO_THROW(w->Append(rec));
    }

    std::vector<Record> got;
    EXPECT_NO_THROW(w->Replay([&](const Record& r) {
        got.push_back(r);
    }));

    ASSERT_EQ(got.size(), inputs.size());

    for (size_t i = 0; i < inputs.size(); i++) {
        EXPECT_EQ(got[i].key, inputs[i].key);
        EXPECT_EQ(got[i].value, inputs[i].value);
        EXPECT_EQ(got[i].timestamp, inputs[i].timestamp);
        EXPECT_EQ(got[i].is_deleted, inputs[i].is_deleted);
    }
}
