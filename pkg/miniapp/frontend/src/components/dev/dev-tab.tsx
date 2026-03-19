import { useEffect, useState, useCallback } from 'preact/hooks';
import type { SSEHook } from '../../hooks/use-sse';
import { apiFetch, apiPost } from '../../hooks/use-api';
import { isFresh } from '../../utils';

interface DevTabProps {
  active: boolean;
  sse: SSEHook;
}

export function DevTab({ active, sse }: DevTabProps) {
  const [data, setData] = useState<any>(null);

  const loadDev = useCallback(async () => {
    try {
      const d = await apiFetch('/miniapp/api/dev');
      setData(d);
    } catch {}
  }, []);

  useEffect(() => {
    if (sse.dev) setData(sse.dev);
  }, [sse.dev]);

  useEffect(() => {
    if (active && !isFresh(sse.lastUpdate, 'dev')) loadDev();
  }, [active]);

  const targets = data?.targets || [];
  const activeId = data?.active_id || '';
  const isActive = !!data?.active;

  const handleToggle = async (id: string) => {
    const action = id === activeId ? 'deactivate' : 'activate';
    const body =
      action === 'activate'
        ? { action: 'activate', id }
        : { action: 'deactivate' };
    try {
      const d = await apiPost('/miniapp/api/dev', body);
      if (!d.error) setData(d);
    } catch {}
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm('Remove "' + name + '"?')) return;
    try {
      const d = await apiPost('/miniapp/api/dev', {
        action: 'unregister',
        id,
      });
      if (!d.error) setData(d);
    } catch {}
  };

  const iframeSrc = isActive ? location.origin + '/miniapp/dev/' : '';
  const targetDisplay = data?.target
    ? data.target.replace(/^https?:\/\//, '')
    : '';

  return (
    <>
      <div class="dev-header">
        <span class={`dev-target-dot${isActive ? ' on' : ''}`} />
        <span class="dev-header-title">Dev Preview</span>
        <span class="dev-header-target">{targetDisplay}</span>
      </div>

      {targets.length === 0 ? (
        <div class="empty-state">
          No targets registered.
          <br />
          Ask the agent to start a dev server.
        </div>
      ) : (
        targets.map((t: any) => {
          const isTargetActive = t.id === activeId;
          const displayUrl = t.target.replace(/^https?:\/\//, '');
          return (
            <div
              key={t.id}
              class={`dev-target-item glass glass-interactive${isTargetActive ? ' active' : ''}`}
              onClick={() => handleToggle(t.id)}
            >
              <span
                class={`dev-target-dot${isTargetActive ? ' on' : ''}`}
              />
              <span class="dev-target-name">{t.name}</span>
              <span class="dev-target-url">{displayUrl}</span>
              <span
                class="dev-target-delete"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(t.id, t.name);
                }}
              >
                {'\u00D7'}
              </span>
            </div>
          );
        })
      )}

      {isActive && (
        <div style={{ marginTop: '8px' }}>
          <div class="card glass" style={{ padding: 0, overflow: 'hidden' }}>
            <iframe
              src={iframeSrc}
              style={{
                width: '100%',
                height: '70vh',
                border: 'none',
                borderRadius: '16px',
              }}
            />
          </div>
        </div>
      )}
    </>
  );
}
