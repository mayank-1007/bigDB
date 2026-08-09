#include "telemetry.h"

namespace bigdb::telemetry {

::bigdb::telemetry::Snapshot Stats::Snapshot() const {
    ::bigdb::telemetry::Snapshot snap;
    snap.totalKeys = totalKeys.load();
    snap.walSegments = walSegments.load();
    snap.sstables = sstables.load();
    snap.lastWriteMS = lastWriteMS.load();
    snap.lastReadNS = lastReadNS.load();
    snap.lastRecoveryMS = lastRecoveryMS.load();
    snap.compactions = compactions.load();
    snap.flushes = flushes.load();
    return snap;
}

}  // namespace bigdb::telemetry
