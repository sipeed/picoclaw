export function escapeHtml(s: string | null | undefined): string {
  if (!s) return '';
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

export function escapeAttr(s: string | null | undefined): string {
  if (!s) return '';
  return s
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

export function formatAge(sec: number): string {
  if (sec < 60) return sec + 's ago';
  if (sec < 3600) return Math.floor(sec / 60) + 'm ago';
  return Math.floor(sec / 3600) + 'h ago';
}

export function formatTokens(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
  return String(n);
}

export function flashSent(el: HTMLElement) {
  el.classList.add('sent');
  setTimeout(() => el.classList.remove('sent'), 600);
}

export function formatSessionLabel(key: string): string {
  if (key.startsWith('heartbeat:')) return 'Heartbeat';
  const parts = key.split(':');
  if (parts.length >= 4) {
    const channel = parts[2]; // "telegram"
    const scope = parts[3]; // "group" or "dm"
    return capitalize(channel) + ' ' + capitalize(scope);
  }
  if (parts.length > 2) return parts.slice(2).join(':');
  return key;
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

export function isFresh(lastUpdate: Record<string, number>, key: string, ms = 5000): boolean {
  return !!lastUpdate[key] && Date.now() - lastUpdate[key] < ms;
}
