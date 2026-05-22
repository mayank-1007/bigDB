import type { EngineEvent, EngineSnapshot } from "./api";

const stepMap: Record<string, number> = {
  put: 1,
  delete: 1,
  get: 2,
  flush: 3,
  compact: 4,
  recover: 5,
};

export function buildLiveModel(snapshot: EngineSnapshot | null, events: EngineEvent[]) {
  const latest = events[events.length - 1];
  const activeStep = latest ? stepMap[latest.type] ?? 0 : 0;

  const metrics = snapshot
    ? [
        { label: "THROUGHPUT", value: "LIVE", suffix: "", tone: "text-cyan-300" },
        { label: "P99 LATENCY", value: `${snapshot.last_write_ms || 0}`, suffix: "ms", tone: "text-violet-300" },
        { label: "UPTIME", value: `${snapshot.total_keys}`, suffix: "keys", tone: "text-emerald-300" },
        { label: "STATUS", value: "ACTIVE", suffix: "", tone: "text-emerald-300" },
      ]
    : [];

  const consoleRows = events.slice(-4).reverse();

  const packets = [
    { label: "PUT", tone: "text-cyan-200" },
    { label: "WAL", tone: "text-violet-200" },
    { label: "MEM", tone: "text-emerald-200" },
    { label: "SST", tone: "text-amber-200" },
    { label: "CMP", tone: "text-fuchsia-200" },
  ];

  return {
    latest,
    activeStep,
    metrics,
    consoleRows,
    packets,
  };
}