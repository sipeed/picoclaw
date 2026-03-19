import { useEffect, useState, useCallback } from 'preact/hooks';
import { apiFetch, apiPost } from '../../hooks/use-api';
import { escapeHtml } from '../../utils';
import { renderMarkdown } from '../../markdown';

const STATUS_COLORS: Record<string, { bg: string; text: string }> = {
  pending: { bg: 'rgba(234,179,8,0.15)', text: '#ca8a04' },
  active: { bg: 'rgba(59,130,246,0.15)', text: '#2563eb' },
  completed: { bg: 'rgba(34,197,94,0.15)', text: '#16a34a' },
  failed: { bg: 'rgba(239,68,68,0.15)', text: '#dc2626' },
  canceled: { bg: 'rgba(107,114,128,0.15)', text: '#6b7280' },
};

interface ResearchSectionProps {
  active: boolean;
}

export function ResearchSection({ active }: ResearchSectionProps) {
  const [tasks, setTasks] = useState<any[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [detailId, setDetailId] = useState<string | null>(null);

  const loadTasks = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch('/miniapp/api/research');
      setTasks(data);
    } catch {
      setTasks(null);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    if (active) loadTasks();
  }, [active]);

  if (detailId) {
    return (
      <TaskDetail
        taskId={detailId}
        onBack={() => {
          setDetailId(null);
          loadTasks();
        }}
      />
    );
  }

  return (
    <div class="card glass">
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '12px',
        }}
      >
        <span class="card-title" style={{ margin: 0 }}>
          Research Tasks
        </span>
        <button
          class="send-btn"
          style={{ padding: '6px 14px', fontSize: '13px' }}
          onClick={() => setShowForm(true)}
        >
          + New
        </button>
      </div>
      {showForm && (
        <NewTaskForm
          onCreated={() => {
            setShowForm(false);
            loadTasks();
          }}
          onCancel={() => setShowForm(false)}
        />
      )}
      {loading && !tasks ? (
        <div class="loading" style={{ padding: '12px' }}>
          Loading tasks...
        </div>
      ) : !tasks || tasks.length === 0 ? (
        <div class="empty-state" style={{ padding: '24px' }}>
          No research tasks yet.
        </div>
      ) : (
        tasks.map((t) => {
          const sc = STATUS_COLORS[t.status] || STATUS_COLORS.pending;
          return (
            <div
              key={t.id}
              class="card glass glass-interactive"
              style={{
                cursor: 'pointer',
                padding: '14px',
                ...(t.focused
                  ? { borderLeft: '3px solid #a855f7' }
                  : {}),
              }}
              onClick={() => setDetailId(t.id)}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: '8px',
                }}
              >
                <span style={{ fontWeight: 600, fontSize: '15px' }}>
                  {t.title}
                </span>
                <div
                  style={{
                    display: 'flex',
                    gap: '4px',
                    alignItems: 'center',
                  }}
                >
                  {t.focused && (
                    <span
                      style={{
                        fontSize: '10px',
                        fontWeight: 600,
                        padding: '2px 6px',
                        borderRadius: '8px',
                        background: 'rgba(168,85,247,0.2)',
                        color: '#a855f7',
                      }}
                    >
                      focused
                    </span>
                  )}
                  <span
                    style={{
                      fontSize: '11px',
                      fontWeight: 600,
                      padding: '2px 8px',
                      borderRadius: '10px',
                      background: sc.bg,
                      color: sc.text,
                    }}
                  >
                    {t.status}
                  </span>
                </div>
              </div>
              {t.description && (
                <div
                  style={{
                    color: 'var(--hint)',
                    fontSize: '13px',
                    marginTop: '4px',
                    lineHeight: 1.4,
                  }}
                >
                  {t.description.substring(0, 120)}
                </div>
              )}
              <div
                style={{
                  color: 'var(--hint)',
                  fontSize: '11px',
                  marginTop: '6px',
                }}
              >
                {t.document_count} docs{' \u00B7 \u23F1 '}
                {(t.interval === '24h' ? '1d' : t.interval) || '1d'}
                {t.last_researched_at &&
                  ' \u00B7 last: ' +
                    new Date(t.last_researched_at).toLocaleDateString()}
              </div>
            </div>
          );
        })
      )}
    </div>
  );
}

function NewTaskForm({
  onCreated,
  onCancel,
}: {
  onCreated: () => void;
  onCancel: () => void;
}) {
  const [title, setTitle] = useState('');
  const [desc, setDesc] = useState('');

  const handleCreate = async () => {
    const t = title.trim();
    if (!t) return;
    try {
      await apiPost('/miniapp/api/research', {
        title: t,
        description: desc.trim(),
      });
      onCreated();
    } catch {}
  };

  return (
    <div class="card glass" style={{ padding: '12px', marginBottom: '12px' }}>
      <input
        class="send-input glass glass-interactive"
        placeholder="Task title..."
        value={title}
        onInput={(e) => setTitle((e.target as HTMLInputElement).value)}
        style={{ width: '100%', marginBottom: '8px' }}
      />
      <textarea
        class="send-input glass glass-interactive"
        placeholder="Description (optional)..."
        value={desc}
        onInput={(e) => setDesc((e.target as HTMLTextAreaElement).value)}
        style={{
          width: '100%',
          minHeight: '60px',
          resize: 'vertical',
          marginBottom: '8px',
          borderRadius: '12px',
          padding: '10px 16px',
        }}
      />
      <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
        <button
          class="send-btn"
          style={{
            padding: '6px 14px',
            fontSize: '13px',
            background: 'var(--hint)',
          }}
          onClick={onCancel}
        >
          Cancel
        </button>
        <button
          class="send-btn"
          style={{ padding: '6px 14px', fontSize: '13px' }}
          onClick={handleCreate}
        >
          Create
        </button>
      </div>
    </div>
  );
}

function TaskDetail({
  taskId,
  onBack,
}: {
  taskId: string;
  onBack: () => void;
}) {
  const [task, setTask] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch('/miniapp/api/research/' + taskId);
      setTask(data);
    } catch {}
    setLoading(false);
  }, [taskId]);

  useEffect(() => {
    load();
  }, [taskId]);

  const handleAction = async (action: string) => {
    try {
      await apiPost('/miniapp/api/research/' + taskId, { action });
      load();
    } catch {}
  };

  const handleFocus = async (recall: boolean) => {
    try {
      await apiPost('/miniapp/api/research/focus', {
        action: recall ? 'recall' : 'forget',
        task_id: taskId,
      });
      load();
    } catch {}
  };

  const handleInterval = async (interval: string) => {
    try {
      await apiPost('/miniapp/api/research/' + taskId, {
        action: 'set_interval',
        interval,
      });
      load();
    } catch {}
  };

  if (loading || !task) {
    return (
      <div class="card glass">
        <button class="git-back-btn" onClick={onBack}>
          {'\u2039'} Back
        </button>
        <div class="loading">Loading...</div>
      </div>
    );
  }

  const sc = STATUS_COLORS[task.status] || STATUS_COLORS.pending;
  const canCancel = task.status === 'pending' || task.status === 'active';
  const canReopen = task.status === 'completed' || task.status === 'failed';
  const curInterval = (task.interval === '24h' ? '1d' : task.interval) || '1d';

  return (
    <>
      <button class="git-back-btn" onClick={onBack}>
        {'\u2039'} Back
      </button>
      <div class="card glass">
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '8px',
            marginBottom: '8px',
          }}
        >
          <span style={{ fontWeight: 700, fontSize: '17px' }}>
            {task.title}
          </span>
          <span
            style={{
              fontSize: '11px',
              fontWeight: 600,
              padding: '2px 8px',
              borderRadius: '10px',
              background: sc.bg,
              color: sc.text,
            }}
          >
            {task.status}
          </span>
        </div>
        {task.description && (
          <div
            style={{
              color: 'var(--hint)',
              fontSize: '13px',
              lineHeight: 1.5,
              marginBottom: '8px',
              whiteSpace: 'pre-wrap',
            }}
          >
            {task.description}
          </div>
        )}
        <div
          style={{
            color: 'var(--hint)',
            fontSize: '11px',
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
          }}
        >
          <span>Interval:</span>
          <select
            value={curInterval}
            onChange={(e) =>
              handleInterval((e.target as HTMLSelectElement).value)
            }
            style={{
              fontSize: '11px',
              padding: '1px 4px',
              borderRadius: '6px',
              background: 'var(--tab-track-bg)',
              color: 'var(--text)',
              border: '1px solid var(--glass-divider)',
              outline: 'none',
            }}
          >
            {['30m', '1h', '6h', '12h', '1d', '3d', '7d'].map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </select>
          <span>
            {task.last_researched_at
              ? '\u00B7 Last: ' +
                new Date(task.last_researched_at).toLocaleString()
              : '\u00B7 Not yet researched'}
          </span>
        </div>
        <div
          style={{
            color: 'var(--hint)',
            fontSize: '11px',
            marginTop: '2px',
          }}
        >
          Created: {new Date(task.created_at).toLocaleString()}
          {task.completed_at &&
            ' \u00B7 Completed: ' +
              new Date(task.completed_at).toLocaleString()}
        </div>
        <div style={{ marginTop: '10px', display: 'flex', gap: '8px' }}>
          {task.focused ? (
            <button
              class="worktree-btn dispose"
              style={{
                background: 'rgba(168,85,247,0.15)',
                color: '#a855f7',
                borderColor: '#a855f7',
              }}
              onClick={() => handleFocus(false)}
            >
              Forget
            </button>
          ) : (
            <button
              class="worktree-btn merge"
              style={{
                background: 'rgba(168,85,247,0.15)',
                color: '#a855f7',
                borderColor: '#a855f7',
              }}
              onClick={() => handleFocus(true)}
            >
              Recall
            </button>
          )}
          {task.status === 'pending' && (
            <button
              class="worktree-btn merge"
              onClick={() => handleAction('activate')}
            >
              Activate
            </button>
          )}
          {task.status === 'active' && (
            <button
              class="worktree-btn merge"
              style={{
                background: 'rgba(34,197,94,0.15)',
                color: '#22c55e',
                borderColor: '#22c55e',
              }}
              onClick={() => handleAction('complete')}
            >
              Complete
            </button>
          )}
          {canCancel && (
            <button
              class="worktree-btn dispose"
              onClick={() => handleAction('cancel')}
            >
              Cancel
            </button>
          )}
          {canReopen && (
            <button
              class="worktree-btn merge"
              onClick={() => handleAction('reopen')}
            >
              Reopen
            </button>
          )}
        </div>
      </div>

      <div class="card-title" style={{ marginTop: '12px' }}>
        Documents ({task.documents.length})
      </div>
      {task.documents.length === 0 ? (
        <div class="empty-state" style={{ padding: '24px' }}>
          No documents yet.
        </div>
      ) : (
        task.documents.map((d: any) => (
          <DocCard key={d.id} doc={d} taskId={taskId} />
        ))
      )}
    </>
  );
}

function DocCard({ doc, taskId }: { doc: any; taskId: string }) {
  const [expanded, setExpanded] = useState(false);
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const toggle = async () => {
    if (expanded) {
      setExpanded(false);
      return;
    }
    setExpanded(true);
    if (content !== null) return;
    setLoading(true);
    try {
      const data = await apiFetch(
        '/miniapp/api/research/' + taskId + '/doc/' + doc.id,
      );
      setContent(data.content);
    } catch {
      setContent('Failed to load document.');
    }
    setLoading(false);
  };

  return (
    <div
      class="card glass"
      style={{ padding: '12px', cursor: 'pointer' }}
      onClick={toggle}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <span
          style={{
            color: 'var(--hint)',
            fontFamily: 'monospace',
            fontSize: '12px',
          }}
        >
          #{doc.seq}
        </span>
        <span style={{ fontWeight: 600, fontSize: '14px', flex: 1 }}>
          {doc.title}
        </span>
        <span
          style={{
            fontSize: '10px',
            padding: '2px 6px',
            borderRadius: '8px',
            background: 'var(--tab-track-bg)',
            color: 'var(--hint)',
          }}
        >
          {doc.doc_type}
        </span>
      </div>
      {doc.summary && (
        <div
          style={{
            color: 'var(--hint)',
            fontSize: '12px',
            marginTop: '4px',
          }}
        >
          {doc.summary}
        </div>
      )}
      {expanded && (
        <div
          style={{
            marginTop: '8px',
            borderTop: '1px solid var(--glass-divider)',
            paddingTop: '8px',
          }}
        >
          {loading ? (
            <div class="loading" style={{ padding: '12px' }}>
              Loading...
            </div>
          ) : content ? (
            <div
              class="md-rendered"
              style={{ maxHeight: '50vh', overflow: 'auto', padding: '8px 0' }}
              dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
            />
          ) : null}
        </div>
      )}
    </div>
  );
}
