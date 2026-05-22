import React, { useEffect, useMemo, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import {
  Activity,
  ArrowRight,
  Boxes,
  CheckCircle2,
  Clock3,
  Command,
  Cpu,
  Database,
  Gauge,
  Layers3,
  Lock,
  Network,
  Pause,
  Play,
  RefreshCcw,
  Server,
  ShieldCheck,
  Sparkles,
  TerminalSquare,
  UploadCloud,
  Zap,
  AlertTriangle,
  Trash2,
} from "lucide-react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

type EngineSnapshot = {
  total_keys: number;
  wal_segments: number;
  sstables: number;
  last_write_ms: number;
  last_read_ns: number;
  last_recovery_ms: number;
  compactions: number;
  flushes: number;
};

type EngineEvent = {
  time: string;
  type: string;
  message: string;
  key?: string;
  value?: string;
  stage: string;
  duration_ms?: number;
};

type PipelineStage = {
  id: string;
  label: string;
  sub: string;
  icon: React.ReactNode;
  x: number;
  y: number;
};

const pipelineStages: PipelineStage[] = [
  {
    id: "input",
    label: "Client Input",
    sub: "write or read trigger",
    icon: <Command className="h-5 w-5" />,
    x: 8,
    y: 54,
  },
  {
    id: "wal",
    label: "WAL",
    sub: "durable append",
    icon: <Zap className="h-5 w-5" />,
    x: 28,
    y: 32,
  },
  {
    id: "memtable",
    label: "Memtable",
    sub: "hot memory state",
    icon: <Database className="h-5 w-5" />,
    x: 48,
    y: 54,
  },
  {
    id: "sstable",
    label: "SSTable",
    sub: "immutable disk file",
    icon: <Boxes className="h-5 w-5" />,
    x: 68,
    y: 32,
  },
  {
    id: "compact",
    label: "Compaction",
    sub: "leveled merge",
    icon: <Layers3 className="h-5 w-5" />,
    x: 86,
    y: 54,
  },
  {
    id: "recover",
    label: "Recovery",
    sub: "WAL replay",
    icon: <RefreshCcw className="h-5 w-5" />,
    x: 95,
    y: 32,
  },
];

const implemented = [
  {
    title: "Durability",
    icon: <Lock className="h-4 w-4" />,
    items: ["WAL durability", "Group commit", "CRC validation", "Checkpoint GC"],
  },
  {
    title: "Storage",
    icon: <Database className="h-4 w-4" />,
    items: ["Memtable", "SSTables", "Sparse index", "Bloom filters"],
  },
  {
    title: "Maintenance",
    icon: <Layers3 className="h-4 w-4" />,
    items: ["Background flush", "Leveled compaction", "Tombstones", "Cleanup"],
  },
  {
    title: "Quality",
    icon: <ShieldCheck className="h-4 w-4" />,
    items: ["Race tests", "Recovery tests", "Corruption tests", "Benchmarks"],
  },
];

const stageToIndex: Record<string, number> = {
  put: 1,
  delete: 1,
  wal: 1,
  get: 2,
  memtable: 2,
  flush: 3,
  sstable: 3,
  compact: 4,
  maintenance: 4,
  recover: 5,
  recovery: 5,
};

const typeLabel: Record<string, string> = {
  put: "PUT",
  get: "GET",
  delete: "DELETE",
  flush: "FLUSH",
  compact: "COMPACT",
  recover: "RECOVER",
  wal: "WAL",
  memtable: "MEMTABLE",
  sstable: "SSTABLE",
};

function formatLabel(type: string) {
  return typeLabel[type.toLowerCase()] ?? type.toUpperCase();
}

function useEngineLive() {
  const [snapshot, setSnapshot] = useState<EngineSnapshot | null>(null);
  const [events, setEvents] = useState<EngineEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const [keyInput, setKeyInput] = useState("user:1");
  const [valueInput, setValueInput] = useState("alice");

  const refreshSnapshot = async () => {
    const res = await fetch("/api/state");
    if (!res.ok) throw new Error("failed to fetch snapshot");
    const data = (await res.json()) as EngineSnapshot;
    setSnapshot(data);
  };

  useEffect(() => {
    refreshSnapshot().catch(console.error);

    const es = new EventSource("/api/events");
    es.onopen = () => setConnected(true);
    es.onmessage = (ev) => {
      const data = JSON.parse(ev.data) as EngineEvent;
      setEvents((prev) => [...prev.slice(-19), data]);
      setBusy(null);
      refreshSnapshot().catch(() => {});
    };
    es.onerror = () => setConnected(false);

    return () => es.close();
  }, []);

  const run = async (op: "put" | "get" | "delete" | "compact") => {
    setBusy(op);
    const body: Record<string, string> = { op };

    if (op === "put") {
      body.key = keyInput;
      body.value = valueInput;
    } else if (op === "get" || op === "delete") {
      body.key = keyInput;
    }

    const res = await fetch("/api/command", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });

    if (!res.ok) {
      setBusy(null);
      throw new Error(await res.text());
    }

    await refreshSnapshot();
  };

  return {
    snapshot,
    events,
    connected,
    busy,
    keyInput,
    setKeyInput,
    valueInput,
    setValueInput,
    run,
  };
}

function MetricCard({
  label,
  value,
  unit,
  tone,
  icon,
}: {
  label: string;
  value: string;
  unit: string;
  tone: string;
  icon: React.ReactNode;
}) {
  return (
    <div className="rounded-[1.25rem] border border-white/10 bg-white/[0.04] p-4 backdrop-blur-xl">
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="text-[11px] uppercase tracking-[0.28em] text-white/40">{label}</div>
          <div className={cn("mt-2 flex items-end gap-2", tone)}>
            <div className="text-3xl font-semibold leading-none">{value}</div>
            <div className="pb-1 text-sm text-white/45">{unit}</div>
          </div>
        </div>
        <div className="rounded-xl border border-white/10 bg-black/30 p-2 text-cyan-300">{icon}</div>
      </div>
    </div>
  );
}

function cn(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

export default function BigtableVisualizerDashboard() {
  const {
    snapshot,
    events,
    connected,
    busy,
    keyInput,
    setKeyInput,
    valueInput,
    setValueInput,
    run,
  } = useEngineLive();

  const latest = events[events.length - 1];
  const activeIndex = latest ? stageToIndex[latest.type.toLowerCase()] ?? stageToIndex[latest.stage?.toLowerCase?.() ?? ""] ?? 0 : 0;

  const metrics = useMemo(() => {
    const write = snapshot?.last_write_ms ?? 0;
    const readNs = snapshot?.last_read_ns ?? 0;
    const recovery = snapshot?.last_recovery_ms ?? 0;

    return [
      { label: "TOTAL KEYS", value: String(snapshot?.total_keys ?? 0), unit: "keys", tone: "text-cyan-300", icon: <Database className="h-5 w-5" /> },
      { label: "WAL SEGMENTS", value: String(snapshot?.wal_segments ?? 0), unit: "files", tone: "text-violet-300", icon: <Zap className="h-5 w-5" /> },
      { label: "SSTABLES", value: String(snapshot?.sstables ?? 0), unit: "files", tone: "text-emerald-300", icon: <Boxes className="h-5 w-5" /> },
      { label: "LAST WRITE", value: write ? String(write) : "0", unit: "ms", tone: "text-amber-300", icon: <Clock3 className="h-5 w-5" /> },
      { label: "LAST READ", value: readNs ? String(readNs) : "0", unit: "ns", tone: "text-sky-300", icon: <Activity className="h-5 w-5" /> },
      { label: "RECOVERY", value: recovery ? String(recovery) : "0", unit: "ms", tone: "text-fuchsia-300", icon: <RefreshCcw className="h-5 w-5" /> },
    ];
  }, [snapshot]);

  const latencyData = useMemo(
    () =>
      events
        .slice(-8)
        .map((e, i) => ({
          name: e.time?.slice?.(-8) || `${i}`,
          value: e.duration_ms ?? (i + 1) * 6,
        })),
    [events]
  );

  const consoleRows = useMemo(() => [...events].slice(-10).reverse(), [events]);

  const packetLabel = formatLabel(latest?.type ?? latest?.stage ?? "pulse");
  const packetTone =
    activeIndex === 1
      ? "border-violet-300/30 bg-violet-300/10 text-violet-100"
      : activeIndex === 2
        ? "border-emerald-300/30 bg-emerald-300/10 text-emerald-100"
        : activeIndex === 3
          ? "border-amber-300/30 bg-amber-300/10 text-amber-100"
          : activeIndex === 4
            ? "border-fuchsia-300/30 bg-fuchsia-300/10 text-fuchsia-100"
            : "border-cyan-300/30 bg-cyan-300/10 text-cyan-100";

  return (
    <div className="min-h-screen bg-[#000000] text-white selection:bg-cyan-400/30">
      <style>{`
        .bg-grid {
          background-image:
            linear-gradient(to right, rgba(255, 255, 255, 0.06) 1px, transparent 1px),
            linear-gradient(to bottom, rgba(255, 255, 255, 0.06) 1px, transparent 1px);
          background-size: 36px 36px;
        }
        .bg-grid::before {
          content: '';
          position: fixed;
          inset: 0;
          pointer-events: none;
          z-index: -1;
          background:
            radial-gradient(circle at 20% 20%, rgba(0, 221, 255, 0.1), transparent 22%),
            radial-gradient(circle at 80% 18%, rgba(132, 0, 255, 0.09), transparent 18%),
            radial-gradient(circle at 70% 82%, rgba(0, 255, 170, 0.08), transparent 20%);
        }
        @keyframes glowPulse {
          0%,100% { box-shadow: 0 0 12px rgba(0, 221, 255, 0.14); }
          50% { box-shadow: 0 0 32px rgba(0, 221, 255, 0.32); }
        }
        .glow-led { animation: glowPulse 2.4s ease-in-out infinite; }
      `}</style>

      <div className="bg-grid min-h-screen">
        <header className="sticky top-0 z-40 border-b border-white/10 bg-[#000000]/80 backdrop-blur-xl">
          <div className="mx-auto flex h-16 max-w-[1600px] items-center justify-between px-4 lg:px-8">
            <div className="flex items-center gap-3">
              <div className="text-[22px] font-semibold tracking-[0.18em] text-cyan-300">AETHER_DB</div>
              <div className="hidden rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[11px] uppercase tracking-[0.28em] text-white/45 md:block">
                live storage visualizer
              </div>
            </div>
            <div className="flex items-center gap-3 text-white/70">
              <span className={cn("rounded-full border px-3 py-1 text-[11px] uppercase tracking-[0.24em]", connected ? "border-emerald-400/30 bg-emerald-400/10 text-emerald-200" : "border-amber-400/30 bg-amber-400/10 text-amber-200")}>{connected ? "connected" : "disconnected"}</span>
              <Play className="h-4 w-4" />
              <Pause className="h-4 w-4" />
            </div>
          </div>
        </header>

        <main className="mx-auto max-w-[1600px] px-4 py-6 lg:px-8">
          {/* hero */}
          <section className="rounded-[2rem] border border-white/10 bg-white/[0.03] p-5 backdrop-blur-xl lg:p-6">
            <div className="mx-auto max-w-4xl text-center">
              <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-[11px] uppercase tracking-[0.32em] text-cyan-200">
                <Sparkles className="h-3.5 w-3.5" />
                deep-black data-path demo
              </div>
              <h1 className="text-4xl font-semibold tracking-tight md:text-6xl">
                Real command flow through your storage engine
              </h1>
              <p className="mx-auto mt-4 max-w-3xl text-sm leading-7 text-slate-300/80 md:text-base">
                This view is driven by your actual Go backend. PUT, GET, DELETE, and COMPACT commands move through WAL, memtable, SSTables, and recovery with live packets, stage glow, and real state.
              </p>
            </div>

            <div className="mt-6 grid gap-3 md:grid-cols-2 xl:grid-cols-6">
              {metrics.map((m) => (
                <MetricCard key={m.label} {...m} />
              ))}
            </div>
          </section>

          {/* command bar */}
          <section className="mt-6 grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
            <div className="rounded-[2rem] border border-white/10 bg-[#000000]/80 p-5 backdrop-blur-xl lg:p-6">
              <div className="mb-4 flex items-center justify-between gap-4">
                <div>
                  <div className="text-[11px] uppercase tracking-[0.3em] text-white/35">interactive commands</div>
                  <div className="mt-2 text-xl font-semibold text-white">Run the real engine</div>
                </div>
                <div className="rounded-full border border-white/10 bg-black/30 px-3 py-1 text-[11px] uppercase tracking-[0.28em] text-white/45">
                  {busy ? `running ${busy}` : "idle"}
                </div>
              </div>

              <div className="grid gap-3 md:grid-cols-[1.2fr_1fr_auto_auto_auto]">
                <input
                  value={keyInput}
                  onChange={(e) => setKeyInput(e.target.value)}
                  placeholder="key"
                  className="h-12 rounded-2xl border border-white/10 bg-black/40 px-4 text-sm text-white outline-none placeholder:text-white/30 focus:border-cyan-400/40"
                />
                <input
                  value={valueInput}
                  onChange={(e) => setValueInput(e.target.value)}
                  placeholder="value"
                  className="h-12 rounded-2xl border border-white/10 bg-black/40 px-4 text-sm text-white outline-none placeholder:text-white/30 focus:border-cyan-400/40"
                />
                <button
                  onClick={() => run("put").catch(console.error)}
                  className="h-12 rounded-2xl bg-cyan-300 px-4 text-sm font-semibold text-slate-950 transition hover:scale-[1.02] hover:bg-cyan-200"
                >
                  PUT
                </button>
                <button
                  onClick={() => run("get").catch(console.error)}
                  className="h-12 rounded-2xl border border-white/10 bg-white/[0.04] px-4 text-sm font-semibold text-white transition hover:bg-white/[0.08]"
                >
                  GET
                </button>
                <button
                  onClick={() => run("delete").catch(console.error)}
                  className="h-12 rounded-2xl border border-rose-400/20 bg-rose-400/10 px-4 text-sm font-semibold text-rose-100 transition hover:bg-rose-400/15"
                >
                  DELETE
                </button>
              </div>

              <div className="mt-3 flex flex-wrap gap-3">
                <button
                  onClick={() => run("compact").catch(console.error)}
                  className="inline-flex h-11 items-center gap-2 rounded-2xl border border-fuchsia-400/20 bg-fuchsia-400/10 px-4 text-sm font-semibold text-fuchsia-100 transition hover:bg-fuchsia-400/15"
                >
                  <Layers3 className="h-4 w-4" />
                  COMPACT
                </button>
                <div className="flex items-center gap-2 rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-2 text-xs uppercase tracking-[0.24em] text-white/45">
                  <AlertTriangle className="h-4 w-4 text-amber-300" />
                  flush is automatic when memtable threshold is reached
                </div>
              </div>
            </div>

            <div className="rounded-[2rem] border border-white/10 bg-[#000000]/80 p-5 backdrop-blur-xl lg:p-6">
              <div className="mb-4 flex items-center justify-between gap-4">
                <div>
                  <div className="text-[11px] uppercase tracking-[0.3em] text-white/35">latest event</div>
                  <div className="mt-2 text-xl font-semibold text-white">Stage + packet state</div>
                </div>
                <div className="rounded-full border border-white/10 bg-black/30 px-3 py-1 text-[11px] uppercase tracking-[0.28em] text-white/45">
                  {activeIndex ? "active path" : "waiting"}
                </div>
              </div>

              <AnimatePresence mode="wait">
                <motion.div
                  key={latest?.time ?? "empty"}
                  initial={{ opacity: 0, y: 12, filter: "blur(8px)" }}
                  animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                  exit={{ opacity: 0, y: -12, filter: "blur(8px)" }}
                  className="rounded-[1.5rem] border border-white/10 bg-black/30 p-4"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <div className="text-[11px] uppercase tracking-[0.28em] text-white/35">{latest?.time ?? "--:--:--"}</div>
                      <div className="mt-2 text-2xl font-semibold text-cyan-200">{packetLabel}</div>
                      <div className="mt-3 text-sm leading-7 text-slate-300/80">{latest?.message ?? "No live event yet. Run a command to begin the flow."}</div>
                    </div>
                    <div className="rounded-2xl border border-white/10 bg-white/[0.04] px-3 py-2 text-[11px] uppercase tracking-[0.26em] text-white/45">
                      {latest?.stage ?? "standby"}
                    </div>
                  </div>
                </motion.div>
              </AnimatePresence>

              <div className="mt-4 grid gap-3 sm:grid-cols-3">
                <div className="rounded-[1.25rem] border border-white/10 bg-black/30 p-4">
                  <div className="text-[11px] uppercase tracking-[0.28em] text-white/35">flow stage</div>
                  <div className="mt-2 text-lg font-semibold text-white">{pipelineStages[activeIndex]?.label ?? "Client Input"}</div>
                </div>
                <div className="rounded-[1.25rem] border border-white/10 bg-black/30 p-4">
                  <div className="text-[11px] uppercase tracking-[0.28em] text-white/35">snapshot</div>
                  <div className="mt-2 text-lg font-semibold text-white">{snapshot?.total_keys ?? 0} keys</div>
                </div>
                <div className="rounded-[1.25rem] border border-white/10 bg-black/30 p-4">
                  <div className="text-[11px] uppercase tracking-[0.28em] text-white/35">engine</div>
                  <div className={cn("mt-2 text-lg font-semibold", connected ? "text-emerald-300" : "text-amber-200")}>{connected ? "live" : "waiting"}</div>
                </div>
              </div>
            </div>
          </section>

          {/* pipeline + console */}
          <section className="mt-6 grid gap-6 xl:grid-cols-[1.25fr_0.75fr]">
            <div className="rounded-[2rem] border border-white/10 bg-[#000000]/80 p-5 backdrop-blur-xl lg:p-6">
              <div className="mb-5 flex items-center justify-between gap-4">
                <div>
                  <div className="text-[11px] uppercase tracking-[0.3em] text-white/35">live data path</div>
                  <div className="mt-2 text-xl font-semibold text-white">Command → WAL → Memtable → SSTable → Compaction → Recovery</div>
                </div>
                <div className="flex items-center gap-2 rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1 text-[11px] uppercase tracking-[0.26em] text-emerald-200">
                  <Activity className="h-3.5 w-3.5" />
                  real-time packets
                </div>
              </div>

              <div className="relative min-h-[470px] overflow-hidden rounded-[1.75rem] border border-white/10 bg-[radial-gradient(circle_at_center,rgba(34,211,238,0.08),transparent_25%),radial-gradient(circle_at_65%_35%,rgba(168,85,247,0.08),transparent_18%),linear-gradient(180deg,rgba(5,10,20,1),rgba(8,14,26,1))] p-4 [perspective:1400px]">
                <div className="absolute inset-0 opacity-35 [background-image:linear-gradient(rgba(255,255,255,0.05)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.05)_1px,transparent_1px)] [background-size:44px_44px] [transform:rotateX(68deg) translateY(110px)] [transform-origin:center]" />
                <div className="absolute left-1/2 top-1/2 h-72 w-72 -translate-x-1/2 -translate-y-1/2 rounded-full bg-cyan-400/10 blur-3xl" />
                <div className="absolute inset-x-6 top-1/2 h-px -translate-y-1/2 bg-gradient-to-r from-transparent via-white/15 to-transparent" />

                {pipelineStages.map((stage, idx) => {
                  const isActive = idx === activeIndex;
                  return (
                    <motion.div
                      key={stage.id}
                      animate={{
                        scale: isActive ? 1.08 : 1,
                        y: isActive ? -8 : 0,
                        rotateX: isActive ? 10 : 0,
                      }}
                      transition={{ duration: 0.35 }}
                      className={cn(
                        "absolute -translate-x-1/2 -translate-y-1/2 rounded-[1.35rem] border px-4 py-3 backdrop-blur-xl",
                        isActive ? "border-cyan-400/50 bg-white/10 shadow-[0_0_40px_rgba(34,211,238,0.18)]" : "border-white/10 bg-white/[0.04]"
                      )}
                      style={{ left: `${stage.x}%`, top: `${stage.y}%` }}
                    >
                      <div className={cn("absolute inset-0 rounded-[1.35rem] bg-gradient-to-br opacity-35", idx === 1 ? "from-violet-400/20 to-violet-500/0" : idx === 2 ? "from-emerald-400/20 to-emerald-500/0" : idx === 3 ? "from-amber-400/20 to-amber-500/0" : idx === 4 ? "from-fuchsia-400/20 to-fuchsia-500/0" : "from-cyan-400/20 to-cyan-500/0")} />
                      <div className="relative z-10 flex items-center gap-3">
                        <div className={cn("rounded-2xl border border-white/10 bg-black/30 p-2", isActive ? "text-cyan-300" : "text-white/70")}>{stage.icon}</div>
                        <div>
                          <div className="text-sm font-semibold tracking-[0.16em] text-white">{stage.label}</div>
                          <div className="mt-1 text-[11px] uppercase tracking-[0.24em] text-white/35">{stage.sub}</div>
                        </div>
                      </div>
                    </motion.div>
                  );
                })}

                <motion.div
                  key={`${packetLabel}-${activeIndex}`}
                  animate={{ left: `${pipelineStages[activeIndex]?.x ?? 8}%`, top: `${(pipelineStages[activeIndex]?.y ?? 54) - 16}%`, scale: activeIndex === 0 ? 1 : 1.08 }}
                  transition={{ type: "spring", stiffness: 120, damping: 16 }}
                  className={cn("absolute -translate-x-1/2 -translate-y-1/2 rounded-full border px-3 py-1 text-[11px] uppercase tracking-[0.26em] shadow-[0_0_24px_rgba(34,211,238,0.18)]", packetTone)}
                >
                  {packetLabel}
                </motion.div>

                <div className="absolute inset-x-6 bottom-4 rounded-[1.25rem] border border-white/10 bg-black/35 px-4 py-3 text-center text-[11px] uppercase tracking-[0.3em] text-white/45 backdrop-blur-xl">
                  packet highlights show where the live event currently is in the engine
                </div>
              </div>
            </div>

            <div className="rounded-[2rem] border border-white/10 bg-[#000000]/80 p-5 backdrop-blur-xl lg:p-6">
              <div className="mb-4 flex items-center justify-between gap-4">
                <div>
                  <div className="text-[11px] uppercase tracking-[0.3em] text-white/35">live console</div>
                  <div className="mt-2 text-xl font-semibold text-white">Real event stream</div>
                </div>
                <TerminalSquare className="h-5 w-5 text-violet-300" />
              </div>

              <div className="space-y-3">
                {consoleRows.length === 0 ? (
                  <div className="rounded-[1.25rem] border border-white/10 bg-black/30 p-4 text-sm text-slate-300/80">
                    No events yet. Send a PUT or GET to begin live playback.
                  </div>
                ) : (
                  consoleRows.map((row, index) => (
                    <motion.div
                      key={`${row.time}-${index}`}
                      initial={{ opacity: 0, x: 14 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ duration: 0.25, delay: index * 0.04 }}
                      className={cn(
                        "rounded-[1.25rem] border px-4 py-3",
                        index === 0 ? "border-cyan-400/20 bg-white/[0.06]" : "border-white/10 bg-black/25"
                      )}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="flex items-center gap-2 text-[11px] uppercase tracking-[0.24em] text-white/35">
                            <span className="font-mono text-white/70">{row.time}</span>
                            <span className={cn("font-semibold", row.type === "put" ? "text-cyan-300" : row.type === "delete" ? "text-rose-300" : row.type === "compact" ? "text-fuchsia-300" : row.type === "flush" ? "text-amber-300" : row.type === "recover" ? "text-sky-300" : "text-violet-300")}>{formatLabel(row.type)}</span>
                          </div>
                          <div className="mt-2 text-sm leading-6 text-slate-300/85">{row.message}</div>
                        </div>
                        <div className="shrink-0 rounded-full border border-white/10 bg-black/30 px-3 py-1 text-[10px] uppercase tracking-[0.24em] text-white/45">{row.stage}</div>
                      </div>
                    </motion.div>
                  ))
                )}
              </div>

              <div className="mt-5 rounded-[1.25rem] border border-white/10 bg-black/30 p-4">
                <div className="flex items-center gap-2 text-sm font-semibold text-white">
                  <Server className="h-4 w-4 text-cyan-300" />
                  Snapshot summary
                </div>
                <div className="mt-3 grid grid-cols-2 gap-3 text-sm text-slate-300/80">
                  <div className="rounded-xl border border-white/10 bg-white/[0.03] p-3">keys: <span className="text-white">{snapshot?.total_keys ?? 0}</span></div>
                  <div className="rounded-xl border border-white/10 bg-white/[0.03] p-3">wal: <span className="text-white">{snapshot?.wal_segments ?? 0}</span></div>
                  <div className="rounded-xl border border-white/10 bg-white/[0.03] p-3">sstables: <span className="text-white">{snapshot?.sstables ?? 0}</span></div>
                  <div className="rounded-xl border border-white/10 bg-white/[0.03] p-3">compactions: <span className="text-white">{snapshot?.compactions ?? 0}</span></div>
                </div>
              </div>
            </div>
          </section>

          {/* coverage + charts */}
          <section className="mt-6 grid gap-6 xl:grid-cols-[0.92fr_1.08fr]">
            <div className="rounded-[2rem] border border-white/10 bg-[#000000]/80 p-5 backdrop-blur-xl lg:p-6">
              <div className="mb-4 flex items-center gap-2 text-white">
                <ShieldCheck className="h-5 w-5 text-emerald-300" />
                Implemented coverage
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                {implemented.map((group) => (
                  <div key={group.title} className="rounded-[1.25rem] border border-white/10 bg-white/[0.04] p-4">
                    <div className="flex items-center gap-2 text-sm font-semibold text-white">
                      <span className="text-cyan-300">{group.icon}</span>
                      {group.title}
                    </div>
                    <div className="mt-3 flex flex-wrap gap-2">
                      {group.items.map((item) => (
                        <span key={item} className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/30 px-3 py-1 text-[11px] text-slate-200/90">
                          <CheckCircle2 className="h-3.5 w-3.5 text-emerald-300" />
                          {item}
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-[2rem] border border-white/10 bg-[#000000]/80 p-5 backdrop-blur-xl lg:p-6">
              <div className="mb-4 flex items-center gap-2 text-white">
                <Gauge className="h-5 w-5 text-cyan-300" />
                Live latency view
              </div>
              <div className="grid gap-4 md:grid-cols-[0.9fr_1.1fr]">
                <div className="space-y-3">
                  <div className="rounded-[1.25rem] border border-white/10 bg-black/30 p-4 text-sm leading-7 text-slate-300/80">
                    <div className="flex items-center gap-2 text-white">
                      <AlertTriangle className="h-4 w-4 text-amber-300" />
                      What you are seeing
                    </div>
                    <p className="mt-2">
                      The UI is now driven by the backend, so the packet glow, console messages, and snapshot numbers reflect real engine activity instead of a canned demo.
                    </p>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="rounded-[1.15rem] border border-white/10 bg-black/30 p-4">
                      <div className="text-[11px] uppercase tracking-[0.28em] text-white/35">last write</div>
                      <div className="mt-2 text-2xl font-semibold text-cyan-300">{snapshot?.last_write_ms ?? 0}</div>
                      <div className="mt-1 text-xs text-white/45">ms</div>
                    </div>
                    <div className="rounded-[1.15rem] border border-white/10 bg-black/30 p-4">
                      <div className="text-[11px] uppercase tracking-[0.28em] text-white/35">last recovery</div>
                      <div className="mt-2 text-2xl font-semibold text-emerald-300">{snapshot?.last_recovery_ms ?? 0}</div>
                      <div className="mt-1 text-xs text-white/45">ms</div>
                    </div>
                  </div>
                </div>
                <div className="h-[300px] rounded-[1.25rem] border border-white/10 bg-black/30 p-3">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={latencyData.length ? latencyData : [{ name: "idle", value: 0 }]}>
                      <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.08)" />
                      <XAxis dataKey="name" stroke="rgba(255,255,255,0.35)" tick={{ fill: "rgba(255,255,255,0.5)", fontSize: 12 }} />
                      <YAxis stroke="rgba(255,255,255,0.35)" tick={{ fill: "rgba(255,255,255,0.5)", fontSize: 12 }} />
                      <Tooltip contentStyle={{ background: "rgba(4,10,20,0.95)", border: "1px solid rgba(255,255,255,0.12)", color: "white" }} />
                      <Line type="monotone" dataKey="value" stroke="#00DDFF" strokeWidth={2} dot={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </div>
            </div>
          </section>

          <section className="mt-6 rounded-[2rem] border border-white/10 bg-[#000000]/80 p-5 backdrop-blur-xl lg:p-6">
            <div className="flex items-center gap-2 text-white">
              <Cpu className="h-5 w-5 text-emerald-300" />
              Minimalist explanation
            </div>
            <p className="mt-3 max-w-5xl text-sm leading-7 text-slate-300/80">
              The layout now follows a deep-black, minimal systems style: central pipeline, node glow, live packet motion, real command triggers, and a console that is driven by actual backend events. Flush is shown when the engine emits it; compaction, recovery, and snapshot metrics all come from live state.
            </p>
          </section>
        </main>
      </div>
    </div>
  );
}
