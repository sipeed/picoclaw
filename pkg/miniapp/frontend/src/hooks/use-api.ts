const API_BASE = location.origin;

function getInitData(): string {
  return window.Telegram?.WebApp?.initData || '';
}

export async function apiFetch<T = any>(path: string): Promise<T> {
  const sep = path.includes('?') ? '&' : '?';
  const res = await fetch(
    API_BASE + path + sep + 'initData=' + encodeURIComponent(getInitData()),
  );
  if (!res.ok) throw new Error('API error: ' + res.status);
  return res.json();
}

export async function apiPost<T = any>(
  path: string,
  body: Record<string, any>,
): Promise<T> {
  const sep = path.includes('?') ? '&' : '?';
  const res = await fetch(
    API_BASE + path + sep + 'initData=' + encodeURIComponent(getInitData()),
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  );
  if (!res.ok) {
    let errMsg = 'API error: ' + res.status;
    try {
      const data = await res.json();
      if (data.error) errMsg = data.error;
    } catch {}
    throw new Error(errMsg);
  }
  return res.json();
}

export async function apiDelete<T = any>(path: string): Promise<T> {
  const sep = path.includes('?') ? '&' : '?';
  const res = await fetch(
    API_BASE + path + sep + 'initData=' + encodeURIComponent(getInitData()),
    { method: 'DELETE' },
  );
  if (!res.ok) throw new Error('API error: ' + res.status);
  return res.json();
}

export async function sendCommand(cmd: string): Promise<boolean> {
  if (!cmd.startsWith('/')) return false;
  try {
    await apiPost('/miniapp/api/command', { command: cmd });
    return true;
  } catch {
    return false;
  }
}
