import { useState, useCallback } from 'preact/hooks';
import { apiFetch, apiPost } from '../../hooks/use-api';
import { escapeHtml } from '../../utils';

interface GitSectionProps {
  repos: any[] | null;
  worktrees: any[];
  onReload: () => void;
}

export function GitSection({ repos, worktrees, onReload }: GitSectionProps) {
  const [detailRepo, setDetailRepo] = useState<any>(null);
  const [detailName, setDetailName] = useState<string | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);

  const loadDetail = useCallback(async (name: string) => {
    setDetailName(name);
    setLoadingDetail(true);
    try {
      const data = await apiFetch(
        '/miniapp/api/git?repo=' + encodeURIComponent(name),
      );
      setDetailRepo(data);
    } catch {}
    setLoadingDetail(false);
  }, []);

  const goBack = useCallback(() => {
    setDetailRepo(null);
    setDetailName(null);
    onReload();
  }, [onReload]);

  if (detailName) {
    return (
      <RepoDetail
        repo={detailRepo}
        name={detailName}
        loading={loadingDetail}
        onBack={goBack}
      />
    );
  }

  return (
    <>
      <Worktrees items={worktrees} onReload={onReload} />
      <RepoList repos={repos} onSelect={loadDetail} />
    </>
  );
}

function Worktrees({
  items,
  onReload,
}: {
  items: any[];
  onReload: () => void;
}) {
  const [busy, setBusy] = useState<string | null>(null);

  const handleAction = async (
    action: string,
    name: string,
    isDirty: boolean,
  ) => {
    let force = false;
    if (action === 'merge') {
      if (!confirm('Merge "' + name + '" into base branch?')) return;
    } else if (action === 'dispose') {
      if (isDirty) {
        if (
          !confirm(
            '"' +
              name +
              '" has uncommitted changes. Force dispose and auto-commit before removal?',
          )
        )
          return;
        force = true;
      } else if (!confirm('Dispose worktree "' + name + '"?')) {
        return;
      }
    }

    setBusy(name + ':' + action);
    try {
      await apiPost('/miniapp/api/worktrees', { action, name, force });
      onReload();
    } catch (err: any) {
      alert(err.message || 'Action failed');
    }
    setBusy(null);
  };

  return (
    <div class="card glass">
      <div class="card-title">Worktrees</div>
      {items.length === 0 ? (
        <div class="empty-state" style={{ padding: '12px 0 4px' }}>
          No active worktrees.
        </div>
      ) : (
        <div class="worktree-list">
          {items.map((wt) => {
            let last = '(no commits)';
            if (wt.last_commit_hash) {
              last =
                wt.last_commit_hash +
                ' ' +
                (wt.last_commit_subject || '');
              if (wt.last_commit_age) last += ' (' + wt.last_commit_age + ')';
            }

            return (
              <div
                class={`worktree-item${wt.has_uncommitted ? ' dirty' : ''}`}
                key={wt.name}
              >
                <div class="worktree-main">
                  <div class="worktree-name-row">
                    <span class="worktree-name">{wt.name}</span>
                    {wt.has_uncommitted ? (
                      <span class="worktree-dirty">DIRTY</span>
                    ) : (
                      <span class="worktree-clean">CLEAN</span>
                    )}
                  </div>
                  <div class="worktree-branch">{wt.branch || '?'}</div>
                  <div class="worktree-last">{last}</div>
                </div>
                <div class="worktree-actions">
                  <button
                    class="worktree-btn merge"
                    disabled={busy === wt.name + ':merge'}
                    onClick={() =>
                      handleAction('merge', wt.name, wt.has_uncommitted)
                    }
                  >
                    {busy === wt.name + ':merge' ? 'Merging...' : 'Merge'}
                  </button>
                  <button
                    class="worktree-btn dispose"
                    disabled={busy === wt.name + ':dispose'}
                    onClick={() =>
                      handleAction('dispose', wt.name, wt.has_uncommitted)
                    }
                  >
                    {busy === wt.name + ':dispose'
                      ? 'Disposing...'
                      : 'Dispose'}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function RepoList({
  repos,
  onSelect,
}: {
  repos: any[] | null;
  onSelect: (name: string) => void;
}) {
  if (!repos || repos.length === 0) {
    return (
      <div class="empty-state" style={{ marginTop: '12px' }}>
        No git repositories found.
      </div>
    );
  }

  return (
    <>
      <div
        style={{
          padding: '10px 4px 8px',
          fontSize: '12px',
          color: 'var(--hint)',
        }}
      >
        Repositories
      </div>
      {repos.map((r) => (
        <div
          key={r.name}
          class="git-repo-item glass glass-interactive"
          onClick={() => onSelect(r.name)}
        >
          <div class="git-repo-body">
            <div class="git-repo-name">{r.name}</div>
            <div class="git-repo-branch">{r.branch || '?'}</div>
          </div>
          <span class="git-repo-arrow">{'\u203A'}</span>
        </div>
      ))}
    </>
  );
}

function RepoDetail({
  repo,
  name,
  loading,
  onBack,
}: {
  repo: any;
  name: string;
  loading: boolean;
  onBack: () => void;
}) {
  if (loading || !repo) {
    return (
      <>
        <button class="git-back-btn" onClick={onBack}>
          {'\u2190'} {name}
        </button>
        <div class="loading">Loading {name}...</div>
      </>
    );
  }

  return (
    <>
      <button class="git-back-btn" onClick={onBack}>
        {'\u2190'} {repo.name || name}
      </button>
      <div class="card glass">
        <div class="card-title">
          {repo.name} &mdash; {repo.branch || '?'}
        </div>

        {repo.modified && repo.modified.length > 0 && (
          <>
            <div
              style={{
                padding: '4px 12px 8px',
                fontSize: '12px',
                color: 'var(--hint)',
              }}
            >
              Changes ({repo.modified.length})
            </div>
            {repo.modified.map((f: any, i: number) => (
              <div class="git-commit" key={i}>
                <span
                  class={`git-status git-status-${f.status === '??' ? 'u' : f.status.toLowerCase()}`}
                >
                  {f.status}
                </span>
                <span class="git-subject">{f.path}</span>
              </div>
            ))}
          </>
        )}

        {repo.commits && repo.commits.length > 0 ? (
          <>
            <div
              style={{
                padding: '4px 12px 8px',
                fontSize: '12px',
                color: 'var(--hint)',
              }}
            >
              Commits
            </div>
            {repo.commits.map((c: any, i: number) => (
              <div class="git-commit" key={i}>
                <span class="git-hash">{c.hash}</span>
                <span class="git-subject">{c.subject}</span>
                <span class="git-meta">{c.date}</span>
              </div>
            ))}
          </>
        ) : (
          <div style={{ padding: '12px', color: 'var(--hint)' }}>
            No commits found.
          </div>
        )}
      </div>
    </>
  );
}
