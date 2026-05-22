package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"bigtable/internal/record"
)

const checksumFlag uint32 = 1 << 31

type WAL struct {
	mu          sync.Mutex
	dir         string
	activePath  string
	file        *os.File
	nextArchive uint64
}

func Open(path string) (*WAL, error) {
	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create wal dir: %w", err)
	}

	nextSeq, err := scanNextArchiveSeq(dir, filepath.Base(path))
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open wal: %w", err)
	}

	return &WAL{
		dir:         dir,
		activePath:  path,
		file:        file,
		nextArchive: nextSeq,
	}, nil
}

func (w *WAL) Append(rec record.Record) error {
	payload, err := rec.Encode()
	if err != nil {
		return fmt.Errorf("encode record: %w", err)
	}

	entry := make([]byte, 4+len(payload)+4)
	binary.BigEndian.PutUint32(entry[0:4], uint32(len(payload))|checksumFlag)
	copy(entry[4:], payload)
	binary.BigEndian.PutUint32(entry[4+len(payload):], crc32.ChecksumIEEE(payload))

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return fmt.Errorf("wal is closed")
	}

	n, err := w.file.Write(entry)
	if err != nil {
		return fmt.Errorf("write wal entry: %w", err)
	}
	if n != len(entry) {
		return fmt.Errorf("short wal write: wrote %d of %d bytes", n, len(entry))
	}

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("sync wal: %w", err)
	}

	return nil
}

func (w *WAL) Rotate() (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return "", fmt.Errorf("wal is closed")
	}

	if err := w.file.Sync(); err != nil {
		return "", fmt.Errorf("sync wal before rotate: %w", err)
	}
	if err := w.file.Close(); err != nil {
		return "", fmt.Errorf("close wal before rotate: %w", err)
	}

	archivedPath := w.archivePath(w.nextArchive)
	w.nextArchive++

	if err := os.Rename(w.activePath, archivedPath); err != nil {
		reopen, reopenErr := os.OpenFile(w.activePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
		if reopenErr == nil {
			w.file = reopen
		}
		return "", fmt.Errorf("rename wal to archive: %w", err)
	}

	newFile, err := os.OpenFile(w.activePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("create new wal: %w", err)
	}

	w.file = newFile
	return archivedPath, nil
}

func (w *WAL) Replay(fn func(record.Record) error) error {
	return replayFile(w.activePath, fn)
}

func (w *WAL) ReplayAll(fn func(record.Record) error) error {
	paths, err := w.segmentPaths()
	if err != nil {
		return err
	}

	for _, path := range paths {
		if err := replayFile(path, fn); err != nil {
			return err
		}
	}
	return nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	err := w.file.Close()
	w.file = nil
	return err
}

func (w *WAL) archivePath(seq uint64) string {
	base := filepath.Base(w.activePath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	name := fmt.Sprintf("%s-%06d%s", stem, seq, ext)
	return filepath.Join(w.dir, name)
}

func (w *WAL) segmentPaths() ([]string, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, fmt.Errorf("read wal dir: %w", err)
	}

	type seg struct {
		seq  uint64
		path string
	}

	activeBase := filepath.Base(w.activePath)
	segments := make([]seg, 0)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if name == activeBase {
			continue
		}

		seq, ok := parseArchiveName(name, activeBase)
		if !ok {
			continue
		}

		segments = append(segments, seg{
			seq:  seq,
			path: filepath.Join(w.dir, name),
		})
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].seq < segments[j].seq
	})

	paths := make([]string, 0, len(segments)+1)
	for _, s := range segments {
		paths = append(paths, s.path)
	}

	if _, err := os.Stat(w.activePath); err == nil {
		paths = append(paths, w.activePath)
	}

	return paths, nil
}

func scanNextArchiveSeq(dir, activeBase string) (uint64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1, nil
	}

	var maxSeq uint64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		seq, ok := parseArchiveName(e.Name(), activeBase)
		if !ok {
			continue
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}

	return maxSeq + 1, nil
}

func parseArchiveName(name, activeBase string) (uint64, bool) {
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

func replayFile(path string, fn func(record.Record) error) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open wal for replay: %w", err)
	}
	defer f.Close()

	for {
		var lenBuf [4]byte
		_, err := io.ReadFull(f, lenBuf[:])
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return fmt.Errorf("read wal length: %w", err)
		}

		rawLen := binary.BigEndian.Uint32(lenBuf[:])
		hasCRC := rawLen&checksumFlag != 0
		payloadLen := rawLen &^ checksumFlag

		if payloadLen == 0 {
			return nil
		}

		payload := make([]byte, payloadLen)
		_, err = io.ReadFull(f, payload)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return fmt.Errorf("read wal payload: %w", err)
		}

		if hasCRC {
			var crcBuf [4]byte
			_, err = io.ReadFull(f, crcBuf[:])
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return nil
				}
				return fmt.Errorf("read wal crc: %w", err)
			}

			want := binary.BigEndian.Uint32(crcBuf[:])
			got := crc32.ChecksumIEEE(payload)
			if want != got {
				return fmt.Errorf("wal crc mismatch")
			}
		}

		rec, err := record.Decode(payload)
		if err != nil {
			return fmt.Errorf("decode wal record: %w", err)
		}

		if err := fn(rec); err != nil {
			return err
		}
	}
}