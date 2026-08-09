#include <filesystem>
#include <iostream>
#include <string>

#include "internal/db/db.h"

int main() {
    namespace fs = std::filesystem;
    const std::string dataDir = "./build/test_data";
    fs::remove_all(dataDir);
    fs::create_directories(dataDir);

    bigdb::db::Options opts = bigdb::db::DefaultOptions();
    opts.DataDir = dataDir;
    opts.MemtableThreshold = 2;

    bigdb::db::DB db(opts);
    db.Put("k1", "v1");
    db.Put("k2", "v2");
    auto [value, ok] = db.Get("k1");
    if (!ok || value != "v1") {
        std::cerr << "get after put failed" << std::endl;
        return 1;
    }

    db.Delete("k1");
    auto [deletedValue, deletedOk] = db.Get("k1");
    if (deletedOk || !deletedValue.empty()) {
        std::cerr << "delete semantics failed" << std::endl;
        return 1;
    }

    db.Compact();
    db.Close();
    return 0;
}
