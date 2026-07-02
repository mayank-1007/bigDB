package compaction

import (
	"fmt"

	"bigdb/internal/record"
	"bigdb/internal/sstable"
)

func Merge(paths []string, outPath string, sparseGap int) error {
	if len(paths) == 0 {
		return nil
	}

	latest := make(map[string]record.Record)

	for _, path := range paths {
		file, err := sstable.Open(path)
		if err != nil {
			return fmt.Errorf("open sstable %s: %w", path, err)
		}

		records, err := file.All()
		closeErr := file.Close()
		if err != nil {
			return fmt.Errorf("scan sstable %s: %w", path, err)
		}
		if closeErr != nil {
			return fmt.Errorf("close sstable %s: %w", path, closeErr)
		}

		for _, rec := range records {
			key := string(rec.Key)
			existing, ok := latest[key]
			if !ok || rec.Timestamp > existing.Timestamp {
				latest[key] = rec
			}
		}
	}

	snapshot := make(map[string]record.Record)

	for k, rec := range latest {
		snapshot[k] = rec
	}

	if err := sstable.WriteSnapshot(outPath, snapshot, sparseGap); err != nil {
		return fmt.Errorf("write compacted sstable: %w", err)
	}

	return nil
}
