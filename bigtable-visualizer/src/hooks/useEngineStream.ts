import { useCallback, useEffect, useState } from "react";
import { fetchSnapshot, sendCommand, EngineEvent, EngineSnapshot } from "../lib/api";

export function useEngineStream() {
  const [snapshot, setSnapshot] = useState<EngineSnapshot | null>(null);
  const [events, setEvents] = useState<EngineEvent[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    fetchSnapshot()
      .then(setSnapshot)
      .catch(console.error);

    const es = new EventSource("/api/events");

    es.onopen = () => setConnected(true);

    es.onmessage = (ev) => {
      const data = JSON.parse(ev.data) as EngineEvent;
      setEvents((prev) => [...prev.slice(-19), data]);
    };

    es.onerror = () => {
      setConnected(false);
    };

    return () => es.close();
  }, []);

  const run = useCallback(
    async (op: "put" | "get" | "delete" | "compact", key?: string, value?: string) => {
      await sendCommand({ op, key, value });
      const next = await fetchSnapshot();
      setSnapshot(next);
    },
    []
  );

  return { snapshot, events, connected, run };
}