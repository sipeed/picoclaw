import { useState } from 'preact/hooks';
import { apiFetch } from '../../hooks/use-api';
import { formatAge, formatTokens, formatSessionLabel, escapeHtml } from '../../utils';

interface SessionSectionProps {
  sessions: any[] | null;
  stats: any;
  graph: any;
  context: any;
}

export function SessionSection({
  sessions,
  stats,
  graph,
  context,
}: SessionSectionProps) {
  return (
    <>
      <ActiveSessions sessions={sessions} />
      <StatsCards stats={stats} />
      {context && <ContextCard context={context} />}
      {graph && <SessionGraph graph={graph} />}
    </>
  );
}

function ActiveSessions({ sessions }: { sessions: any[] | null }) {
  if (!sessions || sessions.length === 0) {
    return (
      <div class="card glass">
        <div class="card-title">Active Sessions</div>
        <div style={{ color: 'var(--hint)', fontSize: '13px' }}>
          No active sessions
        </div>
      </div>
    );
  }

  return (
    <div class="card glass">
      <div class="card-title">Active Sessions</div>
      {sessions.map((s) => {
        const label = formatSessionLabel(s.session_key);
        const isHeartbeat = s.session_key.startsWith('heartbeat:');
        const latestMsg = s.latest_message || null;

        return (
          <div
            key={s.session_key}
            style={{
              padding: '8px 0',
              borderBottom: '1px solid var(--secondary-bg)',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
            >
              <span
                style={{
                  color: isHeartbeat ? 'var(--link)' : 'var(--done)',
                  fontSize: '10px',
                }}
              >
                {isHeartbeat ? '\u{1F916}' : '\u25CF'}
              </span>
              <span style={{ fontWeight: 600, fontSize: '14px', flex: 1 }}>
                {label}
              </span>
              <span
                style={{
                  color: 'var(--hint)',
                  fontSize: '12px',
                  flexShrink: 0,
                }}
              >
                {s.turn_count || 0} turns
              </span>
              <span
                style={{
                  marginLeft: '4px',
                  color: 'var(--hint)',
                  fontSize: '12px',
                }}
              >
                {formatAge(s.age_sec)}
              </span>
            </div>
            {latestMsg && (
              <div
                style={{
                  color: 'var(--hint)',
                  fontSize: '12px',
                  paddingLeft: '22px',
                  marginTop: '2px',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {latestMsg}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

function StatsCards({ stats }: { stats: any }) {
  if (!stats || stats.status === 'stats not enabled') {
    return (
      <div class="empty-state">
        Stats tracking not enabled.
        <br />
        Start gateway with --stats flag.
      </div>
    );
  }

  const since = stats.since
    ? new Date(stats.since).toLocaleDateString()
    : 'N/A';
  const today = stats.today || {};

  return (
    <>
      <div class="card glass">
        <div class="card-title">Today</div>
        <div class="stat-row">
          <span class="stat-label">Prompts</span>
          <span class="stat-value">{today.prompts || 0}</span>
        </div>
        <div class="stat-row">
          <span class="stat-label">Requests</span>
          <span class="stat-value">{today.requests || 0}</span>
        </div>
        <div class="stat-row">
          <span class="stat-label">Tokens</span>
          <span class="stat-value">
            {formatTokens(today.total_tokens || 0)}
          </span>
        </div>
      </div>
      <div class="card glass">
        <div class="card-title">All Time (since {since})</div>
        <div class="stat-row">
          <span class="stat-label">Prompts</span>
          <span class="stat-value">{stats.total_prompts || 0}</span>
        </div>
        <div class="stat-row">
          <span class="stat-label">Requests</span>
          <span class="stat-value">{stats.total_requests || 0}</span>
        </div>
        <div class="stat-row">
          <span class="stat-label">Total Tokens</span>
          <span class="stat-value">
            {formatTokens(stats.total_tokens || 0)}
          </span>
        </div>
        <div class="stat-row">
          <span class="stat-label">Prompt Tokens</span>
          <span class="stat-value">
            {formatTokens(stats.total_prompt_tokens || 0)}
          </span>
        </div>
        <div class="stat-row">
          <span class="stat-label">Completion Tokens</span>
          <span class="stat-value">
            {formatTokens(stats.total_completion_tokens || 0)}
          </span>
        </div>
      </div>
    </>
  );
}

function ContextCard({ context }: { context: any }) {
  const [showPrompt, setShowPrompt] = useState(false);
  const [promptText, setPromptText] = useState<string | null>(null);
  const [promptLoading, setPromptLoading] = useState(false);

  const togglePrompt = async () => {
    if (showPrompt) {
      setShowPrompt(false);
      return;
    }
    setPromptLoading(true);
    try {
      const data = await apiFetch('/miniapp/api/prompt');
      setPromptText(data.prompt || '(empty)');
      setShowPrompt(true);
    } catch {}
    setPromptLoading(false);
  };

  const wd = context.work_dir || '\u2014';
  const pwd = context.plan_work_dir || '\u2014';
  const ws = context.workspace || '\u2014';

  return (
    <div class="card glass">
      <div class="card-title">Context</div>
      <div style={{ fontSize: '12px' }}>
        <div class="stat-row">
          <span class="stat-label">workDir</span>
          <span
            class="stat-value"
            style={{
              fontSize: '12px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
            title={wd}
          >
            {wd}
          </span>
        </div>
        <div class="stat-row">
          <span class="stat-label">planWorkDir</span>
          <span
            class="stat-value"
            style={{
              fontSize: '12px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
            title={pwd}
          >
            {pwd}
          </span>
        </div>
        <div class="stat-row">
          <span class="stat-label">workspace</span>
          <span
            class="stat-value"
            style={{
              fontSize: '12px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
            title={ws}
          >
            {ws}
          </span>
        </div>
      </div>
      {context.bootstrap && context.bootstrap.length > 0 && (
        <div style={{ marginTop: '8px' }}>
          {context.bootstrap.map((b: any, i: number) => {
            const path = b.path || '\u2014';
            const scope = b.scope === 'global' ? 'global' : 'project';
            return (
              <div
                key={i}
                style={{
                  display: 'flex',
                  gap: '8px',
                  padding: '2px 0',
                  fontSize: '12px',
                }}
              >
                <span
                  style={{
                    minWidth: '90px',
                    fontWeight: 600,
                    color: b.path ? 'var(--text)' : 'var(--hint)',
                  }}
                >
                  {b.name}
                </span>
                <span
                  style={{
                    color: 'var(--hint)',
                    flex: 1,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                  title={path}
                >
                  {path}
                </span>
                <span style={{ color: 'var(--hint)', fontSize: '11px' }}>
                  {scope}
                </span>
              </div>
            );
          })}
        </div>
      )}
      <div style={{ marginTop: '8px', textAlign: 'center' }}>
        <button
          onClick={togglePrompt}
          style={{
            background: 'var(--secondary-bg)',
            color: 'var(--text)',
            border: 'none',
            padding: '6px 12px',
            borderRadius: '8px',
            fontSize: '12px',
            cursor: 'pointer',
          }}
        >
          {promptLoading
            ? 'Loading...'
            : showPrompt
              ? 'Hide System Prompt'
              : 'Show System Prompt'}
        </button>
      </div>
      {showPrompt && promptText && (
        <pre
          style={{
            marginTop: '8px',
            fontSize: '11px',
            maxHeight: '400px',
            overflow: 'auto',
            background: 'var(--secondary-bg)',
            padding: '8px',
            borderRadius: '6px',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {promptText}
        </pre>
      )}
    </div>
  );
}

function SessionGraph({ graph }: { graph: any }) {
  if (!graph || !graph.nodes || graph.nodes.length === 0) return null;

  // Build parent->children map
  const childrenMap: Record<string, string[]> = {};
  const nodeMap: Record<string, any> = {};
  const roots: string[] = [];

  graph.nodes.forEach((n: any) => {
    childrenMap[n.key] = [];
    nodeMap[n.key] = n;
  });
  graph.edges.forEach((e: any) => {
    if (childrenMap[e.from]) childrenMap[e.from].push(e.to);
  });
  graph.nodes.forEach((n: any) => {
    const isChild = graph.edges.some((e: any) => e.to === n.key);
    if (!isChild) roots.push(n.key);
  });

  function renderNode(key: string): any {
    const n = nodeMap[key];
    if (!n) return null;
    const icon = n.status === 'completed' ? '\u2713' : '\u25CF';
    const iconClass = n.status === 'completed' ? 'completed' : 'active';
    const label = n.label || n.short_key || n.key;
    const kids = childrenMap[key] || [];

    return (
      <li class="session-tree-node" key={key}>
        <span class={`session-tree-icon ${iconClass}`}>{icon}</span>
        <span class="session-tree-label">{label}</span>
        <span class="session-tree-meta">turns={n.turn_count}</span>
        {kids.length > 0 && (
          <ul class="session-tree-children">
            {kids.map(renderNode)}
          </ul>
        )}
      </li>
    );
  }

  return (
    <div class="card glass">
      <div class="card-title">Session Graph</div>
      <ul class="session-tree">{roots.map(renderNode)}</ul>
    </div>
  );
}
