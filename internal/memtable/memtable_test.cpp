#include "memtable.h"
#include <gtest/gtest.h>

using namespace bigdb::memtable;
using namespace bigdb::record;

TEST(MemtableTest, PutGetDelete) {
    Memtable m;

    Record put = Record::new_put({'k','1'}, {'v','1'}, 1);
    m.Put(put);

    auto [got, ok] = m.Get({'k','1'});
    ASSERT_TRUE(ok) << "expected key to exist";
    
    std::string valStr(got.value.begin(), got.value.end());
    ASSERT_EQ(valStr, "v1") << "unexpected value";

    Record del = Record::new_delete({'k','1'}, 2);
    m.Delete(del);

    auto [got2, ok2] = m.Get({'k','1'});
    ASSERT_FALSE(ok2) << "expected deleted key to be hidden";
}
