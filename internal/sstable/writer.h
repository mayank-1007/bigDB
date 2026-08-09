#pragma once

#include "format.h"
#include "../record/record.h"
#include <string>
#include <map>

namespace bigdb::sstable {

void WriteSnapshot(const std::string& path, const std::map<std::string, record::Record>& snapshot, int sparse_gap);

} // namespace bigdb::sstable
