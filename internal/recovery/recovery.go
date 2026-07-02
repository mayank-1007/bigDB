package recovery

import (
	"bigdb/internal/memtable"
	"bigdb/internal/record"
	"bigdb/internal/wal"
)

func ReplayToMemtable(w *wal.WAL, m *memtable.Memtable) error {
	return w.ReplayAll(func(rec record.Record) error {
		if rec.IsDeleted {
			m.Delete(rec)
			return nil
		}
		m.Put(rec)
		return nil
	})
}

// func DebugReplaySummary(w *wal.WAL) error {
// 	count := 0
// 	return w.Replay(func(rec record.Record) error {
// 		count++
// 		_ = rec
// 		return nil
// 	})
// }
