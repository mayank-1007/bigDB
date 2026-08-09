#include "recovery.h"

namespace bigdb::recovery {

void ReplayToMemtable(const wal::WAL& wal, memtable::Memtable& memtable) {
    auto records = wal.Replay();
    for (const auto& rec : records) {
        if (rec.isDeleted) {
            memtable.Delete(rec);
        } else {
            memtable.Put(rec);
        }
    }
}

}  // namespace bigdb::recovery
