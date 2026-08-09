#include "db.h"
#include <filesystem>
#include <fstream>
#include <iostream>

namespace fs = std::filesystem;

namespace bigdb::db {

const char* walCheckpointFileName = "wal.checkpoint";

std::string DB::walCheckpointPath() const {
    return (fs::path(opts_.DataDir) / walCheckpointFileName).string();
}

uint64_t DB::loadWALCheckpoint() const {
    std::string path = walCheckpointPath();
    if (!fs::exists(path)) return 0;

    std::ifstream f(path);
    if (!f) return 0;

    std::string raw;
    f >> raw;
    if (raw.empty()) return 0;

    try {
        return std::stoull(raw);
    } catch (...) {
        return 0;
    }
}

void DB::saveWALCheckpoint(uint64_t seq) {
    std::string tmp = walCheckpointPath() + ".tmp";
    std::ofstream f(tmp);
    if (!f) {
        std::cerr << "write wal checkpoint tmp failed\n";
        return;
    }
    f << seq;
    f.close();

    try {
        fs::rename(tmp, walCheckpointPath());
    } catch (const std::exception& e) {
        std::cerr << "rename wal checkpoint: " << e.what() << "\n";
        try { fs::remove(tmp); } catch (...) {}
    }
}

void DB::advanceWALCheckpoint(const std::string& archivedPath) {
    std::string name = fs::path(archivedPath).filename().string();
    std::string activeBase = fs::path(opts_.WALFileName).filename().string();
    
    std::string ext = fs::path(activeBase).extension().string();
    std::string stem = fs::path(activeBase).stem().string();
    std::string prefix = stem + "-";

    uint64_t seq = 0;
    if (name.rfind(prefix, 0) == 0 && name.length() >= ext.length() && name.compare(name.length() - ext.length(), ext.length(), ext) == 0) {
        std::string raw = name.substr(prefix.length(), name.length() - prefix.length() - ext.length());
        try {
            seq = std::stoull(raw);
        } catch (...) {
            return;
        }
    } else {
        return;
    }

    uint64_t current = loadWALCheckpoint();
    if (seq <= current) return;

    saveWALCheckpoint(seq);
}

void DB::cleanupArchivedWALs() {
    uint64_t checkpoint = loadWALCheckpoint();
    if (checkpoint == 0) return;

    std::string activeBase = fs::path(opts_.WALFileName).filename().string();
    std::string ext = fs::path(activeBase).extension().string();
    std::string stem = fs::path(activeBase).stem().string();
    std::string prefix = stem + "-";

    for (const auto& entry : fs::directory_iterator(opts_.DataDir)) {
        if (!entry.is_regular_file()) continue;
        std::string name = entry.path().filename().string();
        
        uint64_t seq = 0;
        if (name.rfind(prefix, 0) == 0 && name.length() >= ext.length() && name.compare(name.length() - ext.length(), ext.length(), ext) == 0) {
            std::string raw = name.substr(prefix.length(), name.length() - prefix.length() - ext.length());
            try {
                seq = std::stoull(raw);
            } catch (...) {
                continue;
            }
            if (seq <= checkpoint) {
                try { fs::remove(entry.path()); } catch (...) {}
            }
        }
    }
}

} // namespace bigdb::db
