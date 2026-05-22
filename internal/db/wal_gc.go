package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const walCheckpointFileName = "wal.checkpoint"

// walCheckpoint stores the highest archived WAL sequence that is safe to delete.
func (db *DB) walCheckpointPath() string {
	return filepath.Join(db.opts.DataDir, walCheckpointFileName)
}

func (db *DB) loadWALCheckpoint() (uint64, error) {
	data, err := os.ReadFile(db.walCheckpointPath())
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read wal checkpoint: %w", err)
	}

	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return 0, nil
	}

	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse wal checkpoint: %w", err)
	}
	return v, nil
}

func (db *DB) saveWALCheckpoint(seq uint64) error {
	tmp := db.walCheckpointPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.FormatUint(seq, 10)), 0o644); err != nil {
		return fmt.Errorf("write wal checkpoint tmp: %w", err)
	}
	if err := os.Rename(tmp, db.walCheckpointPath()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename wal checkpoint: %w", err)
	}
	return nil
}

func (db *DB) advanceWALCheckpoint(archivedPath string) {
	seq, ok := parseWALArchiveSeq(filepath.Base(archivedPath), filepath.Base(db.opts.WALFileName))
	if !ok {
		return
	}

	current, err := db.loadWALCheckpoint()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load wal checkpoint: %v\n", err)
		return
	}
	if seq <= current {
		return
	}

	if err := db.saveWALCheckpoint(seq); err != nil {
		fmt.Fprintf(os.Stderr, "save wal checkpoint: %v\n", err)
	}
}

func (db *DB) cleanupArchivedWALs() error {
	checkpoint, err := db.loadWALCheckpoint()
	if err != nil {
		return err
	}
	if checkpoint == 0 {
		return nil
	}

	entries, err := os.ReadDir(db.opts.DataDir)
	if err != nil {
		return fmt.Errorf("read data dir: %w", err)
	}

	activeBase := filepath.Base(db.opts.WALFileName)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		seq, ok := parseWALArchiveSeq(name, activeBase)
		if !ok {
			continue
		}
		if seq <= checkpoint {
			_ = os.Remove(filepath.Join(db.opts.DataDir, name))
		}
	}

	return nil
}

func parseWALArchiveSeq(name, activeBase string) (uint64, bool) {
	ext := filepath.Ext(activeBase)
	stem := strings.TrimSuffix(activeBase, ext)
	prefix := stem + "-"

	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ext) {
		return 0, false
	}

	raw := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ext)
	seq, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return seq, true
}

