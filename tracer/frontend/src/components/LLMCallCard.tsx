import { useState } from "react";
import type { LLMCall, Message, Tool } from "../types";

interface Props {
  call: LLMCall;
}

export default function LLMCallCard({ call }: Props) {
  const [openSection, setOpenSection] = useState<string | null>("messages");

  const toggle = (id: string) => setOpenSection((s) => (s === id ? null : id));

  const systemMsg = call.messages.find((m) => m.role === "system");
  const userMsgs = call.messages.filter((m) => m.role !== "system");

  return (
    <div
      style={{
        border: "1px solid var(--border)",
        borderRadius: 8,
        overflow: "hidden",
        background: "var(--surface)",
      }}
    >
      {/* Call header */}
      <div
        style={{
          padding: "10px 14px",
          display: "flex",
          alignItems: "center",
          gap: 12,
          borderBottom: "1px solid var(--border)",
          background: "var(--surface2)",
        }}
      >
        <span style={{ fontSize: 11, color: "var(--muted)" }}>#{call.iteration}</span>
        <code style={{ fontSize: 12, color: "var(--accent)" }}>{call.model}</code>
        <div style={{ display: "flex", gap: 8, marginLeft: "auto", flexWrap: "wrap" }}>
          <Chip label={`${call.messages_count} msgs`} />
          <Chip label={`${call.tools_count} tools`} />
          {call.content_len !== null && (
            <Chip label={`→ ${call.content_len} chars`} color="var(--green)" />
          )}
          {call.tool_calls_count !== null && call.tool_calls_count > 0 && (
            <Chip label={`${call.tool_calls_count} tool call${call.tool_calls_count > 1 ? "s" : ""}`} color="var(--purple)" />
          )}
          {call.has_reasoning && <Chip label="reasoning" color="var(--yellow)" />}
          {call.temperature && <Chip label={`T=${call.temperature}`} />}
        </div>
      </div>

      {/* Collapsible sections */}
      <div style={{ display: "flex", flexDirection: "column" }}>

        {/* System Prompt */}
        {systemMsg && (
          <Collapsible
            id="system"
            label="System Prompt"
            badge={call.system_prompt_len ? `${call.system_prompt_len} chars` : undefined}
            open={openSection === "system"}
            onToggle={toggle}
          >
            <MessageBlock msg={systemMsg} />
          </Collapsible>
        )}

        {/* Messages */}
        {userMsgs.length > 0 && (
          <Collapsible
            id="messages"
            label="Messages"
            badge={String(userMsgs.length)}
            open={openSection === "messages"}
            onToggle={toggle}
          >
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              {userMsgs.map((m, i) => (
                <MessageBlock key={i} msg={m} />
              ))}
            </div>
          </Collapsible>
        )}

        {/* Available Tools */}
        {call.tools.length > 0 && (
          <Collapsible
            id="tools"
            label="Available Tools"
            badge={String(call.tools.length)}
            open={openSection === "tools"}
            onToggle={toggle}
          >
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              {call.tools.map((t, i) => (
                <ToolItem key={i} tool={t} />
              ))}
            </div>
          </Collapsible>
        )}

        {/* Tool names only (when full data not available) */}
        {call.tools.length === 0 && call.tools_count > 0 && (
          <div
            style={{
              padding: "10px 14px",
              fontSize: 12,
              color: "var(--muted)",
              borderTop: "1px solid var(--border)",
            }}
          >
            {call.tools_count} tools available (run gateway with{" "}
            <code>--debug --no-truncate</code> to see full definitions)
          </div>
        )}
      </div>
    </div>
  );
}

function Collapsible({
  id,
  label,
  badge,
  open,
  onToggle,
  children,
}: {
  id: string;
  label: string;
  badge?: string;
  open: boolean;
  onToggle: (id: string) => void;
  children: React.ReactNode;
}) {
  return (
    <div style={{ borderTop: "1px solid var(--border)" }}>
      <button
        onClick={() => onToggle(id)}
        style={{
          width: "100%",
          padding: "9px 14px",
          display: "flex",
          alignItems: "center",
          gap: 8,
          textAlign: "left",
          background: open ? "var(--surface2)" : "transparent",
          transition: "background 0.1s",
        }}
      >
        <span style={{ fontSize: 11, color: "var(--muted)", flexShrink: 0 }}>
          {open ? "▼" : "▶"}
        </span>
        <span style={{ fontSize: 12, fontWeight: 500 }}>{label}</span>
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
      {open && (
        <div style={{ padding: "10px 14px", background: "var(--bg)" }}>{children}</div>
      )}
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

function Chip({ label, color }: { label: string; color?: string }) {
  return (
    <span
      style={{
        fontSize: 10,
        padding: "2px 7px",
        borderRadius: 10,
        background: "var(--bg)",
        border: "1px solid var(--border)",
        color: color ?? "var(--muted)",
        whiteSpace: "nowrap",
      }}
    >
      {label}
    </span>
  );
}
