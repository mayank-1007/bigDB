#include <iostream>
#include <string>

#include "../../internal/db/db.h"

int main(int argc, char** argv) {
    std::string dataDir = "./data";
    std::string putKey;
    std::string putValue;
    std::string getKey;
    std::string delKey;
    bool compact = false;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "-data-dir" && i + 1 < argc) {
            dataDir = argv[++i];
        } else if (arg == "-put-key" && i + 1 < argc) {
            putKey = argv[++i];
        } else if (arg == "-put-value" && i + 1 < argc) {
            putValue = argv[++i];
        } else if (arg == "-get-key" && i + 1 < argc) {
            getKey = argv[++i];
        } else if (arg == "-del-key" && i + 1 < argc) {
            delKey = argv[++i];
        } else if (arg == "-compact") {
            compact = true;
        }
    }

    bigdb::db::Options opts = bigdb::db::DefaultOptions();
    opts.DataDir = dataDir;

    auto engine = std::make_unique<bigdb::db::DB>(opts);

    if (!putKey.empty()) {
        engine->Put(putKey, putValue);
        std::cout << "put ok\n";
    }
    if (!delKey.empty()) {
        engine->Delete(delKey);
        std::cout << "delete ok\n";
    }
    if (!getKey.empty()) {
        const auto [value, ok] = engine->Get(getKey);
        if (!ok) {
            std::cout << "not found\n";
        } else {
            std::cout << getKey << " => " << value << "\n";
        }
    }
    if (compact) {
        engine->Compact();
        std::cout << "compaction ok\n";
    }

    engine->Close();
    return 0;
}
