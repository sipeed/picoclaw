import type { Turn } from "../types";
import LLMCallCard from "./LLMCallCard";

interface Props {
  turn: Turn | null;
}

export default function TurnDetail({ turn }: Props) {
  if (!turn) {
    return (
      <div
        style={{
          flex: 1,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          color: "var(--muted)",
        }}
      >
        Select a turn to inspect
      </div>
    );
  }

  const statusColor =
    turn.status === "completed" ? "var(--green)"
    : turn.status === "running" ? "var(--yellow)"
    : "var(--red)";

  return (
    <div style={{ flex: 1, overflowY: "auto", padding: 20, display: "flex", flexDirection: "column", gap: 16 }}>
      {/* Turn header */}
      <div
        style={{
          background: "var(--surface)",
          border: "1px solid var(--border)",
          borderRadius: 8,
          padding: "14px 16px",
          display: "flex",
          flexWrap: "wrap",
          gap: "12px 24px",
        }}
      >
        <Meta label="Turn" value={turn.turn_id} mono />
        <Meta label="Time" value={turn.timestamp} />
        <Meta label="Channel" value={turn.channel} />
        <Meta label="Sender" value={turn.sender_id} />
        <Meta label="Status" value={turn.status} color={statusColor} />
        {turn.duration_ms !== null && (
          <Meta label="Duration" value={`${(turn.duration_ms / 1000).toFixed(2)}s`} />
        )}
        {turn.iterations > 0 && (
          <Meta label="Iterations" value={String(turn.iterations)} />
        )}
        <Meta label="User msg" value={`${turn.user_len} chars`} />
      </div>

      {/* Tool executions summary */}
      {turn.tool_execs.length > 0 && (
        <Section title="Tool Executions">
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            {turn.tool_execs.map((te, i) => (
              <div
                key={i}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 10,
                  padding: "8px 12px",
                  background: "var(--surface)",
                  borderRadius: 6,
                  border: "1px solid var(--border)",
                }}
              >
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: "50%",
                    background: te.is_error ? "var(--red)" : "var(--green)",
                    flexShrink: 0,
                  }}
                />
                <code style={{ color: "var(--accent)", fontSize: 12 }}>{te.tool}</code>
                <span style={{ color: "var(--muted)", fontSize: 11, marginLeft: "auto" }}>
                  {te.duration_ms !== null ? `${te.duration_ms}ms` : "..."}
                </span>
                {te.for_llm_len !== null && te.for_llm_len > 0 && (
                  <span style={{ color: "var(--muted)", fontSize: 11 }}>
                    {te.for_llm_len} chars
                  </span>
                )}
                {te.is_error && (
                  <span style={{ color: "var(--red)", fontSize: 11 }}>error</span>
                )}
              </div>
            ))}
          </div>
        </Section>
      )}

      {/* LLM Calls */}
      {turn.llm_calls.length > 0 && (
        <Section title={`LLM Calls (${turn.llm_calls.length})`}>
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            {turn.llm_calls.map((call) => (
              <LLMCallCard key={call.iteration} call={call} />
            ))}
          </div>
        </Section>
      )}

      {turn.llm_calls.length === 0 && (
        <div style={{ color: "var(--muted)", fontSize: 12 }}>No LLM calls recorded yet.</div>
      )}
    </div>
  );
}

function Meta({
  label,
  value,
  mono,
  color,
}: {
  label: string;
  value: string;
  mono?: boolean;
  color?: string;
}) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <span style={{ fontSize: 10, color: "var(--muted)", textTransform: "uppercase", letterSpacing: 0.6 }}>
        {label}
      </span>
      <span
        style={{
          fontSize: 12,
          fontFamily: mono ? "var(--font-mono)" : undefined,
          color: color ?? "var(--text)",
          wordBreak: "break-all",
        }}
      >
        {value}
      </span>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <div
        style={{
          fontSize: 11,
          fontWeight: 600,
          color: "var(--muted)",
          textTransform: "uppercase",
          letterSpacing: 0.8,
        }}
      >
        {title}
      </div>
      {children}
    </div>
  );
}
