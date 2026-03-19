import { useEffect, useRef, useState, useCallback } from 'preact/hooks';
import { renderLogs as renderLogsView } from '../../logs_view.js';

const LOGS_PAGE_SIZE = 60;

interface LogsSectionProps {
  active: boolean;
}

export function LogsSection({ active }: LogsSectionProps) {
  const [entries, setEntries] = useState<any[]>([]);
  const [component, setComponent] = useState('');
  const [page, setPage] = useState(1);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef<any>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const connect = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState <= 1) return;
    const initData = window.Telegram?.WebApp?.initData || '';
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    let url =
      proto +
      '//' +
      location.host +
      '/miniapp/api/logs/ws?initData=' +
      encodeURIComponent(initData);
    if (component) url += '&component=' + encodeURIComponent(component);

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);
    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      if (msg.type === 'init') {
        setEntries(msg.entries || []);
        setPage(1);
      } else if (msg.type === 'entry') {
        setEntries((prev) => {
          const next = [...prev, msg.entry];
          return next.length > 200 ? next.slice(1) : next;
        });
      }
    };
    ws.onclose = () => {
      setConnected(false);
      wsRef.current = null;
      if (active) {
        reconnectRef.current = setTimeout(connect, 3000);
      }
    };
    ws.onerror = () => {};
  }, [component, active]);

  const disconnect = useCallback(() => {
    if (reconnectRef.current) {
      clearTimeout(reconnectRef.current);
      reconnectRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setConnected(false);
  }, []);

  useEffect(() => {
    if (active) {
      connect();
    } else {
      disconnect();
    }
    return disconnect;
  }, [active, component]);

  const handleFilterChange = (comp: string) => {
    setComponent(comp);
    setPage(1);
    setEntries([]);
    disconnect();
    // Will reconnect via useEffect when component changes
  };

  const view = renderLogsView(entries, {
    component,
    page,
    pageSize: LOGS_PAGE_SIZE,
  });

  const handleSaveSnapshot = async () => {
    try {
      const initData = window.Telegram?.WebApp?.initData || '';
      const res = await fetch(
        location.origin +
          '/miniapp/api/logs/snapshot?initData=' +
          encodeURIComponent(initData),
        { method: 'POST' },
      );
      if (!res.ok) return;
      const data = await res.json();
      if (data.download_url) {
        const a = document.createElement('a');
        a.href =
          location.origin +
          data.download_url +
          '?initData=' +
          encodeURIComponent(initData);
        a.download = '';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
      }
    } catch {}
  };

  return (
    <div class="card glass">
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          marginBottom: '8px',
        }}
      >
        <span class="card-title" style={{ margin: 0 }}>
          Logs
        </span>
        <span class={`dev-target-dot${connected ? ' on' : ''}`} />
      </div>
      <div class="log-filter-chips">
        {[
          { label: 'All', comp: '' },
          { label: 'Telego', comp: 'telego' },
          { label: 'Console', comp: 'dev-console' },
        ].map((f) => (
          <button
            key={f.comp}
            class={`log-filter-chip${component === f.comp ? ' active' : ''}`}
            onClick={() => handleFilterChange(f.comp)}
          >
            {f.label}
          </button>
        ))}
      </div>
      <div
        ref={containerRef}
        id="logs-content"
        dangerouslySetInnerHTML={{ __html: view.html || '<div class="empty-state">No logs.</div>' }}
      />
      <div class="log-pagination">
        <button
          class="log-page-btn"
          disabled={view.currentPage <= 1}
          onClick={() => setPage((p) => Math.max(1, p - 1))}
        >
          Newer
        </button>
        <span>
          {view.currentPage}/{view.totalPages} ({view.totalItems})
        </span>
        <button
          class="log-page-btn"
          disabled={view.currentPage >= view.totalPages}
          onClick={() => setPage((p) => p + 1)}
        >
          Older
        </button>
      </div>
      <div class="log-actions">
        <button class="log-snap-btn" onClick={handleSaveSnapshot}>
          Save Snapshot
        </button>
      </div>
    </div>
  );
}
