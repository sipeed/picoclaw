import { useState } from "react";
import type { Turn, Message, Tool, LLMCall } from "../types";

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
    <div style={{ flex: 1, minHeight: 0, overflowY: "auto", padding: 20, display: "flex", flexDirection: "column", gap: 16 }}>
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
      </div>

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

      {turn.llm_calls.map((call, i) => (
        <LLMCallSection key={i} call={call} index={i} />
      ))}
    </div>
  );
}

function LLMCallSection({ call, index }: { call: LLMCall; index: number }) {
  const [open, setOpen] = useState(true);
  return (
    <div
      style={{
        flexShrink: 0,
        border: "1px solid var(--border)",
        borderRadius: 8,
        background: "var(--surface)",
      }}
    >
      <button
        onClick={() => setOpen((o) => !o)}
        style={{
          width: "100%",
          display: "flex",
          alignItems: "baseline",
          gap: 10,
          padding: "10px 14px",
          background: open ? "var(--surface2)" : "transparent",
          borderBottom: open ? "1px solid var(--border)" : "none",
          textAlign: "left",
          transition: "background 0.1s",
        }}
      >
        <span style={{ fontSize: 10, color: "var(--muted)" }}>{open ? "▼" : "▶"}</span>
        <span style={{ fontSize: 12, fontWeight: 600 }}>LLM Call #{index + 1}</span>
        <span style={{ fontSize: 12, color: "var(--accent)" }}>{call.model}</span>
        <span style={{ fontSize: 11, color: "var(--muted)" }}>
          {call.messages_count} messages
        </span>
        {call.tools_count > 0 && (
          <span style={{ fontSize: 11, color: "var(--muted)" }}>
            {call.tools_count} tools
          </span>
        )}
        {call.content_len !== null && (
          <span style={{ fontSize: 11, color: "var(--green)" }}>
            → {call.content_len} chars
          </span>
        )}
        {call.tool_calls_count !== null && call.tool_calls_count > 0 && (
          <span style={{ fontSize: 11, color: "var(--purple)" }}>
            {call.tool_calls_count} tool call{call.tool_calls_count > 1 ? "s" : ""}
          </span>
        )}
        <span style={{ marginLeft: "auto", fontSize: 11, color: "var(--muted)" }}>
          {call.timestamp}
        </span>
      </button>
      {open && (
        <div style={{ padding: 12, display: "flex", flexDirection: "column", gap: 6 }}>
          {call.messages.length === 0 ? (
            <div style={{ fontSize: 12, color: "var(--muted)", fontStyle: "italic", padding: "8px 12px", border: "1px solid var(--border)", borderRadius: 6 }}>
              (messages not captured — gateway log_level must be "debug")
            </div>
          ) : (
            call.messages.map((m, i) => <MessageBlock key={i} msg={m} index={i} />)
          )}
          {call.tools.length > 0 && (
            <CollapsibleSection
              title="Available Tools"
              badge={String(call.tools.length)}
            >
              <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                {call.tools.map((t, i) => (
                  <ToolItem key={i} tool={t} />
                ))}
              </div>
            </CollapsibleSection>
          )}
        </div>
      )}
    </div>
  );
}

function CollapsibleSection({
  title,
  badge,
  children,
  defaultOpen = false,
}: {
  title: string;
  badge?: string;
  children: React.ReactNode;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <button
        onClick={() => setOpen((o) => !o)}
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          fontSize: 11,
          fontWeight: 600,
          color: "var(--muted)",
          textTransform: "uppercase",
          letterSpacing: 0.8,
          textAlign: "left",
          background: "transparent",
        }}
      >
        <span style={{ fontSize: 10 }}>{open ? "▼" : "▶"}</span>
        {title}
        {badge && (
          <span
            style={{
              fontSize: 10,
              color: "var(--muted)",
              background: "var(--surface2)",
              padding: "1px 6px",
              borderRadius: 10,
              border: "1px solid var(--border)",
            }}
          >
            {badge}
          </span>
        )}
      </button>
      {open && children}
    </div>
  );
}

function MessageBlock({ msg, index }: { msg: Message; index?: number }) {
  const [expanded, setExpanded] = useState(false);
  const isLong = msg.content.length > 600;
  const display = isLong && !expanded ? msg.content.slice(0, 600) + "…" : msg.content;
  const roleColor =
    msg.role === "system" ? "var(--purple)"
    : msg.role === "user" ? "var(--accent)"
    : msg.role === "assistant" ? "var(--green)"
    : "var(--yellow)";

  return (
    <div
      style={{
        padding: "10px 12px",
        background: "var(--surface)",
        borderRadius: 6,
        border: "1px solid var(--border)",
      }}
    >
      <div style={{ display: "flex", alignItems: "baseline", gap: 8, marginBottom: 6 }}>
        {index !== undefined && (
          <span style={{ fontSize: 10, color: "var(--muted)", fontFamily: "monospace" }}>
            [{index}]
          </span>
        )}
        <span style={{ fontSize: 10, fontWeight: 600, color: roleColor, textTransform: "uppercase", letterSpacing: 0.6 }}>
          {msg.role}
        </span>
        <span style={{ fontSize: 10, color: "var(--muted)", marginLeft: "auto" }}>
          {msg.content.length} chars
        </span>
      </div>
      <pre
        style={{
          fontSize: 12,
          color: "var(--text)",
          whiteSpace: "pre-wrap",
          wordBreak: "break-word",
          margin: 0,
          lineHeight: 1.55,
        }}
      >
        {display}
      </pre>
      {isLong && (
        <button
          onClick={() => setExpanded((e) => !e)}
          style={{ marginTop: 6, fontSize: 11, color: "var(--accent)", textDecoration: "underline" }}
        >
          {expanded ? "Show less" : `Show ${msg.content.length - 600} more chars`}
        </button>
      )}
    </div>
  );
}

function ToolItem({ tool }: { tool: Tool }) {
  const [open, setOpen] = useState(false);
  const truncatedDesc =
    tool.description.length > 120 && !open
      ? tool.description.slice(0, 120) + "…"
      : tool.description;

  return (
    <div
      onClick={() => setOpen((o) => !o)}
      style={{
        padding: "8px 10px",
        background: "var(--bg)",
        borderRadius: 5,
        border: "1px solid var(--border)",
        cursor: "pointer",
      }}
    >
      <div style={{ display: "flex", gap: 8, alignItems: "baseline" }}>
        <span style={{ fontSize: 10, color: "var(--muted)", flexShrink: 0 }}>
          {open ? "▼" : "▶"}
        </span>
        <code style={{ color: "var(--accent)", fontSize: 12 }}>{tool.name}</code>
        {tool.description && (
          <span style={{ color: "var(--muted)", fontSize: 11, flex: 1, whiteSpace: open ? "pre-wrap" : "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
            {truncatedDesc}
          </span>
        )}
      </div>
      {open && tool.parameters && (
        <pre
          style={{
            marginTop: 8,
            padding: "8px 10px",
            background: "var(--surface)",
            border: "1px solid var(--border)",
            borderRadius: 4,
            fontSize: 11,
            color: "var(--text)",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
            lineHeight: 1.5,
            overflow: "auto",
            maxHeight: 400,
          }}
        >
          {tool.parameters}
        </pre>
      )}
    </div>
  );
}

function Meta({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <span style={{ fontSize: 10, color: "var(--muted)", textTransform: "uppercase", letterSpacing: 0.6 }}>
        {label}
      </span>
      <span
        style={{
          fontSize: 12,
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
