#pragma once

#include <vector>
#include <string>

#include "../memtable/memtable.h"
#include "../wal/wal.h"

namespace bigdb::recovery {

void ReplayToMemtable(const wal::WAL& wal, memtable::Memtable& memtable);

}  // namespace bigdb::recovery
