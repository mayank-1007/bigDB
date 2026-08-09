#include <chrono>
#include <filesystem>
#include <iostream>
#include <string>
#include <thread>
#include <vector>

#include "internal/db/db.h"

namespace fs = std::filesystem;

int main() {
    const std::string dir = "./build/benchmark_data";
    fs::remove_all(dir);
    fs::create_directories(dir);

    bigdb::db::Options opts = bigdb::db::DefaultOptions();
    opts.DataDir = dir;
    opts.MemtableThreshold = 10000;
    opts.SparseIndexGap = 8;

    bigdb::db::DB db(opts);

    const int rounds = 5;
    const int operationsPerRound = 2000;
    double totalElapsedMs = 0.0;

    for (int round = 0; round < rounds; ++round) {
        const auto start = std::chrono::steady_clock::now();
        for (int i = 0; i < operationsPerRound; ++i) {
            db.Put("k" + std::to_string(round * operationsPerRound + i), "v" + std::to_string(round * operationsPerRound + i));
        }
        const auto end = std::chrono::steady_clock::now();
        const double elapsedMs = std::chrono::duration<double, std::milli>(end - start).count();
        totalElapsedMs += elapsedMs;
        std::cout << "benchmark_round_" << (round + 1) << "_elapsed_ms=" << elapsedMs << "\n";
    }

    const auto readStart = std::chrono::steady_clock::now();
    for (int i = 0; i < 5000; ++i) {
        auto [value, ok] = db.Get("k1");
        (void)value;
        (void)ok;
    }
    const auto readEnd = std::chrono::steady_clock::now();
    const double readElapsedMs = std::chrono::duration<double, std::milli>(readEnd - readStart).count();

    const int concurrentWriters = 4;
    const int concurrentOpsPerThread = 1000;
    std::vector<std::thread> threads;
    threads.reserve(concurrentWriters);

    const auto concurrentStart = std::chrono::steady_clock::now();
    for (int t = 0; t < concurrentWriters; ++t) {
        threads.emplace_back([&db, t, concurrentOpsPerThread]() {
            for (int i = 0; i < concurrentOpsPerThread; ++i) {
                db.Put("concurrent_" + std::to_string(t) + "_" + std::to_string(i), "value");
            }
        });
    }
    for (auto& thread : threads) {
        thread.join();
    }
    const auto concurrentEnd = std::chrono::steady_clock::now();
    const double concurrentElapsedMs = std::chrono::duration<double, std::milli>(concurrentEnd - concurrentStart).count();

    const double avgElapsedMs = totalElapsedMs / rounds;
    const int totalOperations = rounds * operationsPerRound;
    std::cout << "benchmark_rounds=" << rounds << "\n";
    std::cout << "benchmark_operations_per_round=" << operationsPerRound << "\n";
    std::cout << "benchmark_total_operations=" << totalOperations << "\n";
    std::cout << "benchmark_average_elapsed_ms=" << avgElapsedMs << "\n";
    std::cout << "benchmark_average_ops_per_sec=" << (totalOperations / (avgElapsedMs / 1000.0)) << "\n";
    std::cout << "benchmark_hot_read_elapsed_ms=" << readElapsedMs << "\n";
    std::cout << "benchmark_hot_read_ops_per_sec=" << (5000 / (readElapsedMs / 1000.0)) << "\n";
    std::cout << "benchmark_concurrent_writes_elapsed_ms=" << concurrentElapsedMs << "\n";
    std::cout << "benchmark_concurrent_writes_ops_per_sec=" << ((concurrentWriters * concurrentOpsPerThread) / (concurrentElapsedMs / 1000.0)) << "\n";

    db.Close();
    return 0;
}
