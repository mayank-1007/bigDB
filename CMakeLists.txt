cmake_minimum_required(VERSION 3.16)
project(bigdb_cpp LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

enable_testing()

set(BIGDB_SOURCES
    internal/db/db.cpp
    internal/record/record.cpp
    internal/memtable/memtable.cpp
    internal/telemetry/telemetry.cpp
    internal/wal/wal.cpp
    internal/sstable/format.cpp
    internal/compaction/compaction.cpp
    internal/recovery/recovery.cpp
)

add_library(bigdb_core STATIC ${BIGDB_SOURCES})
target_include_directories(bigdb_core PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})

add_executable(bigdb
    cmd/db/main.cpp
)
target_link_libraries(bigdb PRIVATE bigdb_core)

target_include_directories(bigdb PRIVATE ${CMAKE_CURRENT_SOURCE_DIR})

add_executable(db_smoke_test
    tests/db_smoke_test.cpp
)
target_link_libraries(db_smoke_test PRIVATE bigdb_core)
target_include_directories(db_smoke_test PRIVATE ${CMAKE_CURRENT_SOURCE_DIR})

add_executable(db_validation_tests
    tests/validation_runner.cpp
)
target_link_libraries(db_validation_tests PRIVATE bigdb_core)
target_include_directories(db_validation_tests PRIVATE ${CMAKE_CURRENT_SOURCE_DIR})

add_executable(db_benchmark
    tests/benchmark_db.cpp
)
target_link_libraries(db_benchmark PRIVATE bigdb_core)
target_include_directories(db_benchmark PRIVATE ${CMAKE_CURRENT_SOURCE_DIR})

add_test(NAME db_smoke COMMAND db_smoke_test)
add_test(NAME db_validation COMMAND db_validation_tests)
