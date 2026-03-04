const SAFE_LEVELS = new Set(['debug', 'info', 'warn', 'error']);

function stringifyFieldValue(value) {
  if (value === null || value === undefined) {
    return '';
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

export function escapeHtml(value) {
  return String(value == null ? '' : value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

export function renderFields(fields) {
  if (!fields || typeof fields !== 'object' || Array.isArray(fields)) {
    return '';
  }
  const keys = Object.keys(fields);
  if (keys.length === 0) {
    return '';
  }

  const parts = keys.map((key) => `${key}=${stringifyFieldValue(fields[key])}`);
  return ` <span class="log-fields">{${escapeHtml(parts.join(', '))}}</span>`;
}

export function filterLogs(entries, component = '') {
  if (!Array.isArray(entries) || entries.length === 0) {
    return [];
  }
  if (!component) {
    return entries.slice();
  }
  return entries.filter((entry) => (entry?.component || '') === component);
}

export function paginateLogs(entries, page = 1, pageSize = 100) {
  const list = Array.isArray(entries) ? entries : [];
  const size = Math.max(1, Number(pageSize) || 100);
  const totalPages = Math.max(1, Math.ceil(list.length / size));
  const currentPage = Math.min(Math.max(1, Number(page) || 1), totalPages);

  // Page 1 is the newest page (tail of the list).
  const end = list.length - (currentPage - 1) * size;
  const start = Math.max(0, end - size);
  return {
    items: list.slice(start, Math.max(start, end)),
    currentPage,
    totalPages,
    pageSize: size,
  };
}

export function renderLogs(entries, options = {}) {
  const component = options.component || '';
  const filtered = filterLogs(entries, component);
  const paged = paginateLogs(filtered, options.page, options.pageSize);

  let html = '';
  for (const entry of paged.items) {
    const levelRaw = String(entry?.level || 'info').toLowerCase();
    const level = SAFE_LEVELS.has(levelRaw) ? levelRaw : 'info';
    const ts = entry?.timestamp ? String(entry.timestamp).substring(11, 19) : '';
    const componentHTML = entry?.component
      ? `<span class="log-comp">${escapeHtml(entry.component)}</span>`
      : '';
    const fieldsHTML = renderFields(entry?.fields);
    const message = escapeHtml(entry?.message || '');

    html += '<div class="log-entry">' +
      `<span class="log-ts">${ts}</span>` +
      `<span class="log-badge ${level}">${level}</span>` +
      componentHTML +
      `<span class="log-msg">${message}${fieldsHTML}</span>` +
      '</div>';
  }

  return {
    html,
    totalItems: filtered.length,
    currentPage: paged.currentPage,
    totalPages: paged.totalPages,
    pageSize: paged.pageSize,
  };
}

export function renderLogsInto(container, entries, options = {}) {
  const view = renderLogs(entries, options);
  container.innerHTML = view.html;
  return view;
}
