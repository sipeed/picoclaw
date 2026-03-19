import { useEffect, useState, useCallback } from 'preact/hooks';
import { apiFetch } from '../../hooks/use-api';

interface CacheEntry {
  hash: string;
  type: string;
  result: string;
  file_path?: string;
  pages?: number;
  created_at: string;
  accessed_at: string;
}

const TYPE_LABELS: Record<string, string> = {
  pdf_ocr: 'PDF OCR',
  pdf_text: 'PDF Text',
  image_desc: 'Image',
};

const TYPE_COLORS: Record<string, { bg: string; text: string }> = {
  pdf_ocr: { bg: 'rgba(234,179,8,0.15)', text: '#ca8a04' },
  pdf_text: { bg: 'rgba(59,130,246,0.15)', text: '#2563eb' },
  image_desc: { bg: 'rgba(168,85,247,0.15)', text: '#a855f7' },
};

interface CacheSectionProps {
  active: boolean;
}

export function CacheSection({ active }: CacheSectionProps) {
  const [entries, setEntries] = useState<CacheEntry[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState('');
  const [expanded, setExpanded] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const url = filter
        ? '/miniapp/api/cache?type=' + encodeURIComponent(filter)
        : '/miniapp/api/cache';
      const data = await apiFetch<CacheEntry[]>(url);
      setEntries(data);
    } catch {
      setEntries(null);
    }
    setLoading(false);
  }, [filter]);

  useEffect(() => {
    if (active) load();
  }, [active, filter]);

  const formatDate = (iso: string) => {
    try {
      const d = new Date(iso);
      return d.toLocaleDateString() + ' ' + d.toLocaleTimeString().slice(0, 5);
    } catch {
      return iso;
    }
  };

  return (
    <div class="card glass">
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '8px',
        }}
      >
        <span class="card-title" style={{ margin: 0 }}>
          Media Cache
        </span>
        <button
          class="send-btn"
          style={{ padding: '4px 12px', fontSize: '12px' }}
          onClick={load}
        >
          Refresh
        </button>
      </div>

      <div class="log-filter-chips" style={{ marginBottom: '10px' }}>
        {[
          { label: 'All', value: '' },
          { label: 'PDF OCR', value: 'pdf_ocr' },
          { label: 'PDF Text', value: 'pdf_text' },
          { label: 'Image', value: 'image_desc' },
        ].map((f) => (
          <button
            key={f.value}
            class={`log-filter-chip${filter === f.value ? ' active' : ''}`}
            onClick={() => setFilter(f.value)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {loading && !entries ? (
        <div class="loading" style={{ padding: '12px' }}>
          Loading cache...
        </div>
      ) : !entries || entries.length === 0 ? (
        <div class="empty-state" style={{ padding: '24px 0' }}>
          No cached items.
        </div>
      ) : (
        entries.map((e) => {
          const tc = TYPE_COLORS[e.type] || TYPE_COLORS.image_desc;
          const isExpanded = expanded === e.hash + ':' + e.type;
          const preview =
            e.result.length > 80
              ? e.result.substring(0, 80) + '...'
              : e.result;

          return (
            <div
              key={e.hash + ':' + e.type}
              style={{
                padding: '10px 0',
                borderBottom: '1px solid var(--glass-divider)',
                cursor: 'pointer',
              }}
              onClick={() =>
                setExpanded(isExpanded ? null : e.hash + ':' + e.type)
              }
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                }}
              >
                <span
                  style={{
                    fontSize: '10px',
                    fontWeight: 600,
                    padding: '2px 6px',
                    borderRadius: '8px',
                    background: tc.bg,
                    color: tc.text,
                    flexShrink: 0,
                  }}
                >
                  {TYPE_LABELS[e.type] || e.type}
                </span>
                <span
                  style={{
                    fontSize: '13px',
                    flex: 1,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {preview || '(empty)'}
                </span>
                {e.pages ? (
                  <span
                    style={{
                      fontSize: '11px',
                      color: 'var(--hint)',
                      flexShrink: 0,
                    }}
                  >
                    {e.pages}p
                  </span>
                ) : null}
              </div>

              {isExpanded && (
                <div
                  style={{
                    marginTop: '8px',
                    fontSize: '12px',
                    color: 'var(--hint)',
                  }}
                >
                  <div style={{ marginBottom: '4px' }}>
                    <span style={{ fontWeight: 600 }}>Hash:</span>{' '}
                    <code style={{ fontSize: '11px' }}>{e.hash}</code>
                  </div>
                  {e.file_path && (
                    <div style={{ marginBottom: '4px' }}>
                      <span style={{ fontWeight: 600 }}>File:</span>{' '}
                      <code
                        style={{
                          fontSize: '11px',
                          wordBreak: 'break-all',
                        }}
                      >
                        {e.file_path}
                      </code>
                    </div>
                  )}
                  <div style={{ marginBottom: '4px' }}>
                    <span style={{ fontWeight: 600 }}>Created:</span>{' '}
                    {formatDate(e.created_at)}
                  </div>
                  <div style={{ marginBottom: '4px' }}>
                    <span style={{ fontWeight: 600 }}>Accessed:</span>{' '}
                    {formatDate(e.accessed_at)}
                  </div>
                  {e.result && (
                    <pre
                      style={{
                        marginTop: '6px',
                        fontSize: '11px',
                        maxHeight: '200px',
                        overflow: 'auto',
                        background: 'var(--secondary-bg)',
                        padding: '8px',
                        borderRadius: '6px',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        color: 'var(--text)',
                      }}
                    >
                      {e.result}
                    </pre>
                  )}
                </div>
              )}
            </div>
          );
        })
      )}
    </div>
  );
}
