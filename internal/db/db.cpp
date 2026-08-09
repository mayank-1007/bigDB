#include "db.h"

#include <algorithm>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <thread>

#include "../recovery/recovery.h"

namespace fs = std::filesystem;

namespace bigdb::db {
namespace {
std::string JoinPath(const std::string& base, const std::string& child) {
    return (fs::path(base) / child).string();
}

bool TryRemoveFileWithRetry(const std::string& path) {
    if (path.empty()) {
        return true;
    }

    for (int attempt = 0; attempt < 5; ++attempt) {
        std::error_code ec;
        const bool removed = std::filesystem::remove(path, ec);
        if (removed || ec == std::errc::no_such_file_or_directory) {
            return true;
        }
        if (ec != std::errc::device_or_resource_busy) {
            return false;
        }
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
    }

    return false;
}

std::vector<std::string> ListFiles(const std::string& dir) {
    std::vector<std::string> out;
    if (!fs::exists(dir)) {
        return out;
    }
    for (const auto& entry : fs::directory_iterator(dir)) {
        if (entry.is_regular_file()) {
            out.push_back(entry.path().string());
        }
    }
    std::sort(out.begin(), out.end());
    return out;
}
}  // namespace

Options DefaultOptions() {
    return Options{};
}

DB::DB(const Options& opts) : opts_(opts) {
    fs::create_directories(JoinPath(opts_.DataDir, opts_.SSTableDirName));
    const std::string walPath = JoinPath(opts_.DataDir, opts_.WALFileName);
    wal_ = std::make_unique<wal::WAL>(walPath);
    active_ = std::make_shared<memtable::Memtable>();

    auto records = wal_->Replay();
    for (const auto& rec : records) {
        if (rec.isDeleted) {
            active_->Delete(rec);
        } else {
            active_->Put(rec);
        }
    }

    std::vector<std::string> files = ListFiles(JoinPath(opts_.DataDir, opts_.SSTableDirName));
    for (const auto& filePath : files) {
        const auto file = std::make_shared<sstable::SSTable>(filePath);
        sstables_[file->Level()].push_back(file);
        nextSSTable_ = std::max(nextSSTable_, file->Sequence());
    }

    RefreshCounts();
}

DB::~DB() {
    try {
        Close();
    } catch (...) {
    }
}

void DB::Put(const std::string& key, const std::string& value) {
    record::Record rec = record::NewPut(key, value, std::chrono::duration_cast<std::chrono::nanoseconds>(std::chrono::system_clock::now().time_since_epoch()).count());

    std::lock_guard<std::mutex> lock(mu_);
    wal_->Append(rec);
    active_->Put(rec);
    stats_.lastWriteMS.store(1);
    RefreshCounts();

    if (opts_.MemtableThreshold > 0 && active_->Size() >= static_cast<size_t>(opts_.MemtableThreshold)) {
        FreezeActive();
    }
}

void DB::Delete(const std::string& key) {
    record::Record rec = record::NewDelete(key, std::chrono::duration_cast<std::chrono::nanoseconds>(std::chrono::system_clock::now().time_since_epoch()).count());

    std::lock_guard<std::mutex> lock(mu_);
    wal_->Append(rec);
    active_->Delete(rec);
    stats_.lastWriteMS.store(1);
    RefreshCounts();

    if (opts_.MemtableThreshold > 0 && active_->Size() >= static_cast<size_t>(opts_.MemtableThreshold)) {
        FreezeActive();
    }
}

std::pair<std::string, bool> DB::Get(const std::string& key) const {
    std::lock_guard<std::mutex> lock(mu_);
    auto rec = active_->Get(key);
    if (active_->Contains(key)) {
        return {rec.value, !rec.isDeleted};
    }

    for (auto it = frozen_.rbegin(); it != frozen_.rend(); ++it) {
        auto snapshotRec = it->second.find(key);
        if (snapshotRec != it->second.end()) {
            return {snapshotRec->second.value, !snapshotRec->second.isDeleted};
        }
    }

    for (const auto& [level, files] : sstables_) {
        for (const auto& file : files) {
            const auto fileRec = file->Get(key);
            if (fileRec.key == key) {
                return {fileRec.value, !fileRec.isDeleted};
            }
        }
    }

    return {"", false};
}

void DB::Compact() {
    std::lock_guard<std::mutex> lock(compactMu_);
    std::vector<std::string> paths;
    for (const auto& [level, files] : sstables_) {
        if (level == 0 && files.size() >= static_cast<size_t>(opts_.LevelCompactionFanout)) {
            for (const auto& file : files) {
                paths.push_back(file->Path());
            }
            break;
        }
    }
    if (paths.empty()) {
        return;
    }

    ++nextSSTable_;
    const std::string outPath = SSTablePath(1, nextSSTable_);
    compaction::Merge(paths, outPath, opts_.SparseIndexGap);

    std::lock_guard<std::mutex> sstLock(sstMu_);
    auto merged = std::make_shared<sstable::SSTable>(outPath);
    sstables_[1].push_back(merged);
    sstables_[0].clear();
    RefreshCounts();
}

void DB::Close() {
    if (closed_) {
        return;
    }
    closed_ = true;
    if (flushThread_.joinable()) {
        flushThread_.join();
    }
    if (compactionThread_.joinable()) {
        compactionThread_.join();
    }
    wal_->Close();
}

telemetry::Stats& DB::Stats() {
    return stats_;
}

void DB::FreezeActive() {
    auto snapshot = active_->Snapshot();
    std::string archivedWal = wal_->Rotate();

    ++nextSSTable_;
    const uint64_t id = nextSSTable_;
    const std::string path = SSTablePath(0, id);
    {
        std::lock_guard<std::mutex> lock(flushMu_);
        frozen_.push_back({id, snapshot});
    }

    active_ = std::make_shared<memtable::Memtable>();
    RefreshCounts();

    if (flushThread_.joinable()) {
        flushThread_.join();
    }
    flushThread_ = std::thread([this, id, path, snapshot, archivedWal]() {
        this->FlushSnapshotAsync(id, path, snapshot, archivedWal);
    });
}

void DB::FlushSnapshotAsync(uint64_t id, const std::string& path, const std::map<std::string, record::Record>& snapshot, const std::string& walArchive) {
    try {
        sstable::WriteSnapshot(path, snapshot, opts_.SparseIndexGap);
        std::lock_guard<std::mutex> lock(mu_);
        if (closed_) {
            return;
        }
        MarkFlushComplete(id, path);
        if (!walArchive.empty()) {
            (void)TryRemoveFileWithRetry(walArchive);
        }
        stats_.flushes.fetch_add(1);
        LaunchCompaction();
    } catch (const std::exception& ex) {
        const std::string message = ex.what();
        if (message.find("device or resource busy") == std::string::npos) {
            std::cerr << "flush failed: " << message << std::endl;
        }
    }
}

void DB::MarkFlushComplete(uint64_t id, const std::string& path) {
    auto file = std::make_shared<sstable::SSTable>(path);
    std::lock_guard<std::mutex> sstLock(sstMu_);
    sstables_[file->Level()].push_back(file);
    std::lock_guard<std::mutex> flushLock(flushMu_);
    frozen_.erase(std::remove_if(frozen_.begin(), frozen_.end(), [id](const auto& entry) { return entry.first == id; }), frozen_.end());
    RefreshCounts();
}

void DB::RefreshCounts() {
    stats_.totalKeys.store(static_cast<int64_t>(active_->Size()));
    std::lock_guard<std::mutex> lock(sstMu_);
    int64_t total = 0;
    for (const auto& [_, files] : sstables_) {
        total += static_cast<int64_t>(files.size());
    }
    stats_.sstables.store(total);
}

std::string DB::SSTablePath(uint32_t level, uint64_t seq) const {
    return JoinPath(JoinPath(opts_.DataDir, opts_.SSTableDirName), "l" + std::to_string(level) + "-sst-" + std::to_string(seq) + ".sst");
}

void DB::LaunchCompaction() {
    if (compactionRunning_) {
        return;
    }
    compactionRunning_ = true;
    compactionThread_ = std::thread([this]() {
        Compact();
        compactionRunning_ = false;
    });
}

void DB::CleanupArchivedWALs() {
}

}  // namespace bigdb::db
