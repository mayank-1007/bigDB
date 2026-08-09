#include <filesystem>
#include <functional>
#include <iostream>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "internal/db/db.h"

namespace fs = std::filesystem;

static int RunScenario(const std::string& name, const std::function<void(const std::string&)>& fn) {
    const std::string dir = "./build/validation_" + name;
    fs::remove_all(dir);
    fs::create_directories(dir);

    try {
        fn(dir);
    } catch (const std::exception& ex) {
        std::cerr << "[fail] " << name << ": " << ex.what() << "\n";
        return 1;
    }

    std::cout << "[ok] " << name << "\n";
    return 0;
}

int main() {
    std::vector<std::pair<std::string, std::function<void(const std::string&)>>> scenarios = {
        {"put_get", [](const std::string& dir) {
            bigdb::db::Options opts = bigdb::db::DefaultOptions();
            opts.DataDir = dir;
            opts.MemtableThreshold = 10000;
            opts.SparseIndexGap = 2;
            bigdb::db::DB db(opts);
            db.Put("k1", "v1");
            auto [value, ok] = db.Get("k1");
            if (!ok || value != "v1") {
                throw std::runtime_error("put/get failed");
            }
            db.Close();
        }},
        {"delete", [](const std::string& dir) {
            bigdb::db::Options opts = bigdb::db::DefaultOptions();
            opts.DataDir = dir;
            opts.MemtableThreshold = 10000;
            opts.SparseIndexGap = 2;
            bigdb::db::DB db(opts);
            db.Put("k2", "v2");
            db.Delete("k2");
            auto [value, ok] = db.Get("k2");
            if (ok || !value.empty()) {
                throw std::runtime_error("delete semantics failed");
            }
            db.Close();
        }},
        {"compact", [](const std::string& dir) {
            bigdb::db::Options opts = bigdb::db::DefaultOptions();
            opts.DataDir = dir;
            opts.MemtableThreshold = 10000;
            opts.SparseIndexGap = 1;
            bigdb::db::DB db(opts);
            for (int i = 0; i < 6; ++i) {
                db.Put("k" + std::to_string(i), "v" + std::to_string(i));
            }
            db.Compact();
            db.Close();
        }},
        {"recovery", [](const std::string& dir) {
            bigdb::db::Options opts = bigdb::db::DefaultOptions();
            opts.DataDir = dir;
            opts.MemtableThreshold = 10000;
            opts.SparseIndexGap = 2;
            {
                bigdb::db::DB db(opts);
                db.Put("recovered", "value");
                db.Close();
            }
            bigdb::db::DB reopened(opts);
            auto [value, ok] = reopened.Get("recovered");
            if (!ok || value != "value") {
                throw std::runtime_error("recovery failed");
            }
            reopened.Close();
        }}
    };

    for (const auto& [name, fn] : scenarios) {
        if (RunScenario(name, fn) != 0) {
            return 1;
        }
    }

    return 0;
}
