import { describe, expect, it } from 'vitest';

import { escapeHtml, filterLogs, paginateLogs, renderFields, renderLogs, renderLogsInto } from './logs_view.js';

describe('logs_view', () => {
  it('renders a normal log message row', () => {
    const view = renderLogs([
      {
        timestamp: '2026-03-05T12:34:56Z',
        level: 'INFO',
        component: 'telego',
        message: 'connected',
      },
    ]);

    expect(view.totalItems).toBe(1);
    expect(view.html).toContain('12:34:56');
    expect(view.html).toContain('log-badge info');
    expect(view.html).toContain('connected');
    expect(view.html).toContain('telego');
  });

  it('renders fields for empty/single/multiple cases', () => {
    expect(renderFields(null)).toBe('');
    expect(renderFields({})).toBe('');
    expect(renderFields({ req_id: 42 })).toContain('{req_id=42}');

    const multi = renderFields({ a: 'x', b: 2 });
    expect(multi).toContain('a=x');
    expect(multi).toContain('b=2');
  });

  it('sanitizes potentially dangerous HTML', () => {
    const xss = '<img src=x onerror=alert(1) />';
    const escaped = escapeHtml(xss);
    expect(escaped).not.toContain('<img');
    expect(escaped).toContain('&lt;img');

    const view = renderLogs([{ message: xss, level: 'warn' }]);
    const container = document.createElement('div');
    renderLogsInto(container, [{ message: xss, level: 'warn' }]);
    expect(container.querySelector('img')).toBeNull();
    expect(view.html).toContain('&lt;img');
  });

  it('supports filtering and pagination', () => {
    const entries = [];
    for (let i = 0; i < 25; i++) {
      entries.push({
        level: 'debug',
        component: i % 2 === 0 ? 'telego' : 'dev-console',
        message: `entry-${i}`,
      });
    }

    const filtered = filterLogs(entries, 'telego');
    expect(filtered.length).toBe(13);

    const pageInfo = paginateLogs(filtered, 2, 5);
    expect(pageInfo.totalPages).toBe(3);
    expect(pageInfo.items.length).toBe(5);

    const page1 = renderLogs(entries, { component: 'telego', page: 1, pageSize: 5 });
    const page2 = renderLogs(entries, { component: 'telego', page: 2, pageSize: 5 });
    expect(page1.totalPages).toBe(3);
    expect(page1.html).not.toBe(page2.html);
    expect(page1.html).toContain('entry-24');
    expect(page2.html).toContain('entry-14');
  });
});
