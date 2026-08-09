#pragma once

#include <string>
#include <vector>

namespace bigdb::compaction {

void Merge(const std::vector<std::string>& paths, const std::string& outPath, int sparseGap);

}  // namespace bigdb::compaction
