import { useEffect, useRef, useState } from 'preact/hooks';

export interface SSEData {
  plan: any | null;
  session: any | null;
  skills: any[] | null;
  dev: any | null;
  context: any | null;
  prompt: string | null;
}

export interface SSEHook extends SSEData {
  lastUpdate: Record<string, number>;
}

export function useSSE(): SSEHook {
  const [data, setData] = useState<SSEData>({
    plan: null,
    session: null,
    skills: null,
    dev: null,
    context: null,
    prompt: null,
  });
  const lastUpdate = useRef<Record<string, number>>({});

  useEffect(() => {
    const initData = window.Telegram?.WebApp?.initData || '';
    const url =
      location.origin +
      '/miniapp/api/events?initData=' +
      encodeURIComponent(initData);
    const es = new EventSource(url);

    es.addEventListener('plan', (e: any) => {
      try {
        const d = JSON.parse(e.data);
        lastUpdate.current.plan = Date.now();
        setData((prev) => ({ ...prev, plan: d }));
      } catch {}
    });

    es.addEventListener('session', (e: any) => {
      try {
        const d = JSON.parse(e.data);
        lastUpdate.current.session = Date.now();
        setData((prev) => ({ ...prev, session: d }));
      } catch {}
    });

    es.addEventListener('skills', (e: any) => {
      try {
        const d = JSON.parse(e.data);
        lastUpdate.current.skills = Date.now();
        setData((prev) => ({ ...prev, skills: d }));
      } catch {}
    });

    es.addEventListener('dev', (e: any) => {
      try {
        const d = JSON.parse(e.data);
        lastUpdate.current.dev = Date.now();
        setData((prev) => ({ ...prev, dev: d }));
      } catch {}
    });

    es.addEventListener('context', (e: any) => {
      try {
        const d = JSON.parse(e.data);
        setData((prev) => ({ ...prev, context: d }));
      } catch {}
    });

    es.addEventListener('prompt', (e: any) => {
      try {
        const d = JSON.parse(e.data);
        setData((prev) => ({ ...prev, prompt: d.prompt || null }));
      } catch {}
    });

    return () => es.close();
  }, []);

  return { ...data, lastUpdate: lastUpdate.current };
}
