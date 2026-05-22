export type EngineSnapshot = {
  total_keys: number;
  wal_segments: number;
  sstables: number;
  last_write_ms: number;
  last_read_ns: number;
  last_recovery_ms: number;
  compactions: number;
  flushes: number;
};

export type EngineEvent = {
  time: string;
  type: string;
  message: string;
  key?: string;
  value?: string;
  stage: string;
  duration_ms?: number;
};

export async function fetchSnapshot(): Promise<EngineSnapshot> {
  const res = await fetch("/api/state");
  if (!res.ok) throw new Error("failed to fetch snapshot");
  return res.json();
}

export async function sendCommand(payload: {
  op: "put" | "get" | "delete" | "compact";
  key?: string;
  value?: string;
}): Promise<void> {
  const res = await fetch("/api/command", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    throw new Error(await res.text());
  }
}