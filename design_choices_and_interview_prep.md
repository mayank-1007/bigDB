# Design Choices and Interview Preparation

When discussing this project in interviews, you may face deep counter-questioning about the design choices made during the C++ rewrite. Here is a breakdown of the key decisions, alternative approaches, and how to defend them.

## 1. Testing Framework: Google Test (gtest) vs. Custom Test Runner

**The Decision:** We used Google Test (gtest) via CMake's `FetchContent`.

### Why use Google Test (Pros of external dependency)
1. **Industry Standard:** `gtest` is widely used in production C++ codebases. Familiarity with it signals industry readiness.
2. **Rich Assertions:** It provides a comprehensive set of assertions (`EXPECT_EQ`, `ASSERT_TRUE`, death tests, etc.) that would take significant time to implement robustly from scratch.
3. **Test Discovery and Filtering:** `gtest` automatically discovers tests and allows running specific tests or suites via command-line flags (`--gtest_filter`), which is invaluable for a storage engine with many edge cases (e.g., corruption, concurrency, recovery).
4. **Mocking Support:** While not used everywhere, `gmock` comes bundled, which is useful for testing I/O without hitting the actual disk.

### The Alternative: Custom Test Runner (Pros of zero-dependency)
1. **No External Dependencies:** Reduces the build configuration complexity and download time. The project would be purely self-contained.
2. **Leaner Binary:** A custom runner can be extremely lightweight.
3. **Learning Opportunity:** Writing a custom runner demonstrates a deep understanding of C++ macros, reflection (or lack thereof), and test orchestration.

### How to defend the choice in an interview
*If the interviewer asks: "Why did you pull in a heavy external dependency for tests instead of writing a simple assert script?"*

**Your Answer:**
> "I considered writing a custom test runner to keep the project dependency-free. However, a database engine requires rigorous testing—especially around concurrent compactions, crash recovery, and data corruption. Writing a custom runner that handles test isolation, detailed failure reporting, and test filtering (which is crucial when debugging a specific failed compaction scenario) would become a project in itself, distracting from the core goal of building the storage engine. `gtest` is the industry standard, and by using CMake's `FetchContent`, I was able to integrate it seamlessly without requiring the user to manually install any packages on their machine."

## 2. Concurrency: `std::thread` vs. Go's Goroutines

**The Context:** The original Go implementation used goroutines (`go func()`) for background flushes and leveled compactions. In C++, we use OS-level threads (`std::thread`), `std::mutex`, and `std::condition_variable`.

### Goroutines vs OS Threads
1. **Overhead:** Goroutines are lightweight (starting at ~2KB of stack space) and multiplexed onto a smaller number of OS threads by the Go runtime. `std::thread` maps directly to an OS thread, which has a higher memory footprint (often 1MB-8MB stack) and context-switching cost.
2. **Scalability:** In Go, spawning thousands of concurrent tasks is trivial. In C++, spawning thousands of OS threads will lead to resource exhaustion.
3. **Synchronization:** Go prefers channels (`chan`) for synchronization and passing data, whereas C++ relies heavily on mutexes and condition variables (though C++ can implement thread-safe queues).

### How to defend the choice in an interview
*If the interviewer asks: "How does the concurrency model of your C++ port compare to the original Go version?"*

**Your Answer:**
> "In the Go version, background tasks like memtable flushes and compactions are handled via goroutines, which are extremely lightweight and managed by the Go runtime scheduler. In C++, I implemented these using `std::thread` along with `std::mutex` and `std::condition_variable` for synchronization. Because `std::thread` maps 1:1 to OS threads, it carries higher overhead. However, for this storage engine, we don't spawn thousands of background tasks—we typically only need a few dedicated background threads for flushing and compaction. Therefore, the OS-level thread overhead is negligible for our architecture, and using `std::thread` provides predictable, fine-grained control over our concurrency primitives without needing an external asynchronous runtime (like Boost.Asio or a custom thread pool)."

## 3. Standard Library vs Third-Party Libraries for Core Logic

**The Decision:** The core engine (WAL, Memtable, SSTable, Compaction) is written using purely the C++ Standard Library (C++17).

**Your Answer:**
> "I chose C++17 to leverage `std::filesystem` for portable file operations, `std::string_view` for zero-copy string parsing (critical for read performance), and `std::optional`. I intentionally avoided heavy third-party libraries (other than `gtest`) for the core logic to demonstrate a deep understanding of standard C++ features and to keep the codebase focused on the storage engine mechanics."
