import type { Turn } from "../types";

interface Props {
  turns: Turn[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  loading: boolean;
}

function StatusDot({ status }: { status: string }) {
  const color =
    status === "completed" ? "var(--green)"
    : status === "running" ? "var(--yellow)"
    : "var(--red)";
  return (
    <span
      style={{
        display: "inline-block",
        width: 7,
        height: 7,
        borderRadius: "50%",
        background: color,
        flexShrink: 0,
      }}
    />
  );
}

export default function TurnList({ turns, selectedId, onSelect, loading }: Props) {
  return (
    <div
      style={{
        width: 240,
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
        Turns
      </div>

      <div style={{ overflowY: "auto", flex: 1 }}>
        {loading && turns.length === 0 && (
          <div style={{ padding: 16, color: "var(--muted)", fontSize: 12 }}>
            Waiting for gateway logs…
          </div>
        )}
        {!loading && turns.length === 0 && (
          <div style={{ padding: 16, color: "var(--muted)", fontSize: 12 }}>
            No turns yet. Send a message in the PicoClaw chat.
          </div>
        )}
        {turns.map((turn) => {
          const selected = turn.turn_id === selectedId;
          // Use the LATEST user message in the first LLM call as a readable label
          let userMsg = "";
          const firstCall = turn.llm_calls[0];
          if (firstCall) {
            for (let i = firstCall.messages.length - 1; i >= 0; i--) {
              if (firstCall.messages[i].role === "user") {
                userMsg = firstCall.messages[i].content;
                break;
              }
            }
          }
          return (
            <button
              key={turn.turn_id}
              onClick={() => onSelect(turn.turn_id)}
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
                <StatusDot status={turn.status} />
                <span
                  style={{
                    fontSize: 12,
                    color: selected ? "var(--accent)" : "var(--text)",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                    flex: 1,
                  }}
                >
                  {userMsg || turn.turn_id}
                </span>
              </div>
              <div style={{ display: "flex", gap: 8, paddingLeft: 13, alignItems: "baseline" }}>
                <span style={{ color: "var(--muted)", fontSize: 10, fontFamily: "var(--font-mono)" }}>
                  {turn.turn_id}
                </span>
                <span style={{ color: "var(--muted)", fontSize: 11 }}>
                  {turn.timestamp}
                </span>
                {turn.duration_ms !== null && (
                  <span style={{ color: "var(--muted)", fontSize: 11 }}>
                    {(turn.duration_ms / 1000).toFixed(1)}s
                  </span>
                )}
              </div>
              {turn.llm_calls.length > 0 && (
                <div style={{ paddingLeft: 13, color: "var(--muted)", fontSize: 11 }}>
                  {turn.llm_calls.length} LLM call{turn.llm_calls.length > 1 ? "s" : ""}
                  {turn.tool_execs.length > 0 &&
                    ` · ${turn.tool_execs.length} tool${turn.tool_execs.length > 1 ? "s" : ""}`}
                </div>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
