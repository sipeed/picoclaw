import { useEffect, useState, useCallback } from 'preact/hooks';
import type { SSEHook } from '../../hooks/use-sse';
import { apiFetch } from '../../hooks/use-api';
import { isFresh } from '../../utils';
import { GitSection } from './git-section';
import { SessionSection } from './session-section';

interface WorkTabProps {
  active: boolean;
  sse: SSEHook;
}

export function WorkTab({ active, sse }: WorkTabProps) {
  const [gitRepos, setGitRepos] = useState<any[] | null>(null);
  const [worktrees, setWorktrees] = useState<any[]>([]);
  const [sessions, setSessions] = useState<any[] | null>(null);
  const [stats, setStats] = useState<any>(null);
  const [graph, setGraph] = useState<any>(null);
  const [context, setContext] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [gitData, wtData, sessionData, sessionsData, ctxData, graphData] =
        await Promise.all([
          apiFetch('/miniapp/api/git'),
          apiFetch('/miniapp/api/worktrees').catch(() => []),
          apiFetch('/miniapp/api/session'),
          apiFetch('/miniapp/api/sessions').catch(() => []),
          apiFetch('/miniapp/api/context').catch(() => null),
          apiFetch('/miniapp/api/sessions/graph').catch(() => null),
        ]);
      setGitRepos(gitData);
      setWorktrees(wtData);
      setStats(sessionData);
      setSessions(sessionsData);
      setContext(ctxData);
      setGraph(graphData);
    } catch {}
    setLoading(false);
  }, []);

  // SSE session updates
  useEffect(() => {
    if (sse.session) {
      setSessions(sse.session.sessions || []);
      setStats(sse.session.stats || null);
      if (sse.session.graph) setGraph(sse.session.graph);
    }
  }, [sse.session]);

  useEffect(() => {
    if (sse.context) setContext(sse.context);
  }, [sse.context]);

  useEffect(() => {
    if (active) loadAll();
  }, [active]);

  if (loading && !gitRepos && !sessions)
    return <div class="loading">Loading...</div>;

  return (
    <>
      <GitSection repos={gitRepos} worktrees={worktrees} onReload={loadAll} />
      <SessionSection
        sessions={sessions}
        stats={stats}
        graph={graph}
        context={context}
      />
    </>
  );
}
