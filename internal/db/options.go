package db

import "time"

type Options struct {
	DataDir            string
	WALFileName        string
	SSTableDirName     string
	MemtableThreshold  int
	SparseIndexGap     int
	LevelCompactionFanout int
	CompactionInterval time.Duration
}

func DefaultOptions() Options {
	return Options{
		DataDir:               "./data",
		WALFileName:           "wal.log",
		SSTableDirName:        "sst",
		MemtableThreshold:     2000,
		SparseIndexGap:        10,
		LevelCompactionFanout: 4,
		CompactionInterval:    0,
	}
}