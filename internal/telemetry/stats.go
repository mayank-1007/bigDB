package telemetry

import "sync/atomic"

type Stats struct {
	TotalKeys      atomic.Int64
	WALSegments    atomic.Int64
	SSTables       atomic.Int64
	LastWriteMS    atomic.Int64
	LastReadNS     atomic.Int64
	LastRecoveryMS atomic.Int64
	Compactions    atomic.Int64
	Flushes        atomic.Int64
}

func (s *Stats) Snapshot() Snapshot {
	return Snapshot{
		TotalKeys:      s.TotalKeys.Load(),
		WALSegments:    s.WALSegments.Load(),
		SSTables:       s.SSTables.Load(),
		LastWriteMS:    s.LastWriteMS.Load(),
		LastReadNS:     s.LastReadNS.Load(),
		LastRecoveryMS: s.LastRecoveryMS.Load(),
		Compactions:    s.Compactions.Load(),
		Flushes:        s.Flushes.Load(),
	}
}