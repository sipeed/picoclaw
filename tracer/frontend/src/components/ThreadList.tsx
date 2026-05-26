import type { Turn } from "../types";

export interface Thread {
  key: string;
  channel: string;
  chat_id: string;
  label: string;
  turn_count: number;
  last_timestamp: string;
  has_running: boolean;
}

export function buildThreads(turns: Turn[]): Thread[] {
  const map = new Map<string, Thread>();
  for (const turn of turns) {
    const key = turn.chat_id || `${turn.channel}:__no_chat__`;
    if (!map.has(key)) {
      // Derive a short readable label from chat_id
      // e.g. "pico:d936bc6e-acb4-..." → "pico · d936bc6e"
      const parts = turn.chat_id ? turn.chat_id.split(":") : [];
      const shortId = parts.length > 1
        ? parts.slice(1).join(":").slice(0, 8)
        : turn.chat_id.slice(0, 8);
      const label = `${turn.channel} · ${shortId}`;
      map.set(key, {
        key,
        channel: turn.channel,
        chat_id: turn.chat_id,
        label,
        turn_count: 0,
        last_timestamp: turn.timestamp,
        has_running: false,
      });
    }
    const thread = map.get(key)!;
    thread.turn_count += 1;
    thread.last_timestamp = turn.timestamp;
    if (turn.status === "running") thread.has_running = true;
  }
  return Array.from(map.values());
}

interface Props {
  threads: Thread[];
  selectedKey: string | null;
  onSelect: (key: string) => void;
}

const CHANNEL_COLORS: Record<string, string> = {
  pico: "var(--accent)",
  cron: "var(--yellow)",
  slack: "var(--green)",
  telegram: "var(--accent)",
  discord: "var(--purple)",
};

export default function ThreadList({ threads, selectedKey, onSelect }: Props) {
  return (
    <div
      style={{
        width: 200,
        flexShrink: 0,
        borderRight: "1px solid var(--border)",
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <div
        style={{
          padding: "8px 12px",
          fontSize: 11,
          fontWeight: 600,
          color: "var(--muted)",
          textTransform: "uppercase",
          letterSpacing: 0.8,
          borderBottom: "1px solid var(--border)",
          flexShrink: 0,
        }}
      >
        Threads
      </div>
      <div style={{ overflowY: "auto", flex: 1 }}>
        {threads.length === 0 && (
          <div style={{ padding: 16, color: "var(--muted)", fontSize: 12 }}>
            No threads yet.
          </div>
        )}
        {threads.map((thread) => {
          const selected = thread.key === selectedKey;
          const accentColor = CHANNEL_COLORS[thread.channel] ?? "var(--muted)";
          return (
            <button
              key={thread.key}
              onClick={() => onSelect(thread.key)}
              style={{
                width: "100%",
                textAlign: "left",
                padding: "10px 12px",
                background: selected ? "var(--surface2)" : "transparent",
                borderBottom: "1px solid var(--border)",
                display: "flex",
                flexDirection: "column",
                gap: 4,
                transition: "background 0.1s",
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                {thread.has_running && (
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: "50%",
                      background: "var(--yellow)",
                      flexShrink: 0,
                    }}
                  />
                )}
                <span
                  style={{
                    fontSize: 11,
                    fontWeight: 600,
                    color: selected ? accentColor : "var(--text)",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {thread.channel}
                </span>
              </div>
              <div style={{ fontSize: 10, color: "var(--muted)", fontFamily: "var(--font-mono)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {thread.chat_id.length > 24 ? thread.chat_id.slice(-24) : thread.chat_id}
              </div>
              <div style={{ display: "flex", gap: 8 }}>
                <span style={{ fontSize: 10, color: "var(--muted)" }}>
                  {thread.turn_count} turn{thread.turn_count !== 1 ? "s" : ""}
                </span>
                <span style={{ fontSize: 10, color: "var(--muted)" }}>
                  {thread.last_timestamp}
                </span>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
