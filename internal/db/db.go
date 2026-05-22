package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"bigtable/internal/compaction"
	"bigtable/internal/memtable"
	"bigtable/internal/recovery"
	"bigtable/internal/record"
	"bigtable/internal/sstable"
	"bigtable/internal/telemetry"
	"bigtable/internal/wal"
)

type frozenSnapshot struct {
	id         uint64
	records    map[string]record.Record
	walArchive string
}

type DB struct {
	opts Options

	wal *wal.WAL

	mu          sync.RWMutex
	active      *memtable.Memtable
	nextSSTable uint64

	sstMu    sync.RWMutex
	sstables map[uint32][]*sstable.File

	flushMu sync.RWMutex
	frozen  []frozenSnapshot

	compactMu sync.Mutex
	wg        sync.WaitGroup

	closed bool

	events chan telemetry.Event
	stats  *telemetry.Stats

	compactionRunning bool
}

func (db *DB) Events() <-chan telemetry.Event {
	return db.events
}

func (db *DB) Snapshot() telemetry.Snapshot {
	if db.stats == nil {
		return telemetry.Snapshot{}
	}
	return db.stats.Snapshot()
}

func (db *DB) publish(ev telemetry.Event) {
	if db == nil || db.events == nil {
		return
	}
	select {
	case db.events <- ev:
	default:
	}
}

func (db *DB) refreshCountsLocked() {
	if db.stats == nil {
		return
	}

	// caller already owns active safely
	if db.active != nil {
		db.stats.TotalKeys.Store(int64(db.active.Size()))
	} else {
		db.stats.TotalKeys.Store(0)
	}

	// only protect sstable map
	db.sstMu.RLock()

	var totalSSTables int64

	for _, files := range db.sstables {
		totalSSTables += int64(len(files))
	}

	db.sstMu.RUnlock()

	db.stats.SSTables.Store(totalSSTables)
}

func Open(opts Options) (*DB, error) {
	if opts.DataDir == "" {
		return nil, fmt.Errorf("data dir cannot be empty")
	}

	if err := os.MkdirAll(opts.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	sstDir := filepath.Join(opts.DataDir, opts.SSTableDirName)
	if err := os.MkdirAll(sstDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sstable dir: %w", err)
	}

	walPath := filepath.Join(opts.DataDir, opts.WALFileName)
	w, err := wal.Open(walPath)
	if err != nil {
		return nil, err
	}

	db := &DB{
		opts:     opts,
		wal:      w,
		active:   memtable.New(),
		sstables: make(map[uint32][]*sstable.File),
		events:   make(chan telemetry.Event, 1024),
		stats:    &telemetry.Stats{},
	}

	recoveryStart := time.Now()
	if err := recovery.ReplayToMemtable(db.wal, db.active); err != nil {
		_ = db.wal.Close()
		return nil, fmt.Errorf("replay wal: %w", err)
	}
	db.stats.LastRecoveryMS.Store(time.Since(recoveryStart).Milliseconds())
	db.publish(telemetry.Event{
		Time:     time.Now(),
		Type:     telemetry.EventRecover,
		Message:  "WAL replay completed",
		Stage:    "recovery",
		Duration: time.Since(recoveryStart).Milliseconds(),
	})

	loaded, nextID, err := loadSSTableCache(sstDir)
	if err != nil {
		_ = db.wal.Close()
		return nil, err
	}
	db.sstables = loaded
	db.nextSSTable = nextID
	db.refreshCountsLocked()

	if err := db.cleanupArchivedWALs(); err != nil {
		_ = db.wal.Close()
		return nil, fmt.Errorf("cleanup archived wal files: %w", err)
	}

	return db, nil
}

func (db *DB) Put(key, value []byte) error {
	rec := record.NewPut(key, value, time.Now().UnixNano())

	db.mu.Lock()
	defer db.mu.Unlock()

	start := time.Now()

	if err := db.wal.Append(rec); err != nil {
		return err
	}

	db.active.Put(rec)

	db.stats.LastWriteMS.Store(time.Since(start).Milliseconds())
	db.refreshCountsLocked()
	db.publish(telemetry.Event{
		Time:     time.Now(),
		Type:     telemetry.EventPut,
		Message:  "write appended to WAL and staged in memtable",
		Key:      string(key),
		Value:    string(value),
		Stage:    "wal",
		Duration: time.Since(start).Milliseconds(),
	})

	if db.opts.MemtableThreshold > 0 && db.active.Size() >= db.opts.MemtableThreshold {
		if err := db.freezeActiveLocked(); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) Delete(key []byte) error {
	rec := record.NewDelete(key, time.Now().UnixNano())

	db.mu.Lock()
	defer db.mu.Unlock()

	start := time.Now()

	if err := db.wal.Append(rec); err != nil {
		return err
	}

	db.active.Delete(rec)

	db.stats.LastWriteMS.Store(time.Since(start).Milliseconds())
	db.refreshCountsLocked()
	db.publish(telemetry.Event{
		Time:     time.Now(),
		Type:     telemetry.EventDelete,
		Message:  "tombstone appended to WAL and staged in memtable",
		Key:      string(key),
		Stage:    "wal",
		Duration: time.Since(start).Milliseconds(),
	})

	if db.opts.MemtableThreshold > 0 && db.active.Size() >= db.opts.MemtableThreshold {
		if err := db.freezeActiveLocked(); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) freezeActiveLocked() error {
	snapshot := db.active.Snapshot()

	archivedWal, err := db.wal.Rotate()
	if err != nil {
		return fmt.Errorf("rotate wal: %w", err)
	}

	db.nextSSTable++
	id := db.nextSSTable
	path := db.sstablePath(0, id)

	db.flushMu.Lock()
	db.frozen = append(db.frozen, frozenSnapshot{
		id:         id,
		records:    snapshot,
		walArchive: archivedWal,
	})
	db.flushMu.Unlock()

	db.active = memtable.New()
	db.refreshCountsLocked()

	db.wg.Add(1)
	go db.flushSnapshotAsync(id, path, snapshot, archivedWal)

	return nil
}

func (db *DB) flushSnapshotAsync(id uint64, path string, snapshot map[string]record.Record, walArchive string) {
	defer db.wg.Done()

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		lastErr = sstable.WriteSnapshot(path, snapshot, db.opts.SparseIndexGap)
		if lastErr == nil {
			db.mu.RLock()
			closed := db.closed
			db.mu.RUnlock()

			if closed {
				return
			}

			db.markFlushComplete(id, path)

			if walArchive != "" {
				db.compactMu.Lock()

				db.advanceWALCheckpoint(walArchive)

				if err := os.Remove(walArchive); err != nil && !os.IsNotExist(err) {
					fmt.Fprintf(
						os.Stderr,
						"failed removing archived wal %s: %v\n",
						walArchive,
						err,
					)
				}

				db.compactMu.Unlock()
			}

			db.stats.Flushes.Add(1)
			db.publish(telemetry.Event{
				Time:    time.Now(),
				Type:    telemetry.EventFlush,
				Message: "memtable flushed to immutable SSTable",
				Stage:   "flush",
			})

			db.launchCompaction()
			return
		}
		time.Sleep(time.Duration(attempt) * 50 * time.Millisecond)
	}

	fmt.Fprintf(os.Stderr, "sstable flush failed for %d: %v\n", id, lastErr)
}

func (db *DB) markFlushComplete(id uint64, path string) {
	file, err := sstable.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to cache new sstable %s: %v\n", path, err)
		return
	}

	db.sstMu.Lock()
	level := file.Level()
	db.sstables[level] = append(db.sstables[level], file)
	sort.Slice(db.sstables[level], func(i, j int) bool {
		return db.sstables[level][i].Sequence() < db.sstables[level][j].Sequence()
	})
	db.sstMu.Unlock()

	db.flushMu.Lock()
	filtered := db.frozen[:0]
	for _, snap := range db.frozen {
		if snap.id != id {
			filtered = append(filtered, snap)
		}
	}
	db.frozen = filtered
	db.flushMu.Unlock()

	db.refreshCountsLocked()
}

func (db *DB) Get(key []byte) ([]byte, bool, error) {
	start := time.Now()

	db.mu.RLock()
	active := db.active
	db.mu.RUnlock()

	if rec, ok := active.Get(key); ok {
		if rec.IsDeleted {
			db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
			db.publish(telemetry.Event{
				Time:    time.Now(),
				Type:    telemetry.EventGet,
				Message: "tombstone encountered in memtable",
				Key:     string(key),
				Stage:   "memtable",
			})
			return nil, false, nil
		}

		value := make([]byte, len(rec.Value))
		copy(value, rec.Value)

		db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
		db.publish(telemetry.Event{
			Time:     time.Now(),
			Type:     telemetry.EventGet,
			Message:  "read served from memtable",
			Key:      string(key),
			Stage:    "memtable",
			Duration: time.Since(start).Milliseconds(),
		})
		return value, true, nil
	}

	db.flushMu.RLock()
	frozen := make([]frozenSnapshot, len(db.frozen))
	copy(frozen, db.frozen)
	db.flushMu.RUnlock()

	for i := len(frozen) - 1; i >= 0; i-- {
		rec, ok := lookupSnapshot(frozen[i].records, key)
		if ok {
			if rec.IsDeleted {
				db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
				db.publish(telemetry.Event{
					Time:    time.Now(),
					Type:    telemetry.EventGet,
					Message: "tombstone encountered in frozen snapshot",
					Key:     string(key),
					Stage:   "flush",
				})
				return nil, false, nil
			}

			value := make([]byte, len(rec.Value))
			copy(value, rec.Value)

			db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
			db.publish(telemetry.Event{
				Time:     time.Now(),
				Type:     telemetry.EventGet,
				Message:  "read served from frozen snapshot",
				Key:      string(key),
				Stage:    "flush",
				Duration: time.Since(start).Milliseconds(),
			})
			return value, true, nil
		}
	}

	db.sstMu.RLock()
	defer db.sstMu.RUnlock()

	levels := make([]uint32, 0, len(db.sstables))
	for level := range db.sstables {
		levels = append(levels, level)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })

	for _, level := range levels {
		files := db.sstables[level]

		if level == 0 {
			for i := len(files) - 1; i >= 0; i-- {
				rec, ok, err := files[i].Get(key)
				if err != nil {
					return nil, false, err
				}

				if ok {
					if rec.IsDeleted {
						db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
						db.publish(telemetry.Event{
							Time:    time.Now(),
							Type:    telemetry.EventGet,
							Message: "tombstone encountered in SSTable",
							Key:     string(key),
							Stage:   "sstable",
						})
						return nil, false, nil
					}

					value := make([]byte, len(rec.Value))
					copy(value, rec.Value)

					db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
					db.publish(telemetry.Event{
						Time:     time.Now(),
						Type:     telemetry.EventGet,
						Message:  "read served from SSTable",
						Key:      string(key),
						Stage:    "sstable",
						Duration: time.Since(start).Milliseconds(),
					})
					return value, true, nil
				}
			}
			continue
		}

		for i := 0; i < len(files); i++ {
			rec, ok, err := files[i].Get(key)
			if err != nil {
				return nil, false, err
			}

			if ok {
				if rec.IsDeleted {
					db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
					db.publish(telemetry.Event{
						Time:    time.Now(),
						Type:    telemetry.EventGet,
						Message: "tombstone encountered in SSTable",
						Key:     string(key),
						Stage:   "sstable",
					})
					return nil, false, nil
				}

				value := make([]byte, len(rec.Value))
				copy(value, rec.Value)

				db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
				db.publish(telemetry.Event{
					Time:     time.Now(),
					Type:     telemetry.EventGet,
					Message:  "read served from SSTable",
					Key:      string(key),
					Stage:    "sstable",
					Duration: time.Since(start).Milliseconds(),
				})
				return value, true, nil
			}
		}
	}

	db.stats.LastReadNS.Store(time.Since(start).Nanoseconds())
	db.publish(telemetry.Event{
		Time:    time.Now(),
		Type:    telemetry.EventGet,
		Message: "key not found",
		Key:     string(key),
		Stage:   "lookup",
	})

	return nil, false, nil
}

func (db *DB) Compact() error {
	db.compactMu.Lock()
	defer db.compactMu.Unlock()

	for {
		level, ok := db.lowestOverfullLevel()
		if !ok {
			return nil
		}
		if err := db.compactLevel(level); err != nil {
			return err
		}
	}
}

func (db *DB) lowestOverfullLevel() (uint32, bool) {
	fanout := db.compactionFanout()

	db.sstMu.RLock()
	defer db.sstMu.RUnlock()

	levels := make([]uint32, 0, len(db.sstables))
	for level := range db.sstables {
		levels = append(levels, level)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })

	for _, level := range levels {
		if len(db.sstables[level]) >= fanout {
			return level, true
		}
	}

	return 0, false
}

func (db *DB) compactLevel(level uint32) error {
	db.sstMu.RLock()
	files := append([]*sstable.File(nil), db.sstables[level]...)
	db.sstMu.RUnlock()

	if len(files) < db.compactionFanout() {
		return nil
	}

	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path())
	}

	db.mu.Lock()
	db.nextSSTable++
	newSeq := db.nextSSTable
	db.mu.Unlock()

	outPath := db.sstablePath(level+1, newSeq)

	if err := compaction.Merge(paths, outPath, db.opts.SparseIndexGap); err != nil {
		return err
	}

	merged, err := sstable.Open(outPath)
	if err != nil {
		return fmt.Errorf("open compacted sstable: %w", err)
	}

	db.sstMu.Lock()

	oldFiles := db.sstables[level]
	db.sstables[level] = make([]*sstable.File, 0)
	db.sstables[level+1] = append(db.sstables[level+1], merged)

	sort.Slice(db.sstables[level+1], func(i, j int) bool {
		return db.sstables[level+1][i].Sequence() <
			db.sstables[level+1][j].Sequence()
	})

	db.sstMu.Unlock()

	db.refreshCountsLocked()

	for i := range oldFiles {
		if oldFiles[i] != nil {
			_ = oldFiles[i].Close()
			_ = os.Remove(oldFiles[i].Path())
			oldFiles[i] = nil
		}
	}

	db.stats.Compactions.Add(1)
	db.publish(telemetry.Event{
		Time:    time.Now(),
		Type:    telemetry.EventCompact,
		Message: "leveled compaction completed",
		Stage:   "maintenance",
	})

	return nil
}

func (db *DB) compactionFanout() int {
	if db.opts.LevelCompactionFanout <= 1 {
		return 4
	}
	return db.opts.LevelCompactionFanout
}

func lookupSnapshot(snapshot map[string]record.Record, key []byte) (record.Record, bool) {
	rec, ok := snapshot[string(key)]
	if !ok {
		return record.Record{}, false
	}
	return rec, true
}

func (db *DB) Close() error {
	db.mu.Lock()

	if db.closed {
		db.mu.Unlock()
		return nil
	}

	db.closed = true

	db.mu.Unlock()

	db.wg.Wait()

	db.sstMu.Lock()
	for level := range db.sstables {
		for _, f := range db.sstables[level] {
			if f != nil {
				_ = f.Close()
			}
		}
		db.sstables[level] = nil
	}
	db.sstMu.Unlock()

	if db.wal != nil {
		return db.wal.Close()
	}

	return nil
}

func (db *DB) sstablePath(level uint32, seq uint64) string {
	name := fmt.Sprintf("l%d-sst-%020d.sst", level, seq)
	return filepath.Join(db.opts.DataDir, db.opts.SSTableDirName, name)
}

func loadSSTableCache(dir string) (map[uint32][]*sstable.File, uint64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, fmt.Errorf("read sstable dir: %w", err)
	}

	cache := make(map[uint32][]*sstable.File)
	var maxSeq uint64
	var opened []*sstable.File

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		path := filepath.Join(dir, e.Name())
		file, err := sstable.Open(path)
		if err != nil {
			for _, f := range opened {
				_ = f.Close()
			}
			return nil, 0, fmt.Errorf("open sstable %s: %w", path, err)
		}

		opened = append(opened, file)
		cache[file.Level()] = append(cache[file.Level()], file)
		if file.Sequence() > maxSeq {
			maxSeq = file.Sequence()
		}
	}

	for level := range cache {
		sort.Slice(cache[level], func(i, j int) bool {
			return cache[level][i].Sequence() < cache[level][j].Sequence()
		})
	}

	return cache, maxSeq, nil
}

func (db *DB) launchCompaction() {
	db.mu.Lock()

	if db.closed || db.compactionRunning {
		db.mu.Unlock()
		return
	}

	db.compactionRunning = true
	db.wg.Add(1)

	db.mu.Unlock()

	go func() {
		defer db.wg.Done()

		_ = db.Compact()

		db.mu.Lock()
		db.compactionRunning = false
		db.mu.Unlock()
	}()
}

func (db *DB) Barrier() error {
	db.wg.Wait()
	return nil
}