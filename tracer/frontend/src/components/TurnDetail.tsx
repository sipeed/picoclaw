import { useState } from "react";
import type { Turn, Message, Tool } from "../types";

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

  // Stitch together all NEW messages added across iterations into one flat list.
  const conversation: Message[] = [];
  let prevLen = 0;
  for (let i = 0; i < turn.llm_calls.length; i++) {
    const call = turn.llm_calls[i];
    if (i === 0) {
      // Take everything from the last user message onwards (the "new" part of this turn)
      const nonSys = call.messages.filter((m) => m.role !== "system");
      let lastUserIdx = -1;
      for (let j = nonSys.length - 1; j >= 0; j--) {
        if (nonSys[j].role === "user") {
          lastUserIdx = j;
          break;
        }
      }
      const slice = lastUserIdx >= 0 ? nonSys.slice(lastUserIdx) : nonSys;
      conversation.push(...slice);
      prevLen = call.messages.length;
    } else {
      // Add only the messages beyond what the previous call had
      conversation.push(...call.messages.slice(prevLen));
      prevLen = call.messages.length;
    }
  }

  // First call's system prompt and tools (these are stable across iterations)
  const firstCall = turn.llm_calls[0];
  const systemMsg = firstCall?.messages.find((m) => m.role === "system");
  const tools = firstCall?.tools ?? [];
  const model = firstCall?.model ?? "";

  return (
    <div style={{ flex: 1, overflowY: "auto", padding: 20, display: "flex", flexDirection: "column", gap: 16 }}>
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
        {model && <Meta label="Model" value={model} color="var(--accent)" />}
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

      {systemMsg && (
        <CollapsibleSection title="System Prompt" badge={`${systemMsg.content.length} chars`}>
          <MessageBlock msg={systemMsg} />
        </CollapsibleSection>
      )}

      {conversation.length > 0 && (
        <Section title="Conversation">
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {conversation.map((m, i) => (
              <MessageBlock key={i} msg={m} />
            ))}
          </div>
        </Section>
      )}

      {tools.length > 0 && (
        <CollapsibleSection title="Available Tools" badge={String(tools.length)}>
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            {tools.map((t, i) => (
              <ToolItem key={i} tool={t} />
            ))}
          </div>
        </CollapsibleSection>
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

function MessageBlock({ msg }: { msg: Message }) {
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
      <div style={{ fontSize: 10, fontWeight: 600, color: roleColor, marginBottom: 6, textTransform: "uppercase", letterSpacing: 0.6 }}>
        {msg.role}
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
