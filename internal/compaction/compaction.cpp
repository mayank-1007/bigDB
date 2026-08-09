#include "compaction.h"

#include <algorithm>
#include <map>
#include <stdexcept>

#include "../record/record.h"
#include "../sstable/format.h"

namespace bigdb::compaction {

void Merge(const std::vector<std::string>& paths, const std::string& outPath, int sparseGap) {
    std::map<std::string, record::Record> latest;

    for (const auto& path : paths) {
        sstable::SSTable file(path);
        for (const auto& rec : file.All()) {
            auto it = latest.find(rec.key);
            if (it == latest.end() || rec.timestamp > it->second.timestamp) {
                latest[rec.key] = rec;
            }
        }
    }

    sstable::WriteSnapshot(outPath, latest, sparseGap);
}

}  // namespace bigdb::compaction
