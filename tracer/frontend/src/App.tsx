import { useEffect, useState, useCallback, useMemo } from "react";
import type { Turn } from "./types";
import ThreadList, { buildThreads } from "./components/ThreadList";
import TurnList from "./components/TurnList";
import TurnDetail from "./components/TurnDetail";

export default function App() {
  const [turns, setTurns] = useState<Turn[]>([]);
  const [selectedThreadKey, setSelectedThreadKey] = useState<string | null>(null);
  const [selectedTurnId, setSelectedTurnId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchTraces = useCallback(async () => {
    try {
      const res = await fetch("/api/traces");
      const data: Turn[] = await res.json();
      setTurns(data);
      setLastUpdated(new Date());
    } catch {
      // backend not ready yet
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTraces();
    const interval = setInterval(fetchTraces, 2000);
    return () => clearInterval(interval);
  }, [fetchTraces]);

  const threads = useMemo(() => buildThreads(turns), [turns]);

  // Auto-select first thread when threads load
  useEffect(() => {
    if (!selectedThreadKey && threads.length > 0) {
      setSelectedThreadKey(threads[0].key);
    }
  }, [threads, selectedThreadKey]);

  const threadTurns = useMemo(() => {
    if (!selectedThreadKey) return [];
    return turns.filter(
      (t) => (t.chat_id || `${t.channel}:__no_chat__`) === selectedThreadKey
    );
  }, [turns, selectedThreadKey]);

  // Auto-select first turn in thread when thread changes
  useEffect(() => {
    if (threadTurns.length > 0) {
      setSelectedTurnId(threadTurns[0].turn_id);
    } else {
      setSelectedTurnId(null);
    }
  }, [selectedThreadKey]); // eslint-disable-line react-hooks/exhaustive-deps

  const selectedTurn = threadTurns.find((t) => t.turn_id === selectedTurnId) ?? null;

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh" }}>
      <Header lastUpdated={lastUpdated} threadCount={threads.length} />
      <div style={{ display: "flex", flex: 1, overflow: "hidden" }}>
        <ThreadList
          threads={threads}
          selectedKey={selectedThreadKey}
          onSelect={(key) => {
            setSelectedThreadKey(key);
            setSelectedTurnId(null);
          }}
        />
        <TurnList
          turns={threadTurns}
          selectedId={selectedTurnId}
          onSelect={setSelectedTurnId}
          loading={loading && threads.length === 0}
        />
        <TurnDetail turn={selectedTurn} />
      </div>
    </div>
  );
}

function Header({ lastUpdated, threadCount }: { lastUpdated: Date | null; threadCount: number }) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 16px",
        background: "var(--surface)",
        borderBottom: "1px solid var(--border)",
        flexShrink: 0,
      }}
    >
      <span style={{ fontSize: 15, fontWeight: 600, letterSpacing: -0.3 }}>
        🦞 PicoClaw Traces
      </span>
      <span style={{ color: "var(--muted)", fontSize: 12 }}>
        {threadCount} thread{threadCount !== 1 ? "s" : ""}
      </span>
      <span style={{ marginLeft: "auto", color: "var(--muted)", fontSize: 11 }}>
        {lastUpdated ? `updated ${lastUpdated.toLocaleTimeString()}` : "connecting..."}
      </span>
    </div>
  );
}
