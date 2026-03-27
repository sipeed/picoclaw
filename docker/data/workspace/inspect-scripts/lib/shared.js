const { chromium } = require('playwright');
const fs = require('fs');
const pathLib = require('path');

const BASE_URL = 'https://dashboard.int3nt.info';
const LOGIN_EMAIL = 'heidi@intnt.ai';
const LOGIN_PASSWORD = 'testing2026!';
const ORG_NAME = 'Testing2026!';

const DESTRUCTIVE_RE = /delete|remove|confirm|save|submit|export|download|logout|sign\s*out|sign\s*up|sign\s*in|discard|back\s+to|login|log\s*in|send\s+email|set\s+password|reset|forgot|cancel/i;
const MODAL_TRIGGER_RE = /add|create|new|edit|invite|upload|import|schedule|filter|sort|column|show|manage|configure|change|select version|continue|next|proceed/i;
const MAX_MODALS = 10;
const CLICK_TIMEOUT_MS = 60000;
const FRAMEWORK_RE_SRC = '^(v-|mdi-|d-|ma-|pa-|mt-|mb-|ml-|mr-|mx-|my-|px-|py-|pt-|pb-|pl-|pr-|ga-|text-|font-|bg-|rounded|elevation|float-|position-|overflow-|align-|justify-|flex-|order-|col-|row-|container|spacer|fill-|w-|h-|min-|max-|gap-|border|opacity|cursor|transition|--v-)';

function stripDynamic(t) {
  return t.replace(/\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},\s+\d{4}/g, '{DATE}')
    .replace(/\{DATE\}\s*[-–]\s*\{DATE\}/g, '{DATE_RANGE}').replace(/\b\d{4}-\d{2}-\d{2}\b/g, '{DATE}').trim();
}

function slugify(name) {
  return name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
}

// ---------------------------------------------------------------------------
// DOM extraction — runs inside browser context.
// Every element gets its REAL className, id, and a computed best-selector.
// ---------------------------------------------------------------------------
async function extractAllComponents(page) {
  return await page.evaluate((fwReSrc) => {
    const FW_RE = new RegExp(fwReSrc);
    function isVis(el) {
      if (!el) return false;
      const r = el.getBoundingClientRect();
      if (r.width === 0 || r.height === 0) return false;
      const s = getComputedStyle(el);
      return s.display !== 'none' && s.visibility !== 'hidden' && parseFloat(s.opacity) > 0;
    }
    function txt(el) { return el?.textContent?.trim().replace(/\s+/g, ' ') || ''; }
    function aria(el) { return el?.getAttribute('aria-label') || null; }
    function tid(el) { return el?.getAttribute('data-testid') || el?.getAttribute('data-test') || null; }

    function customClasses(el) {
      const cls = el?.className?.toString?.() || '';
      return cls.split(/\s+/).filter(c => c.length > 2 && !FW_RE.test(c));
    }

    function bestSel(el) {
      const t = tid(el);
      if (t) return `[data-testid="${t}"]`;
      if (el.id && !/^input-v-|^v-/.test(el.id)) return `#${el.id}`;
      const cc = customClasses(el);
      if (cc.length) return `.${cc[0]}`;
      return null;
    }

    function elMeta(el) {
      return {
        className: el?.className?.toString?.() || '',
        id: el?.id || null,
        testId: tid(el),
        customClasses: customClasses(el),
        selector: bestSel(el),
      };
    }

    function inputLabel(input) {
      let l = null;
      const vf = input.closest('.v-field');
      if (vf) { const fl = vf.querySelector('.v-field__label'); if (fl) l = txt(fl); }
      if (!l) { const vi = input.closest('.v-input'); if (vi) { const vl = vi.querySelector('.v-label'); if (vl) l = txt(vl); } }
      if (!l && input.id) { const lbl = document.querySelector(`label[for="${input.id}"]`); if (lbl) l = txt(lbl); }
      if (!l) { const w = input.closest('label'); if (w) l = txt(w).replace(input.value || '', '').trim(); }
      if (!l) l = input.getAttribute('aria-label') || input.placeholder || null;
      if (!l) { const c = input.closest('.v-input') || input.parentElement; if (c) { const p = c.previousElementSibling; if (p && txt(p).length < 60) l = txt(p); } }
      return l ? l.replace(/\*/g, '').trim() : null;
    }

    function inputContainerSel(input) {
      const containers = [input.closest('.v-field'), input.closest('.v-input'), input.closest('.v-text-field')].filter(Boolean);
      for (const c of containers) {
        const s = bestSel(c);
        if (s && !s.startsWith('.v-')) return `${s} input`;
      }
      const label = inputLabel(input);
      if (label) {
        const ariaL = input.getAttribute('aria-label');
        if (ariaL) return `input[aria-label="${ariaL}"]`;
        const ph = input.placeholder;
        if (ph) return `input[placeholder="${ph}"]`;
      }
      return null;
    }

    const C = {
      headings: [], staticTexts: [], breadcrumbs: [],
      inputs: [], textareas: [], selects: [], checkboxes: [], switches: [], radios: [], btnToggles: [], fileInputs: [],
      buttons: [], iconButtons: [], links: [],
      tables: [], cards: [], chips: [], tabs: [], lists: [],
      alerts: [], snackbars: [], progressBars: [],
      dialogs: [], menus: [],
      expansionPanels: [], navDrawerItems: [], toolbarItems: [], pagination: [],
      images: [], avatarCount: 0, badges: [], hasDividers: false, tooltipTriggers: [],
      vueFlow: null, monacoEditors: [],
      elementRegistry: [],
    };

    // HEADINGS
    const seenH = new Set();
    document.querySelectorAll('h1,h2,h3,h4,h5,h6,.text-h1,.text-h2,.text-h3,.text-h4,.text-h5,.text-h6').forEach(el => {
      if (!isVis(el)) return; const t = txt(el);
      if (!t || t.length > 200 || seenH.has(t)) return; seenH.add(t);
      const tag = el.tagName.toLowerCase();
      C.headings.push({ tag: tag.match(/^h\d$/) ? tag : (Array.from(el.classList).find(c => /^text-h\d$/.test(c)) || ''), text: t, ...elMeta(el) });
    });

    // STATIC TEXTS
    const seenT = new Set();
    document.querySelectorAll('.v-card-text,.v-card-subtitle,.v-card-title,.v-list-subheader,p,.text-body-1,.text-body-2,.text-subtitle-1,.text-subtitle-2,.text-caption,.text-overline,.v-alert__content,.v-banner__text,span.text-medium-emphasis,.v-toolbar-title,.v-empty-state__headline,.v-empty-state__text').forEach(el => {
      if (!isVis(el)) return; const t = txt(el);
      if (!t || t.length < 3 || t.length > 500 || seenH.has(t) || seenT.has(t)) return; seenT.add(t);
      C.staticTexts.push({ type: Array.from(el.classList).find(c => c.startsWith('text-') || c.startsWith('v-')) || el.tagName.toLowerCase(), text: t, ...elMeta(el) });
    });

    // BREADCRUMBS
    document.querySelectorAll('.v-breadcrumbs-item').forEach(el => { if (isVis(el)) { const t = txt(el); if (t) C.breadcrumbs.push(t); } });

    // INPUTS
    document.querySelectorAll('input').forEach((inp, i) => {
      if (inp.type === 'hidden' || inp.type === 'file' || !isVis(inp)) return;
      C.inputs.push({
        index: i, type: inp.type || 'text', label: inputLabel(inp),
        placeholder: inp.placeholder || null, ariaLabel: aria(inp),
        disabled: inp.disabled, readOnly: inp.readOnly,
        inSelect: !!inp.closest('.v-select,.v-autocomplete,.v-combobox'),
        ...elMeta(inp),
        containerSelector: inputContainerSel(inp),
      });
    });

    // TEXTAREAS
    document.querySelectorAll('textarea').forEach((ta, i) => {
      if (!isVis(ta)) return;
      C.textareas.push({ index: i, label: inputLabel(ta), placeholder: ta.placeholder || null, ariaLabel: aria(ta), rows: ta.rows, disabled: ta.disabled, ...elMeta(ta), containerSelector: inputContainerSel(ta) });
    });

    // FILE INPUTS
    document.querySelectorAll('input[type="file"]').forEach((fi, i) => {
      C.fileInputs.push({ index: i, label: inputLabel(fi) || aria(fi), accept: fi.accept || null, multiple: fi.multiple, ...elMeta(fi) });
    });

    // SELECTS / DROPDOWNS
    document.querySelectorAll('.v-select,.v-autocomplete,.v-combobox').forEach((sel, i) => {
      if (!isVis(sel)) return;
      const vi = sel.closest('.v-input');
      const label = (() => {
        const own = vi?.querySelector('.v-label,.v-field__label,label')?.textContent?.trim();
        if (own) return own;
        // Common app structure: <label>...</label> followed by .v-input wrapper.
        const wrapper = vi?.parentElement || sel.parentElement;
        const sibling = wrapper?.querySelector(':scope > label')?.textContent?.trim();
        if (sibling) return sibling;
        // Fallback: nearest ancestor label in small radius.
        let cur = wrapper;
        for (let d = 0; d < 3 && cur; d++) {
          const near = cur.querySelector(':scope > label')?.textContent?.trim();
          if (near) return near;
          cur = cur.parentElement;
        }
        return null;
      })();
      const value = sel.querySelector('.v-select__selection-text,.v-autocomplete__selection')?.textContent?.trim() || null;
      const type = sel.classList.contains('v-autocomplete') ? 'autocomplete' : sel.classList.contains('v-combobox') ? 'combobox' : 'select';
      C.selects.push({ index: i, type, label, currentValue: value, disabled: sel.classList.contains('v-input--disabled'), ...elMeta(sel), containerSelector: bestSel(vi) });
    });

    // CHECKBOXES
    document.querySelectorAll('.v-checkbox,input[type="checkbox"]').forEach((el, i) => {
      const w = el.closest('.v-checkbox') || el.closest('.v-selection-control');
      const inp = el.tagName === 'INPUT' ? el : el.querySelector('input[type="checkbox"]');
      if (!inp || !isVis(w || el)) return;
      const label = w?.querySelector('.v-label')?.textContent?.trim() || aria(inp);
      C.checkboxes.push({ index: i, label, checked: inp.checked, disabled: inp.disabled, ...elMeta(w || el) });
    });

    // SWITCHES
    document.querySelectorAll('.v-switch').forEach((sw, i) => {
      if (!isVis(sw)) return;
      const label = sw.querySelector('.v-label')?.textContent?.trim() || null;
      const inp = sw.querySelector('input');
      C.switches.push({ index: i, label, checked: inp?.checked || false, disabled: sw.classList.contains('v-input--disabled'), ...elMeta(sw) });
    });

    // RADIO GROUPS
    document.querySelectorAll('.v-radio-group').forEach((rg, i) => {
      if (!isVis(rg)) return;
      const gl = rg.querySelector(':scope > .v-label,:scope > .v-input__control > .v-label')?.textContent?.trim() || null;
      const opts = Array.from(rg.querySelectorAll('.v-radio .v-label,.v-selection-control .v-label')).map(l => l.textContent.trim()).filter(Boolean);
      const sel = rg.querySelector('.v-selection-control--checked .v-label')?.textContent?.trim() || null;
      if (opts.length) C.radios.push({ index: i, groupLabel: gl, options: opts, selected: sel, ...elMeta(rg) });
    });

    // BUTTON TOGGLES (e.g. SIMPLE / ADVANCED)
    document.querySelectorAll('.v-btn-toggle').forEach((toggle, i) => {
      if (!isVis(toggle)) return;
      const parent = toggle.closest('.v-input');
      const groupLabel = (() => {
        let cur = toggle.previousElementSibling || toggle.parentElement;
        for (let d = 0; d < 4 && cur; d++) {
          const t = cur?.textContent?.trim();
          if (t && t.length < 60 && t.length > 1) return t;
          cur = cur.previousElementSibling || cur.parentElement;
        }
        return null;
      })();
      const opts = Array.from(toggle.querySelectorAll('.v-btn')).map(b => ({
        text: (b.textContent?.trim() || '').replace(/\s+/g, ' '),
        active: b.classList.contains('v-btn--active') || b.classList.contains('v-btn--variant-flat'),
      })).filter(o => o.text);
      if (opts.length >= 2) C.btnToggles.push({ index: i, groupLabel, options: opts, ...elMeta(toggle) });
    });

    // BUTTONS
    const seenBtn = new Set();
    document.querySelectorAll('button,[role="button"]').forEach((btn, i) => {
      if (!isVis(btn)) return;
      // Expansion panel headers are rendered as <button>, but should be modeled
      // as panels (not generic action buttons) to avoid wrong test selectors.
      if (btn.matches('.v-expansion-panel-title') || btn.closest('.v-expansion-panel-title')) return;
      const t = txt(btn); const al = aria(btn);
      const hasIcon = !!btn.querySelector('.v-icon,.mdi,i[class*="mdi"]');
      if ((!t || !t.length) && hasIcon) {
        const ic = btn.querySelector('.v-icon,[class*="mdi-"]');
        const iconCls = ic ? Array.from(ic.classList).find(c => c.startsWith('mdi-')) || '' : '';
        C.iconButtons.push({ index: i, icon: iconCls, ariaLabel: al, title: btn.title || null, disabled: btn.disabled, ...elMeta(btn) });
      } else if (t && t.length > 0 && t.length < 200) {
        const key = t.replace(/\s+/g, ' ');
        const dup = seenBtn.has(key); seenBtn.add(key);
        C.buttons.push({ text: key, ariaLabel: al, disabled: btn.disabled, duplicate: dup, ...elMeta(btn) });
      }
    });

    // LINKS
    const seenL = new Set();
    document.querySelectorAll('a[href]').forEach(a => {
      if (!isVis(a) || a.closest('.v-navigation-drawer')) return;
      const t = txt(a); const h = a.getAttribute('href');
      if (!t || t.length > 200) return;
      const k = `${t}|${h}`; if (seenL.has(k)) return; seenL.add(k);
      C.links.push({ text: t, href: h, ...elMeta(a) });
    });

    // TABLES
    document.querySelectorAll('table,.v-data-table,.v-table').forEach((tbl, i) => {
      if (!isVis(tbl)) return;
      const hdrs = Array.from(tbl.querySelectorAll('th')).map(th => txt(th)).filter(Boolean);
      const rows = tbl.querySelectorAll('tbody tr');
      let sample = [];
      if (rows.length > 0) sample = Array.from(rows[0].querySelectorAll('td')).map(td => { const t = txt(td); return t.length > 60 ? t.slice(0, 57) + '...' : t; });
      if (!hdrs.length && !rows.length) return;
      C.tables.push({ index: i, headers: hdrs, rowCount: rows.length, sampleRow: sample, hasPagination: !!tbl.querySelector('.v-data-table-footer,.v-pagination'), ...elMeta(tbl) });
    });

    // CARDS
    document.querySelectorAll('.v-card').forEach((card, i) => {
      if (!isVis(card) || card.closest('.v-dialog,.v-navigation-drawer,.v-menu__content,.v-bottom-sheet,[class*="drawer"]')) return;
      const title = card.querySelector('.v-card-title')?.textContent?.trim() || null;
      const sub = card.querySelector('.v-card-subtitle')?.textContent?.trim() || null;
      const body = card.querySelector('.v-card-text')?.textContent?.trim() || null;
      const acts = Array.from(card.querySelectorAll('.v-card-actions button,.v-card-actions .v-btn')).map(b => txt(b)).filter(Boolean);
      if (!title && !sub && !body && !acts.length) return;
      C.cards.push({ index: i, title, subtitle: sub, text: body && body.length > 200 ? body.slice(0, 197) + '...' : body, actions: acts, ...elMeta(card) });
    });

    // CHIPS
    document.querySelectorAll('.v-chip').forEach((ch, i) => { if (!isVis(ch)) return; const t = txt(ch); if (t) C.chips.push({ index: i, text: t, closable: !!ch.querySelector('.v-chip__close'), ...elMeta(ch) }); });

    // TABS
    document.querySelectorAll('.v-tab').forEach((tab, i) => {
      if (!isVis(tab)) return;
      C.tabs.push({ index: i, text: txt(tab), isActive: tab.classList.contains('v-tab--selected') || tab.getAttribute('aria-selected') === 'true', ...elMeta(tab) });
    });

    // LISTS
    document.querySelectorAll('.v-list').forEach((list, i) => {
      if (!isVis(list) || list.closest('.v-navigation-drawer,.v-menu__content,.v-select__content,.v-autocomplete__content')) return;
      const items = Array.from(list.querySelectorAll('.v-list-item')).map(item => {
        const t = item.querySelector('.v-list-item-title')?.textContent?.trim() || txt(item);
        const s = item.querySelector('.v-list-item-subtitle')?.textContent?.trim() || null;
        return { title: t, subtitle: s, ...elMeta(item) };
      }).filter(it => it.title);
      if (!items.length) return;
      C.lists.push({ index: i, items: items.slice(0, 25), totalItems: items.length, ...elMeta(list) });
    });

    // ALERTS
    document.querySelectorAll('.v-alert').forEach((al, i) => {
      if (!isVis(al)) return;
      const type = ['error', 'success', 'warning', 'info'].find(t => al.classList.toString().includes(t)) || 'default';
      C.alerts.push({ index: i, type, text: txt(al).slice(0, 300), ...elMeta(al) });
    });

    // SNACKBARS
    document.querySelectorAll('.v-snackbar').forEach((sb, i) => { if (isVis(sb)) C.snackbars.push({ index: i, text: txt(sb).slice(0, 300), ...elMeta(sb) }); });

    // PROGRESS BARS
    document.querySelectorAll('.v-progress-linear,.v-progress-circular').forEach((pb, i) => {
      if (!isVis(pb)) return;
      C.progressBars.push({ index: i, type: pb.classList.contains('v-progress-circular') ? 'circular' : 'linear', ...elMeta(pb) });
    });

    // VISIBLE OVERLAYS — broad scan: dialogs, drawers, sheets, custom panels
    const oSels = ['.v-overlay--active .v-dialog','.v-dialog--active','.v-bottom-sheet','[class*="drawer"]:not(.v-navigation-drawer)','[class*="modal"]','[class*="sheet"]','[class*="popup"]','[class*="sidebar"]:not(.v-navigation-drawer)','.v-overlay--active > .v-overlay__content > *'];
    const seenO = new Set();
    document.querySelectorAll(oSels.join(',')).forEach((el, i) => {
      if (!isVis(el)) return; const r = el.getBoundingClientRect(); if (r.width < 50 || r.height < 50) return;
      const cls = el.className?.toString?.() || ''; if (seenO.has(cls)) return; seenO.add(cls);
      const content = el.querySelector('.v-card,.v-sheet,[class*="content"]') || el;
      const title = content.querySelector('.v-card-title,.v-toolbar-title,h1,h2,h3')?.textContent?.trim() || null;
      const body = content.querySelector('.v-card-text,.v-card__text')?.textContent?.trim() || null;
      const inps = Array.from(el.querySelectorAll('input:not([type="hidden"])')).filter(isVis).map(inp => ({ type: inp.type, label: inputLabel(inp) || inp.placeholder || null }));
      const btns = Array.from(el.querySelectorAll('button'))
        .filter(b => isVis(b) && !b.matches('.v-expansion-panel-title') && !b.closest('.v-expansion-panel-title'))
        .map(b => txt(b))
        .filter(Boolean);
      C.dialogs.push({ index: i, title, bodyText: body?.slice(0, 300) || null, inputs: inps, buttons: btns, ...elMeta(el) });
    });

    // MENUS
    document.querySelectorAll('.v-overlay--active .v-list').forEach((menu, i) => {
      const ov = menu.closest('.v-overlay');
      if (!ov || ov.closest('.v-dialog,.v-bottom-sheet,.v-navigation-drawer,[class*="drawer"]')) return;
      if (!isVis(menu)) return;
      const items = Array.from(menu.querySelectorAll('.v-list-item')).map(item => txt(item)).filter(Boolean);
      if (items.length) C.menus.push({ index: i, items, ...elMeta(menu) });
    });

    // EXPANSION PANELS
    document.querySelectorAll('.v-expansion-panel').forEach((p, i) => {
      if (!isVis(p)) return;
      const title = p.querySelector('.v-expansion-panel-title')?.textContent?.trim() || null;
      const open = p.classList.contains('v-expansion-panel--active');
      const cont = open ? p.querySelector('.v-expansion-panel-text')?.textContent?.trim() : null;
      C.expansionPanels.push({ index: i, title, isOpen: open, contentPreview: cont?.slice(0, 200) || null, ...elMeta(p) });
    });

    // NAV DRAWER
    const seenN = new Set();
    document.querySelectorAll('.v-navigation-drawer .v-list-item,nav .v-list-item').forEach(item => {
      if (!isVis(item)) return;
      const t = item.querySelector('.v-list-item-title')?.textContent?.trim() || txt(item);
      const href = item.getAttribute('href') || item.querySelector('a')?.getAttribute('href') || null;
      if (t && !seenN.has(t)) { seenN.add(t); C.navDrawerItems.push({ text: t, href, isActive: item.classList.contains('v-list-item--active') }); }
    });

    // TOOLBAR
    document.querySelectorAll('.v-app-bar,.v-toolbar:not(.v-app-bar .v-toolbar)').forEach(tb => {
      if (!isVis(tb)) return;
      C.toolbarItems.push({ title: tb.querySelector('.v-toolbar-title')?.textContent?.trim() || null, buttons: Array.from(tb.querySelectorAll('button,.v-btn')).map(b => txt(b) || aria(b)).filter(Boolean) });
    });

    // PAGINATION
    document.querySelectorAll('.v-pagination').forEach((pg, i) => { if (isVis(pg)) C.pagination.push({ index: i, totalPages: pg.querySelectorAll('.v-pagination__item').length }); });
    // IMAGES
    document.querySelectorAll('img').forEach(img => { if (isVis(img) && img.alt) C.images.push({ alt: img.alt }); });
    // AVATARS
    C.avatarCount = Array.from(document.querySelectorAll('.v-avatar')).filter(isVis).length;
    // BADGES
    document.querySelectorAll('.v-badge').forEach(b => { if (isVis(b)) { const c = b.querySelector('.v-badge__badge')?.textContent?.trim(); if (c) C.badges.push({ content: c }); } });
    // DIVIDERS
    C.hasDividers = Array.from(document.querySelectorAll('.v-divider')).some(isVis);
    // TOOLTIPS
    document.querySelectorAll('[title]:not(iframe)').forEach(el => { if (isVis(el) && el.title && el.title.length < 200) C.tooltipTriggers.push({ tooltip: el.title, tag: el.tagName.toLowerCase() }); });
    C.tooltipTriggers = C.tooltipTriggers.slice(0, 20);

    // VUE FLOW — nodes, edges, panels, toolbar controls
    // Use permissive check: just look for the root element existing, skip isVis
    // because the canvas container may have unusual layout (overflow, transforms).
    const vfRoot = document.querySelector('.vue-flow, .basic-flow, .dnd-flow');
    if (vfRoot) {
      const vf = { nodes: [], edges: [], panels: [], hasCanvas: true };

      // Nodes may be inside a deeply transformed container; query globally
      document.querySelectorAll('[data-id].vue-flow__node, .vue-flow__node[data-id]').forEach(node => {
        const dataId = node.getAttribute('data-id') || '';
        const typeClass = Array.from(node.classList).find(c => c.startsWith('vue-flow__node-') && c !== 'vue-flow__node') || '';
        const nodeType = typeClass.replace('vue-flow__node-', '');
        const label = node.querySelector('.node-container span, .terminal-node span, .default-node span')?.textContent?.trim() || dataId;
        const icon = node.querySelector('.mdi')
          ? Array.from(node.querySelector('.mdi').classList).find(c => c.startsWith('mdi-') && c !== 'mdi') || ''
          : '';
        vf.nodes.push({ dataId, nodeType, label, icon, selector: `[data-id="${dataId}"]` });
      });

      // Fallback: if no nodes found via class, try by data-id inside the flow container
      if (!vf.nodes.length) {
        vfRoot.querySelectorAll('[data-id][role="group"]').forEach(node => {
          const dataId = node.getAttribute('data-id') || '';
          const label = node.querySelector('span')?.textContent?.trim() || dataId;
          vf.nodes.push({ dataId, nodeType: 'unknown', label, icon: '', selector: `[data-id="${dataId}"]` });
        });
      }

      document.querySelectorAll('.vue-flow__edge, [data-id][aria-roledescription="edge"]').forEach(edge => {
        const dataId = edge.getAttribute('data-id') || '';
        const ariaLabel = edge.getAttribute('aria-label') || '';
        const pathEl = edge.querySelector('path.vue-flow__edge-path') || edge.querySelector('path[class*="edge-path"]');
        const source = pathEl?.getAttribute('source') || '';
        const target = pathEl?.getAttribute('target') || '';
        vf.edges.push({ dataId, source, target, ariaLabel, selector: `[data-id="${dataId}"]` });
      });

      document.querySelectorAll('.vue-flow__panel').forEach(panel => {
        const posClasses = Array.from(panel.classList).filter(c => /^(top|bottom|left|right)$/.test(c));
        const position = posClasses.join(' ') || 'unknown';
        const buttons = [];
        panel.querySelectorAll('button, [role="button"], .v-icon--clickable').forEach(btn => {
          const r = btn.getBoundingClientRect();
          if (r.width < 5 || r.height < 5) return;
          const btnText = btn.textContent?.trim().replace(/\s+/g, ' ') || '';
          const iconEl = btn.querySelector('.mdi, [class*="mdi-"]');
          const iconCls = iconEl ? Array.from(iconEl.classList).find(c => c.startsWith('mdi-') && c !== 'mdi') || '' : '';
          const cc = Array.from(btn.classList).filter(c => c.length > 2 && !FW_RE.test(c));
          buttons.push({ text: btnText, icon: iconCls, customClass: cc[0] || '', selector: cc[0] ? `.${cc[0]}` : (iconCls ? `button:has(.${iconCls})` : '') });
        });
        vf.panels.push({ position, buttons });
      });

      C.vueFlow = vf;
    }

    // MONACO EDITORS
    document.querySelectorAll('.overflow-guard, .monaco-editor').forEach((editor, i) => {
      if (!isVis(editor)) return;
      const rect = editor.getBoundingClientRect();
      const textarea = editor.querySelector('textarea[role="textbox"]');
      const ariaLabel = textarea?.getAttribute('aria-label') || '';
      const lineCount = editor.querySelectorAll('.view-line').length;
      C.monacoEditors.push({
        index: i,
        width: Math.round(rect.width),
        height: Math.round(rect.height),
        ariaLabel,
        lineCount,
        selector: `page.locator('.overflow-guard').nth(${i})`,
        textareaSelector: `page.locator('textarea[role="textbox"]').nth(${i})`,
      });
    });

    // ELEMENT REGISTRY — all elements with custom classes, IDs, or test IDs
    const regSeen = new Set();
    document.querySelectorAll('*').forEach(el => {
      if (!isVis(el)) return;
      const cc = customClasses(el); const id = el.id || ''; const t = tid(el);
      if (!cc.length && !id && !t) return;
      const k = `${el.tagName}|${cc.join(' ')}|${id}`; if (regSeen.has(k)) return; regSeen.add(k);
      const r = el.getBoundingClientRect(); if (r.width < 5 || r.height < 5) return;
      C.elementRegistry.push({ tag: el.tagName.toLowerCase(), customClasses: cc, allClasses: el.className?.toString?.() || '', id: id || null, testId: t || null, role: el.getAttribute('role') || null, selector: bestSel(el), text: (txt(el) || '').slice(0, 100) || null, size: `${Math.round(r.width)}x${Math.round(r.height)}` });
    });

    return C;
  }, FRAMEWORK_RE_SRC);
}

// ---------------------------------------------------------------------------
// Exploration helpers
// ---------------------------------------------------------------------------
async function scrollFullPage(page) {
  await page.evaluate(async () => {
    const t = document.querySelector('.v-main') || document.documentElement;
    for (let y = 0; y < t.scrollHeight; y += window.innerHeight) { t.scrollTo(0, y); await new Promise(r => setTimeout(r, 300)); }
    t.scrollTo(0, 0);
  });
  await page.waitForTimeout(800);
}

async function exploreTabs(page) {
  const results = [];
  const n = await page.locator('.v-tab:visible').count();
  if (n <= 1) return results;
  for (let i = 0; i < n; i++) {
    try {
      const tab = page.locator('.v-tab:visible').nth(i);
      const text = (await tab.textContent()).trim();
      const active = await tab.evaluate(el => el.classList.contains('v-tab--selected') || el.getAttribute('aria-selected') === 'true');
      if (!active) { await tab.click(); await page.waitForTimeout(1500); }
      results.push({ tabName: text, tabIndex: i, components: await extractAllComponents(page) });
    } catch (e) {}
  }
  return results;
}

async function snapshotOverlays(page) {
  return page.evaluate(() => {
    const out = [];
    document.querySelectorAll('*').forEach(el => {
      const r = el.getBoundingClientRect(); if (r.width < 80 || r.height < 50) return;
      const s = getComputedStyle(el); if (s.display === 'none' || s.visibility === 'hidden' || parseFloat(s.opacity) === 0) return;
      const z = parseInt(s.zIndex) || 0; const cls = el.className?.toString?.() || '';
      if ((s.position === 'fixed' && z >= 4 && r.width > 100) || /dialog|drawer|modal|sheet|overlay--active|popup|sidebar/i.test(cls))
        out.push({ className: cls, id: el.id || '', tag: el.tagName });
    });
    return out;
  });
}

async function identifyNewOverlay(page, beforeKeys) {
  return page.evaluate(({ bKeys, fwReSrc }) => {
    const bSet = new Set(bKeys); const FW = new RegExp(fwReSrc);
    const results = [];

    // First: check for any new .v-overlay--active (Vuetify's standard overlay wrapper)
    document.querySelectorAll('.v-overlay--active').forEach(ov => {
      const key = `${ov.tagName}|${ov.className?.toString?.() || ''}|${ov.id || ''}`;
      if (bSet.has(key)) return;
      const content = ov.querySelector('.v-overlay__content');
      if (!content) return;
      const r = content.getBoundingClientRect();
      if (r.width < 50 || r.height < 50) return;

      let bestChild = null;
      for (const child of content.querySelectorAll('*')) {
        const childCls = child.className?.toString?.() || '';
        const childCC = childCls.split(/\s+/).filter(c => c.length > 2 && !FW.test(c));
        if (!childCC.length) continue;
        const cr = child.getBoundingClientRect();
        if (cr.width < 20 || cr.height < 10) continue;
        if (!bestChild || cr.width * cr.height > bestChild.area) {
          bestChild = { sel: child.id ? `#${child.id}` : `.${childCC[0]}`, cls: childCls, cc: childCC, area: cr.width * cr.height };
        }
      }

      const ovCls = ov.className?.toString?.() || '';
      results.push({
        tag: 'div', className: ovCls, id: ov.id || null,
        selector: bestChild ? bestChild.sel : '.v-overlay--active .v-overlay__content',
        contentClassName: bestChild ? bestChild.cls : ovCls,
        allClasses: bestChild ? bestChild.cls : ovCls,
        customChildClasses: bestChild ? bestChild.cc : [],
      });
    });

    // Second: check for any other new overlay-like elements (custom modals, drawers, sheets)
    document.querySelectorAll('*').forEach(el => {
      if (el.closest('.v-overlay--active')) return;
      const r = el.getBoundingClientRect(); if (r.width < 80 || r.height < 50) return;
      const s = getComputedStyle(el); if (s.display === 'none' || s.visibility === 'hidden' || parseFloat(s.opacity) === 0) return;
      const z = parseInt(s.zIndex) || 0; const cls = el.className?.toString?.() || '';
      if (!((s.position === 'fixed' && z >= 4 && r.width > 100) || /dialog|drawer|modal|sheet|popup|sidebar/i.test(cls))) return;
      const key = `${el.tagName}|${cls}|${el.id || ''}`; if (bSet.has(key)) return;
      const classes = cls.split(/\s+/).filter(c => c);
      const cc = classes.find(c => c.length > 2 && !FW.test(c));
      let cSel = null, cCls = null;
      if (!cc) {
        for (const child of el.querySelectorAll('*')) {
          const childCls = child.className?.toString?.() || '';
          const childCC = childCls.split(/\s+/).filter(c => c.length > 2 && !FW.test(c));
          if (childCC.length) { const cr = child.getBoundingClientRect(); if (cr.width > 20 && cr.height > 10) { cSel = child.id ? `#${child.id}` : `.${childCC[0]}`; cCls = childCls; break; } }
        }
      }
      results.push({ tag: el.tagName.toLowerCase(), className: cls, id: el.id || null, selector: cSel || (el.id ? `#${el.id}` : cc ? `.${cc}` : classes[0] ? `.${classes[0]}` : el.tagName.toLowerCase()), contentClassName: cCls || cls, allClasses: cls, customChildClasses: [] });
    });

    results.sort((a, b) => { const ac = !a.selector.startsWith('.v-'); const bc = !b.selector.startsWith('.v-'); return ac === bc ? 0 : ac ? -1 : 1; });
    return results.length ? results[0] : null;
  }, { bKeys: beforeKeys, fwReSrc: FRAMEWORK_RE_SRC });
}

async function exploreDropdowns(page) {
  const results = [];
  const sel = '.v-select:visible,.v-autocomplete:visible,.v-combobox:visible';
  const n = await page.locator(sel).count();
  for (let i = 0; i < n; i++) {
    try {
      const s = page.locator(sel).nth(i);
      if (!(await s.isVisible().catch(() => false))) continue;
      if (await s.evaluate(el => el.classList.contains('v-input--disabled'))) continue;
      const label = await s.evaluate(el => {
        const vi = el.closest('.v-input');
        const own = vi?.querySelector('.v-label,.v-field__label,label')?.textContent?.trim();
        if (own) return own;
        const wrapper = vi?.parentElement || el.parentElement;
        const sibling = wrapper?.querySelector(':scope > label')?.textContent?.trim();
        if (sibling) return sibling;
        let cur = wrapper;
        for (let d = 0; d < 3 && cur; d++) {
          const near = cur.querySelector(':scope > label')?.textContent?.trim();
          if (near) return near;
          cur = cur.parentElement;
        }
        return null;
      });
      const selCls = await s.evaluate(el => el.className?.toString?.() || '');
      await s.click(); await page.waitForTimeout(1000);
      const info = await page.evaluate(() => {
        const ovs = Array.from(document.querySelectorAll('.v-overlay--active'));
        for (let j = ovs.length - 1; j >= 0; j--) {
          const list = ovs[j].querySelector('.v-list'); if (!list) continue;
          const r = list.getBoundingClientRect(); if (r.width < 20 || r.height < 20) continue;
          const items = Array.from(list.querySelectorAll('.v-list-item')).map(it => {
            const t = it.querySelector('.v-list-item-title')?.textContent?.trim() || it.textContent?.trim() || '';
            return { text: t, className: it.className?.toString?.() || '' };
          }).filter(it => it.text);
          return { items, listClassName: list.className?.toString?.() || '', itemClassName: items.length ? items[0].className : '' };
        }
        return null;
      });
      if (info?.items.length) {
        results.push({ index: i, label: label || `Dropdown ${i + 1}`, selectClassName: selCls, options: info.items.map(it => it.text), itemClassName: info.itemClassName, listClassName: info.listClassName });
        console.log(`    📋 Dropdown "${label || i}": ${info.items.length} opt(s) — ${info.items.map(o => o.text).join(', ')}`);
      }
      await page.keyboard.press('Escape'); await page.waitForTimeout(500);
    } catch (e) { await page.keyboard.press('Escape').catch(() => {}); await page.waitForTimeout(300); }
  }
  return results;
}

async function closeAnyOverlay(page, overlaySelector) {
  const closeSels = [
    // Custom drawer overlay close controls
    '.custom-drawer-overlay .drawer-header button',
    '.custom-drawer-overlay .custom-drawer .drawer-header button',
    '.custom-drawer-overlay button:has(.mdi-close)',
    // Specific overlay selector first
    `${overlaySelector} button:has-text("Cancel")`,
    `${overlaySelector} button:has-text("Close")`,
    `${overlaySelector} .drawer-header button`,
    `${overlaySelector} button:has(.mdi-close)`,
    // Vuetify overlay fallbacks
    '.v-overlay--active button:has-text("Cancel")',
    '.v-overlay--active button:has-text("Close")',
    '.v-overlay--active .mdi-close',
    '.v-overlay--active button.v-btn--icon',
  ];
  for (const cs of closeSels) {
    try {
      const cb = page.locator(cs).first();
      if (await cb.isVisible({ timeout: 300 }).catch(() => false)) { await cb.click(); await page.waitForTimeout(800); return true; }
    } catch (e) {}
  }
  await page.keyboard.press('Escape');
  await page.waitForTimeout(800);
  return false;
}

async function isOverlayStillOpen(page) {
  return page.evaluate(() => {
    const overlays = document.querySelectorAll('.v-overlay--active .v-overlay__content, .custom-drawer-overlay, .custom-drawer');
    for (const ov of overlays) {
      const r = ov.getBoundingClientRect();
      if (r.width > 50 && r.height > 50) return true;
    }
    return false;
  });
}

async function forceCloseAllOverlays(page, pageUrl) {
  for (let attempt = 0; attempt < 5; attempt++) {
    if (!(await isOverlayStillOpen(page))) return;
    // Try clicking explicit close controls before Escape.
    const closedByButton = await closeAnyOverlay(page, '.custom-drawer-overlay').catch(() => false);
    if (closedByButton) {
      await page.waitForTimeout(500);
      if (!(await isOverlayStillOpen(page))) return;
    }
    await page.keyboard.press('Escape');
    await page.waitForTimeout(600);
  }
  if (await isOverlayStillOpen(page)) {
    console.log(`    ⚠️ Overlays stuck after 5 Escape presses — navigating back`);
    await page.goto(`${BASE_URL}${pageUrl || ''}`, { waitUntil: 'networkidle', timeout: 15000 }).catch(() => {});
    await page.waitForTimeout(2000);
  }
}

async function exploreModals(page, pageUrl) {
  const results = [];
  const btns = await page.locator('button:visible,[role="button"]:visible').all();
  let explored = 0;
  const seenTriggers = new Set();
  for (const btn of btns) {
    if (explored >= MAX_MODALS) break;
    try {
      const t = (await btn.textContent()).trim().replace(/\s+/g, ' ');
      if (!t || t.length > 100 || DESTRUCTIVE_RE.test(t) || !MODAL_TRIGGER_RE.test(t)) continue;
      if (seenTriggers.has(t)) continue; // skip duplicate button text
      if (await btn.evaluate(el => el.disabled || el.classList.contains('v-btn--disabled'))) continue;
      if (await btn.evaluate(el => !!el.closest('.v-navigation-drawer'))) continue;
      seenTriggers.add(t);

      // Ensure no previous overlay is still blocking interactions.
      await forceCloseAllOverlays(page, pageUrl).catch(() => {});
      await page.waitForTimeout(500);

      const before = await snapshotOverlays(page);
      const bKeys = before.map(o => `${o.tag}|${o.className}|${o.id}`);
      const urlBefore = page.url();
      console.log(`    🔍 Clicking "${t}"...`);
      try {
        await btn.click({ timeout: CLICK_TIMEOUT_MS });
      } catch (clickErr) {
        // Fallback: re-find a fresh trigger by text in case the original handle
        // is stale/covered after prior modal interactions.
        const safeText = t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const candidates = await page.locator('button:visible,[role="button"]:visible')
          .filter({ hasText: new RegExp(`^\\s*${safeText}\\s*$`, 'i') })
          .all();
        let clicked = false;
        for (const c of candidates) {
          try {
            const blocked = await c.evaluate(el => !!el.closest('.v-overlay--active,.v-overlay__content,.custom-drawer-overlay,.custom-drawer'));
            if (blocked) continue;
            await c.scrollIntoViewIfNeeded().catch(() => {});
            await c.click({ timeout: CLICK_TIMEOUT_MS });
            clicked = true;
            break;
          } catch (_) {}
        }
        if (!clicked) throw clickErr;
      }
      await page.waitForTimeout(3000);

      if (page.url() !== urlBefore) {
        console.log(`    ↩ Navigation — going back`);
        await page.goto(urlBefore, { waitUntil: 'networkidle', timeout: 15000 }).catch(() => {});
        await page.waitForTimeout(1500);
        continue;
      }

      const overlay = await identifyNewOverlay(page, bKeys);
      if (overlay) {
        console.log(`    ✓ Overlay: ${overlay.selector} (${overlay.allClasses})`);

        // Wait for any loading state to finish inside the modal
        try {
          const loadingEl = page.locator('.v-overlay--active, .custom-drawer-overlay').locator('text=/loading/i').first();
          if (await loadingEl.isVisible({ timeout: 1000 }).catch(() => false)) {
            console.log('    ⏳ Modal is loading, waiting for data...');
            await loadingEl.waitFor({ state: 'hidden', timeout: 15000 }).catch(() => {});
            await page.waitForTimeout(1500);
          }
        } catch (e) {}

        const comps = await extractAllComponents(page);
        console.log(`    📋 Exploring dropdowns inside overlay...`);
        const dds = await exploreDropdowns(page);

        // Explore sub-buttons inside the modal that reveal more form elements
        // (e.g. "Create Schedule" reveals Sync Type / Frequency dropdowns)
        const subResults = [];
        const WIZARD_STEP_RE = /continue|next|proceed|step/i;

        // For wizard-style modals: fill empty text inputs with dummy values so
        // Continue/Next buttons become enabled, then explore each dropdown option path
        const wizardBtnSel = '.v-overlay--active button:visible, .custom-drawer button:visible, .custom-drawer-overlay button:visible';
        const hasWizardBtn = await page.locator(wizardBtnSel).evaluateAll(btns =>
          btns.some(b => /continue|next|proceed/i.test(b.textContent?.trim() || ''))
        ).catch(() => false);

        if (hasWizardBtn) {
          // Fill empty visible text inputs with dummy value to unblock wizard validation
          const emptyInputs = await page.locator('.v-overlay--active input[type="text"]:visible, .custom-drawer input[type="text"]:visible, .custom-drawer-overlay input[type="text"]:visible').all();
          for (const inp of emptyInputs) {
            try {
              const val = await inp.inputValue().catch(() => '');
              if (!val) { await inp.fill('__inspect_dummy__'); await page.waitForTimeout(300); console.log('    📝 Filled empty input with dummy value'); }
            } catch (e) {}
          }

          // Explore each dropdown option path (e.g. source type variations)
          const ddSel = '.v-overlay--active .v-select:visible, .custom-drawer .v-select:visible, .custom-drawer-overlay .v-select:visible, .v-overlay--active .v-autocomplete:visible, .custom-drawer .v-autocomplete:visible';
          const ddCount = await page.locator(ddSel).count();
          const exploredPaths = [];
          for (let di = 0; di < ddCount; di++) {
            try {
              const dd = page.locator(ddSel).nth(di);
              if (!(await dd.isVisible().catch(() => false))) continue;
              if (await dd.evaluate(el => el.classList.contains('v-input--disabled'))) continue;
              const ddLabel = await dd.evaluate(el => el.closest('.v-input')?.querySelector('.v-label,.v-field__label')?.textContent?.trim() || null);
                  await dd.click({ timeout: CLICK_TIMEOUT_MS }); await page.waitForTimeout(800);
              const opts = await page.locator('.v-overlay--active .v-list-item:visible').allTextContents();
              await page.keyboard.press('Escape'); await page.waitForTimeout(500);
              if (opts.length <= 1) continue;
              for (const optText of opts) {
                const ot = optText.trim();
                if (!ot) continue;
                try {
                  console.log(`    🔄 Trying dropdown "${ddLabel || di}" → "${ot}" path...`);
                  await dd.click({ timeout: CLICK_TIMEOUT_MS }); await page.waitForTimeout(500);
                  await page.locator('.v-overlay--active .v-list-item').filter({ hasText: new RegExp(ot.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')) }).first().click({ timeout: CLICK_TIMEOUT_MS });
                  await page.waitForTimeout(1000);
                  // Capture state right after option selection (before moving to next step).
                  // Website Crawler panels live on this step.
                  const currentComps = await extractAllComponents(page);
                  const currentDds = await exploreDropdowns(page);
                  const currentPanels = await exploreExpansionPanels(page);
                  exploredPaths.push({ subTrigger: `Selected ${ddLabel || 'dropdown'} = ${ot}`, components: currentComps, dropdowns: currentDds, panels: currentPanels });
                  console.log(`    ✓ Captured selected-option state for "${ot}" — ${currentPanels.length} panel(s), ${currentDds.length} dropdown(s)`);

                  // Now click Continue/Next.
                  // IMPORTANT: for Website Crawler, crawler panels are on this same step;
                  // continuing here can hide them and mislead generated tests.
                  if (/website\s*crawler/i.test(ot)) {
                    continue;
                  }
                  const wizBtn = page.locator('.v-overlay--active button:visible, .custom-drawer button:visible').filter({ hasText: /continue|next/i }).first();
                  if (await wizBtn.isVisible({ timeout: 1000 }).catch(() => false) && !(await wizBtn.evaluate(el => el.disabled || el.classList.contains('v-btn--disabled')).catch(() => true))) {
                    await wizBtn.click({ timeout: CLICK_TIMEOUT_MS }); await page.waitForTimeout(2000);
                    const pathComps = await extractAllComponents(page);
                    const pathDds = await exploreDropdowns(page);
                    const pathPanels = await exploreExpansionPanels(page);
                    exploredPaths.push({ subTrigger: `Continue (${ddLabel || 'dropdown'} = ${ot})`, components: pathComps, dropdowns: pathDds, panels: pathPanels });
                    console.log(`    ✓ Explored wizard step for "${ot}" — ${pathPanels.length} panel(s), ${pathDds.length} dropdown(s)`);
                    // Go back: press Back button or re-navigate
                    const backBtn = page.locator('.v-overlay--active button:visible, .custom-drawer button:visible').filter({ hasText: /back|previous/i }).first();
                    if (await backBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
                      await backBtn.click(); await page.waitForTimeout(1500);
                    } else {
                      // Re-open the modal to try next option
                      await closeAnyOverlay(page, overlay.selector);
                      await forceCloseAllOverlays(page, pageUrl);
                      await page.waitForTimeout(1000);
                      // Re-click the original trigger button
                      try { await page.locator(`button:has-text("${t}")`).first().click({ timeout: CLICK_TIMEOUT_MS }); await page.waitForTimeout(2000); } catch (e) { break; }
                      // Re-fill dummy inputs
                      const reInputs = await page.locator('.v-overlay--active input[type="text"]:visible, .custom-drawer input[type="text"]:visible, .custom-drawer-overlay input[type="text"]:visible').all();
                      for (const ri of reInputs) { try { const v = await ri.inputValue().catch(() => ''); if (!v) await ri.fill('__inspect_dummy__'); } catch (e) {} }
                      await page.waitForTimeout(500);
                    }
                  }
                } catch (e) { console.log(`    ✗ Path "${ot}": ${e.message?.slice(0, 60)}`); }
              }
            } catch (e) {}
          }
          subResults.push(...exploredPaths);
        }

        // Also explore non-wizard sub-buttons (add/create/manage/etc.)
        const subBtns = await page.locator('.v-overlay--active button:visible, .custom-drawer button:visible').all();
        for (const subBtn of subBtns) {
          try {
            const subText = (await subBtn.textContent()).trim().replace(/\s+/g, ' ');
            if (!subText || subText.length > 80) continue;
            if (DESTRUCTIVE_RE.test(subText)) continue;
            if (WIZARD_STEP_RE.test(subText)) continue;
            if (!MODAL_TRIGGER_RE.test(subText)) continue;
            if (await subBtn.evaluate(el => el.disabled || el.classList.contains('v-btn--disabled'))) continue;
            console.log(`    🔍 Sub-clicking "${subText}" inside overlay...`);
            await subBtn.click({ timeout: CLICK_TIMEOUT_MS });
            await page.waitForTimeout(2000);
            const subComps = await extractAllComponents(page);
            console.log(`    📋 Exploring dropdowns after sub-click...`);
            const subDds = await exploreDropdowns(page);
            console.log(`    📋 Exploring expansion panels after sub-click...`);
            const subPanels = await exploreExpansionPanels(page);
            subResults.push({ subTrigger: subText, components: subComps, dropdowns: subDds, panels: subPanels });
          } catch (e) {}
        }

        results.push({ trigger: t, overlaySelector: overlay.selector, overlayClassName: overlay.allClasses, overlayId: overlay.id, components: comps, dropdowns: dds, subExplorations: subResults });
        explored++;

        // Close the overlay
        await closeAnyOverlay(page, overlay.selector);
        await forceCloseAllOverlays(page, pageUrl);
        // After capturing the Create KB Bucket wizard panels, continue
        // to explore other triggers (e.g. Schedule) on the same page.
      } else {
        console.log(`    ✗ No new overlay detected after clicking "${t}"`);
      }
    } catch (e) {
      console.log(`    ✗ Error exploring "${e.message?.slice(0, 80)}"`);
      await forceCloseAllOverlays(page, pageUrl).catch(() => {});
    }
  }
  return results;
}

async function exploreExpansionPanels(page) {
  const results = [];
  const n = await page.locator('.v-expansion-panel:visible').count();
  for (let i = 0; i < n; i++) {
    try {
      const p = page.locator('.v-expansion-panel:visible').nth(i);
      const title = (await p.locator('.v-expansion-panel-title').textContent().catch(() => '')).trim();
      if (!(await p.evaluate(el => el.classList.contains('v-expansion-panel--active')))) { await p.locator('.v-expansion-panel-title').click(); await page.waitForTimeout(1000); }
      results.push({ panelTitle: title, panelIndex: i, components: await extractAllComponents(page) });
    } catch (e) {}
  }
  return results;
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------
async function isOnLoginPage(page) { return page.url().includes('/login') || page.url().includes('/signup'); }

async function doLogin(page) {
  console.log(`  Logging in as ${LOGIN_EMAIL}...`);
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });

  const emailInput = page.locator('.v-text-field').nth(0).locator('input');
  const passwordInput = page.locator('.v-text-field').nth(1).locator('input');
  const loginButton = page.getByRole('button', { name: /login/i });

  await emailInput.waitFor({ state: 'visible', timeout: 10000 });
  await passwordInput.waitFor({ state: 'visible', timeout: 5000 });
  await loginButton.waitFor({ state: 'visible', timeout: 5000 });

  await emailInput.fill(LOGIN_EMAIL);
  await passwordInput.fill(LOGIN_PASSWORD);
  await loginButton.click();

  try {
    await page.waitForURL('**/dashboard.int3nt.info/?select_org', { timeout: 20000 });
    console.log('  ✓ Login successful, on org selection page');
  } catch (e) {
    console.log('  ⚠️  Login failed or timed out');
    return false;
  }

  // Wait for any loader to disappear before looking for org cards
  const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
  if (await loader.first().isVisible().catch(() => false)) {
    console.log('  ⏳ Waiting for loader to disappear...');
    try { await loader.first().waitFor({ state: 'hidden', timeout: 15000 }); } catch (e) {}
  }
  await page.waitForTimeout(2000);

  console.log(`  Selecting org "${ORG_NAME}"...`);
  const nameRegex = new RegExp(`^\\s*${ORG_NAME.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*$`, 'i');

  // Primary: find .organization-card with matching .organization-name
  const orgCard = page.locator('.organization-card').filter({
    has: page.locator('.organization-name', { hasText: nameRegex })
  }).first();

  if (await orgCard.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log(`  Found org card for "${ORG_NAME}"`);
    await orgCard.click();
  } else {
    // Fallback: sidebar dropdown (already logged in with an org)
    console.log('  Card not found, checking sidebar dropdown...');
    const trigger = page.locator('.org-dropdown-trigger');
    if (await trigger.isVisible({ timeout: 5000 }).catch(() => false)) {
      const currentLabel = await trigger.locator('.org-name').innerText().catch(() => '');
      if (currentLabel.trim().toLowerCase() === ORG_NAME.toLowerCase()) {
        console.log(`  ✓ "${ORG_NAME}" already selected`);
        return true;
      }
      await trigger.click();
      const dropdownItem = page.locator('.org-dropdown-item').filter({
        has: page.locator('.org-name', { hasText: nameRegex })
      }).first();
      await dropdownItem.waitFor({ state: 'visible', timeout: 3000 });
      await dropdownItem.click();
    } else {
      // Last resort: click any visible org card
      console.log('  No dropdown either, trying first available card...');
      try {
        await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
        await page.locator('.organization-card').first().click();
      } catch (e2) {
        console.log('  ❌ Org selection failed:', e2.message);
        return false;
      }
    }
  }

  await page.waitForLoadState('networkidle');
  try {
    await page.waitForURL(url => !url.includes('select_org'), { timeout: 15000 });
  } catch (e) {}

  // Ensure the org selection container is gone
  const selectOrgContainer = page.locator('.select-org-container');
  if (await selectOrgContainer.isVisible().catch(() => false)) {
    try { await selectOrgContainer.waitFor({ state: 'hidden', timeout: 10000 }); } catch (e) {}
  }

  console.log('  ✓ Org selected, URL:', page.url());
  return true;
}

// ---------------------------------------------------------------------------
// Page inspection
// ---------------------------------------------------------------------------
async function inspectPage(page, url, pageName, skipExploration = false) {
  console.log(`\n${'─'.repeat(60)}\n📍 ${pageName}  (${url})`);
  try { await page.goto(`${BASE_URL}${url}`, { waitUntil: 'networkidle', timeout: 20000 }); await page.waitForTimeout(2000); }
  catch (e) { console.log('  ⚠️  Timeout — continuing'); await page.waitForTimeout(1000); }
  if (await isOnLoginPage(page) && !url.includes('/login') && !url.includes('/signup') && !url.includes('/forgot') && !url.includes('/register') && !url.includes('/set-password')) {
    console.log('  ⛔ Redirected to login'); return { url, pageName, authFailed: true, components: null, explored: null };
  }
  console.log('  ① Scrolling...'); await scrollFullPage(page);
  console.log('  ② Component scan...'); const components = await extractAllComponents(page);
  let tabs = [], modals = [], panels = [], dropdowns = [];
  if (!skipExploration) {
    console.log('  ③ Tabs...'); tabs = await exploreTabs(page);
    console.log('  ④ Modals...'); modals = await exploreModals(page, url);
    console.log('  ⑤ Panels...'); panels = await exploreExpansionPanels(page);
    console.log('  ⑥ Dropdowns...'); dropdowns = await exploreDropdowns(page);
    // ⑦ Vue Flow exploration — panel buttons + node clicks
    if (components.vueFlow) {
      const vf = components.vueFlow;
      console.log(`  ⑦ Vue Flow (${vf.nodes.length} nodes, ${vf.edges.length} edges, ${vf.panels.length} panels)...`);

      // Helper: after clicking something on the flow canvas, detect what appeared.
      // Checks Vuetify overlays, then custom dropdown menus (non-Vuetify), then Vuetify menus.
      async function detectFlowUIAfterClick(page, beforeKeys, triggerLabel) {
        // 1. Standard Vuetify overlay (modals, dialogs)
        const overlay = await identifyNewOverlay(page, beforeKeys);
        if (overlay) {
          console.log(`    ✓ Overlay: ${overlay.selector} (${overlay.allClasses})`);
          try {
            const loadEl = page.locator('.v-overlay--active, .custom-drawer-overlay').locator('text=/loading/i').first();
            if (await loadEl.isVisible({ timeout: 1000 }).catch(() => false)) {
              await loadEl.waitFor({ state: 'hidden', timeout: 10000 }).catch(() => {});
              await page.waitForTimeout(1000);
            }
          } catch (_) {}
          const comps = await extractAllComponents(page);
          const dds = await exploreDropdowns(page);
          modals.push({ trigger: triggerLabel, overlaySelector: overlay.selector, overlayClassName: overlay.allClasses, overlayId: overlay.id, components: comps, dropdowns: dds, subExplorations: [] });
          await closeAnyOverlay(page, overlay.selector);
          await forceCloseAllOverlays(page, url);
          return true;
        }

        // 2. Custom dropdown menus (not Vuetify overlays — rendered inline/absolute)
        const customDDSels = [
          '[class*="dropdown-menu"]:visible',
          '.custom-dialog:visible',
          '.modal-card:visible',
        ];
        for (const ddSel of customDDSels) {
          try {
            const dd = page.locator(ddSel).first();
            if (await dd.isVisible({ timeout: 1000 }).catch(() => false)) {
              const ddClass = await dd.evaluate(el => {
                const cc = Array.from(el.classList).filter(c => c.length > 2 && !/^(v-|pa-|ma-|d-|text-)/.test(c));
                return cc[0] || el.className?.toString?.().slice(0, 60) || 'custom-dropdown';
              });
              console.log(`    ✓ Custom dropdown: .${ddClass}`);
              const comps = await extractAllComponents(page);
              const dds = await exploreDropdowns(page);
              const items = await dd.locator('[class*="dropdown-item"]').allTextContents().catch(() => []);
              modals.push({ trigger: triggerLabel, overlaySelector: `.${ddClass}`, overlayClassName: ddClass, overlayId: '', components: comps, dropdowns: dds, subExplorations: [], menuItems: items.map(t => t.trim()).filter(Boolean) });
              // Close by clicking elsewhere or Escape
              await page.keyboard.press('Escape');
              await page.waitForTimeout(500);
              // If still visible, click on the canvas background to dismiss
              if (await dd.isVisible({ timeout: 300 }).catch(() => false)) {
                await page.locator('.vue-flow__pane').click({ position: { x: 10, y: 10 }, force: true }).catch(() => {});
                await page.waitForTimeout(500);
              }
              return true;
            }
          } catch (_) {}
        }

        // 3. Vuetify overlay that may have appeared (re-check directly)
        try {
          const directOverlay = page.locator('.v-overlay--active .v-card, .v-overlay--active .v-list').first();
          if (await directOverlay.isVisible({ timeout: 1000 }).catch(() => false)) {
            const isList = await directOverlay.evaluate(el => el.classList.contains('v-list'));
            if (isList) {
              console.log('    ✓ Vuetify menu appeared');
              const items = await page.locator('.v-overlay--active .v-list-item').allTextContents();
              const comps = await extractAllComponents(page);
              modals.push({ trigger: triggerLabel, overlaySelector: '.v-overlay--active', overlayClassName: '', overlayId: '', components: comps, dropdowns: [], subExplorations: [], menuItems: items.map(t => t.trim()).filter(Boolean) });
            } else {
              console.log('    ✓ Vuetify modal appeared');
              const comps = await extractAllComponents(page);
              const dds = await exploreDropdowns(page);
              modals.push({ trigger: triggerLabel, overlaySelector: '.v-overlay--active .v-card', overlayClassName: '', overlayId: '', components: comps, dropdowns: dds, subExplorations: [] });
            }
            await page.keyboard.press('Escape');
            await page.waitForTimeout(500);
            await forceCloseAllOverlays(page, url).catch(() => {});
            return true;
          }
        } catch (_) {}

        console.log(`    ✗ No UI detected from "${triggerLabel}"`);
        return false;
      }

      // 7a: Click each panel button to discover menus/overlays/dropdowns
      const SKIP_ICONS = /mdi-undo|mdi-redo|mdi-magnify-minus|mdi-magnify-plus|mdi-content-save/i;
      for (const panel of vf.panels) {
        for (const btn of panel.buttons) {
          if (!btn.selector) continue;
          if (SKIP_ICONS.test(btn.icon)) continue;
          if (btn.customClass && /history-button-container/i.test(btn.customClass)) continue;
          try {
            const el = page.locator(btn.selector).first();
            if (!(await el.isVisible({ timeout: 1500 }).catch(() => false))) continue;
            if (await el.evaluate(e => e.disabled || e.classList.contains('v-btn--disabled')).catch(() => false)) continue;

            // Dismiss anything open first
            await page.keyboard.press('Escape'); await page.waitForTimeout(300);
            await page.locator('.vue-flow__pane').click({ position: { x: 10, y: 10 }, force: true }).catch(() => {});
            await page.waitForTimeout(300);
            await forceCloseAllOverlays(page, url).catch(() => {});
            await page.waitForTimeout(300);

            const beforeSnap = await snapshotOverlays(page);
            const beforeKeys = beforeSnap.map(o => `${o.tag}|${o.className}|${o.id}`);
            const label = btn.text || btn.icon || btn.customClass;
            console.log(`    🔍 Clicking panel button "${label}" (${btn.selector})...`);
            await el.click({ timeout: CLICK_TIMEOUT_MS });
            await page.waitForTimeout(3000);

            await detectFlowUIAfterClick(page, beforeKeys, `Panel: ${label}`);
          } catch (e) {
            console.log(`    ✗ Panel button "${btn.text || btn.icon}": ${e.message?.slice(0, 60)}`);
            await page.keyboard.press('Escape').catch(() => {});
            await forceCloseAllOverlays(page, url).catch(() => {});
          }
        }
      }

      // 7b: Click each non-terminal node to discover config modals/drawers
      for (const node of vf.nodes) {
        if (/^(START|END)$/i.test(node.dataId)) continue;
        try {
          const nodeEl = page.locator(`[data-id="${node.dataId}"]`).first();
          if (!(await nodeEl.isVisible({ timeout: 2000 }).catch(() => false))) continue;

          await page.keyboard.press('Escape'); await page.waitForTimeout(300);
          await forceCloseAllOverlays(page, url).catch(() => {});
          const beforeSnap = await snapshotOverlays(page);
          const beforeKeys = beforeSnap.map(o => `${o.tag}|${o.className}|${o.id}`);
          console.log(`    🔍 Clicking node "${node.label}" (${node.dataId})...`);
          await nodeEl.click({ timeout: CLICK_TIMEOUT_MS });
          await page.waitForTimeout(2000);

          await detectFlowUIAfterClick(page, beforeKeys, `Node: ${node.label}`);
        } catch (e) {
          console.log(`    ✗ Node "${node.label}": ${e.message?.slice(0, 60)}`);
          await page.keyboard.press('Escape').catch(() => {});
          await forceCloseAllOverlays(page, url).catch(() => {});
        }
      }

      // 7c: Click each edge to discover edge config modals
      for (const edge of vf.edges) {
        try {
          const edgeEl = page.locator(`[data-id="${edge.dataId}"]`).first();
          if (!(await edgeEl.isVisible({ timeout: 2000 }).catch(() => false))) continue;

          await page.keyboard.press('Escape'); await page.waitForTimeout(300);
          await forceCloseAllOverlays(page, url).catch(() => {});
          const beforeSnap = await snapshotOverlays(page);
          const beforeKeys = beforeSnap.map(o => `${o.tag}|${o.className}|${o.id}`);
          const edgeLabel = edge.ariaLabel || `${edge.source} → ${edge.target}`;
          console.log(`    🔍 Clicking edge "${edgeLabel}" (${edge.dataId})...`);
          await edgeEl.click({ timeout: CLICK_TIMEOUT_MS });
          await page.waitForTimeout(2000);

          await detectFlowUIAfterClick(page, beforeKeys, `Edge: ${edgeLabel}`);
        } catch (e) {
          console.log(`    ✗ Edge "${edge.dataId}": ${e.message?.slice(0, 60)}`);
          await page.keyboard.press('Escape').catch(() => {});
          await forceCloseAllOverlays(page, url).catch(() => {});
        }
      }
    }
  } else { console.log('  ③–⑦ Skipping exploration (public page)'); }
  const explored = { tabs, modals, panels, dropdowns };
  const counts = Object.entries(components).filter(([, v]) => Array.isArray(v) ? v.length > 0 : (typeof v === 'number' ? v > 0 : !!v)).map(([k, v]) => `${k}:${Array.isArray(v) ? v.length : v}`).join('  ');
  console.log(`  → ${counts}`);
  console.log(`  → Explored ${tabs.length} tab(s), ${modals.length} modal(s), ${panels.length} panel(s), ${dropdowns.length} dropdown(s)`);
  return { url, pageName, authFailed: false, components, explored };
}

// ---------------------------------------------------------------------------
// SKILL.md generation — uses REAL selectors from DOM, never guesses classes
// ---------------------------------------------------------------------------
function generateLoginSkillMd() {
  return `---
name: app-selectors
description: Global login and organization selection flow for dashboard.int3nt.info. Use this when writing Playwright tests that require authentication.
---

# Global Login & Org Selection

> Generated: ${new Date().toISOString()}

## Test Credentials

| Field | Value |
|-------|-------|
| Email | \`${LOGIN_EMAIL}\` |
| Password | \`${LOGIN_PASSWORD}\` |
| Organization | \`${ORG_NAME}\` |

## Login Flow (copy-paste ready)

\`\`\`typescript
// Step 1 — Navigate
await page.goto('${BASE_URL}/login', { waitUntil: 'networkidle' });

// Step 2 — Fill credentials (use EXACTLY these selectors)
await page.locator('.v-text-field').nth(0).locator('input').fill('${LOGIN_EMAIL}');
await page.locator('.v-text-field').nth(1).locator('input').fill('${LOGIN_PASSWORD}');
await page.getByRole('button', { name: /login/i }).click();

// Step 3 — Wait for redirect to org selection
await page.waitForURL(/\\?select_org/, { timeout: 20000 });

// Step 4 — Wait for loader to disappear, then select organization
const loader = page.locator('.loading-container, .loading-spinner, .v-progress-linear');
if (await loader.first().isVisible().catch(() => false)) {
  await loader.first().waitFor({ state: 'hidden', timeout: 15000 });
}
await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
await page.locator('.organization-card').filter({ hasText: '${ORG_NAME}' }).click();
await page.waitForURL(/dashboard\\.int3nt\\.info\\/(?!\\?select_org)/, { timeout: 15000 });
\`\`\`

## Key Selectors

| Element | Selector |
|---------|----------|
| Email input | \`.v-text-field:nth(0) input\` |
| Password input | \`.v-text-field:nth(1) input\` |
| Login button | \`getByRole('button', { name: /login/i })\` |
| Org card | \`.organization-card\` |
| Org name in card | \`.organization-name\` |
| Org dropdown trigger | \`.org-dropdown-trigger\` |
| Org dropdown item | \`.org-dropdown-item\` |

**Known orgs:** "${ORG_NAME}", "Testing"
`;
}

function generatePageSkillMd(data, pageName) {
  const slug = slugify(pageName);
  let md = `---
name: app-selectors-${slug}
description: DOM selectors and component map for the ${pageName} page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# ${pageName} — Component Map

> Generated: ${new Date().toISOString()}
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

`;
  md += generatePageSection(data);
  return md;
}

function pickSel(comp, fallback) {
  if (comp.testId) return `[data-testid="${comp.testId}"]`;
  if (comp.id && !/^input-v-|^v-/.test(comp.id)) return `#${comp.id}`;
  if (comp.customClasses?.length) return `.${comp.customClasses[0]}`;
  if (comp.containerSelector) return comp.containerSelector;
  if (comp.selector && !comp.selector.startsWith('.v-')) return comp.selector;
  return fallback;
}

function generatePageSection(data) {
  let s = `### ${data.pageName}\n**URL:** \`${data.url}\`\n\n`;
  if (data.authFailed) { s += `> ⚠️ Auth expired — re-run script\n\n---\n\n`; return s; }
  const C = data.components;

  if (C.headings.length) { s += `**Headings:**\n`; C.headings.forEach(h => { s += `- \`${h.tag}\` — "${h.text}"`; if (h.selector) s += ` (selector: \`${h.selector}\`)`; s += `\n`; }); s += '\n'; }
  if (C.breadcrumbs.length) { s += `**Breadcrumbs:** ${C.breadcrumbs.join(' › ')}\n\n`; }
  if (C.toolbarItems.length) { C.toolbarItems.forEach(tb => { if (tb.title) s += `**Toolbar:** "${tb.title}"\n`; if (tb.buttons.length) s += `**Toolbar Buttons:** ${tb.buttons.map(b => `\`${b}\``).join(', ')}\n`; }); s += '\n'; }
  if (C.staticTexts.length) { s += `**Text Content (${C.staticTexts.length}):**\n`; C.staticTexts.forEach(t => { s += `- [${t.type}] "${t.text}"`; if (t.selector) s += ` → \`${t.selector}\``; s += `\n`; }); s += '\n'; }

  const formInputs = C.inputs.filter(inp => !inp.inSelect);
  if (formInputs.length) {
    s += `**Input Fields (${formInputs.length}):**\n\n| # | Label | Type | Selector |\n|---|-------|------|----------|\n`;
    formInputs.forEach((inp, i) => {
      const label = inp.label || (inp.type === 'password' ? 'Password' : `Input ${i + 1}`);
      let sel = pickSel(inp, null);
      if (!sel && inp.placeholder) sel = `input[placeholder="${inp.placeholder}"]`;
      if (!sel) sel = `.custom-drawer input[type="${inp.type || 'text'}"]`;
      s += `| ${i + 1} | ${label} | \`${inp.type}\` | \`${sel}\` |\n`;
    });
    s += `\n**Input selector rule:** Use \`input[placeholder="..."]\` or \`.nth(N)\` on scoped container inputs. Do NOT use \`.filter({ hasText })\` on a \`div\` to match placeholder text — placeholders are attributes, not visible text content.\n\n`;
  }
  if (C.textareas.length) { s += `**Textareas (${C.textareas.length}):**\n\n| # | Label | Selector |\n|---|-------|----------|\n`; C.textareas.forEach((ta, i) => { s += `| ${i + 1} | ${ta.label || ta.placeholder || `Textarea ${i + 1}`} | \`${pickSel(ta, `page.locator('textarea').nth(${i})`)}\` |\n`; }); s += '\n'; }

  if (C.selects.length) {
    const eDDs = data.explored?.dropdowns || [];
    s += `**Dropdowns / Selects (${C.selects.length}):**\n`;
    C.selects.forEach((sel, i) => {
      const selSelector = pickSel(sel, `.v-select:nth(${sel.index})`);
      s += `- **${sel.label || `Dropdown ${i + 1}`}** (${sel.type})`;
      if (sel.currentValue) s += ` — current: "${sel.currentValue}"`;
      if (sel.disabled) s += ` [disabled]`;
      s += `\n  - Selector: \`${selSelector}\`\n  - Open: \`page.locator('${selSelector}').click()\`\n`;
      const match = eDDs.find(dd => dd.label === sel.label || dd.index === sel.index);
      if (match?.options.length) { s += `  - Options: ${match.options.map(o => `\`${o}\``).join(', ')}\n  - Pick: \`page.locator('.v-list-item:has-text("OPTION")').click()\`\n`; }
      else { s += `  - Pick: \`page.locator('.v-list-item:has-text("OPTION")').click()\`\n`; }
    });
    s += '\n';
  }

  if (C.checkboxes.length) { s += `**Checkboxes (${C.checkboxes.length}):**\n`; C.checkboxes.forEach((cb, i) => { const sel = pickSel(cb, `.v-checkbox:nth(${i})`); s += `- ${cb.label || `Checkbox ${i + 1}`} — ${cb.checked ? '☑' : '☐'}${cb.disabled ? ' [disabled]' : ''}\n  \`page.locator('${sel}').click()\`\n`; }); s += '\n'; }
  if (C.switches.length) { s += `**Switches (${C.switches.length}):**\n`; C.switches.forEach((sw, i) => { const sel = pickSel(sw, `.v-switch:nth(${i})`); s += `- ${sw.label || `Switch ${i + 1}`} — ${sw.checked ? 'ON' : 'OFF'}${sw.disabled ? ' [disabled]' : ''}\n  \`page.locator('${sel}').click()\`\n`; }); s += '\n'; }
  if (C.radios.length) { s += '**Radio Groups (\`.v-radio-group\`):**\n'; C.radios.forEach((rg, i) => { const sel = pickSel(rg, `.v-radio-group:nth(${i})`); s += `- **${rg.groupLabel || 'Group ' + (i + 1)}:** ${rg.options.join(', ')}`; if (rg.selected) s += ` — selected: "${rg.selected}"`; s += '\n  Selector: \`page.locator(\'' + sel + '\')\`\n'; }); s += '\n'; }
  if (C.btnToggles?.length) { s += '**Button Toggles (\`.v-btn-toggle\`):**\n'; C.btnToggles.forEach((bt, i) => { const sel = pickSel(bt, `.v-btn-toggle:nth(${i})`); const active = bt.options.find(o => o.active); s += `- **${bt.groupLabel || 'Toggle ' + (i + 1)}:** ${bt.options.map(o => o.text).join(' | ')}`; if (active) s += ` — active: "${active.text}"`; s += '\n  Selector: \`page.locator(\'' + sel + '\')\`\n  To select an option: \`page.locator(\'' + sel + '\').getByText(\'OPTION_TEXT\').click()\`\n'; }); s += '\n'; }
  if (C.fileInputs.length) { s += `**File Inputs:**\n`; C.fileInputs.forEach((fi, i) => { s += `- ${fi.label || `File ${i + 1}`}`; if (fi.accept) s += ` (${fi.accept})`; s += `\n`; }); s += '\n'; }

  // VUE FLOW
  if (C.vueFlow) {
    const vf = C.vueFlow;
    s += '### Vue Flow Canvas\n\n';
    s += 'Container: `page.locator(\'.vue-flow\')`\n\n';

    if (vf.nodes.length) {
      s += `**Nodes (${vf.nodes.length}):**\n\n`;
      s += '| data-id | Type | Label | Icon | Selector |\n|---------|------|-------|------|----------|\n';
      vf.nodes.forEach(n => {
        s += `| ${n.dataId} | ${n.nodeType} | ${n.label} | ${n.icon} | \`page.locator('${n.selector}')\` |\n`;
      });
      s += '\n';
      s += 'To click a node: `await page.locator(\'[data-id="NODE_ID"]\').click();`\n\n';
    }

    if (vf.edges.length) {
      s += `**Edges (${vf.edges.length}):**\n\n`;
      s += '| data-id | Source | Target | Description |\n|---------|--------|--------|-------------|\n';
      vf.edges.forEach(e => {
        s += `| ${e.dataId} | ${e.source} | ${e.target} | ${e.ariaLabel} |\n`;
      });
      s += '\n';
      s += 'To click an edge: `await page.locator(\'[data-id="EDGE_ID"]\').click();`\n\n';
    }

    if (vf.panels.length) {
      s += '**Toolbar Panels:**\n\n';
      vf.panels.forEach(p => {
        if (!p.buttons.length) return;
        s += `- **${p.position}**: `;
        s += p.buttons.map(b => {
          const label = b.text || b.icon || 'icon-button';
          const sel = b.selector || 'button';
          return '`' + label + '` (' + sel + ')';
        }).join(', ');
        s += '\n';
      });
      s += '\n';
    }
  }

  // MONACO EDITORS
  if (C.monacoEditors?.length) {
    s += `**Monaco Code Editors (${C.monacoEditors.length}):**\n\n`;
    C.monacoEditors.forEach((ed, i) => {
      s += `- Editor ${i + 1}: ${ed.width}x${ed.height}px, ${ed.lineCount} visible lines`;
      if (ed.ariaLabel) s += `, aria-label: "${ed.ariaLabel}"`;
      s += '\n';
      s += `  Container: \`${ed.selector}\`\n`;
      s += `  To type into editor: \`await ${ed.textareaSelector}.fill('code here');\` or use \`await ${ed.textareaSelector}.type('code');\`\n`;
    });
    s += '\n';
  }

  const panelTitleSet = new Set((C.expansionPanels || []).map(p => (p.title || '').replace(/\s+/g, ' ').trim()).filter(Boolean));
  const stableBtns = []; const dynBtns = [];
  C.buttons.forEach(btn => {
    const normalized = (btn.text || '').replace(/\s+/g, ' ').trim();
    if (panelTitleSet.has(normalized)) return;
    const n = stripDynamic(btn.text);
    if (n.includes('{DATE}')) dynBtns.push({ ...btn, pattern: n });
    else stableBtns.push(btn);
  });
  if (stableBtns.length) {
    s += `**Buttons (${stableBtns.length}):**\n`;
    const seen = new Set();
    stableBtns.forEach(btn => {
      const dup = seen.has(btn.text); seen.add(btn.text);
      const custom = pickSel(btn, null);
      const sel = custom && !custom.startsWith('.v-') ? `page.locator('${custom}')` : (dup ? `page.locator('button:has-text("${btn.text}")').nth(N)` : `page.locator('button:has-text("${btn.text}")')`);
      s += `- \`${sel}\`${btn.disabled ? ' [disabled]' : ''}${btn.duplicate ? ' *(dup)*' : ''}\n`;
      if (custom && custom !== btn.text) s += `  classes: \`${btn.className}\`\n`;
    });
    s += '\n';
  }
  if (dynBtns.length) { s += `**⚠️ Dynamic Buttons:**\n`; dynBtns.forEach(btn => { s += `- "${btn.text}" → pattern: \`${btn.pattern}\`\n  Use: \`page.locator('button:visible').nth(N)\`\n`; }); s += '\n'; }
  if (C.iconButtons.length) { s += `**Icon Buttons (${C.iconButtons.length}):**\n`; C.iconButtons.forEach((ib, i) => { const desc = ib.ariaLabel || ib.title || ib.icon || `Icon ${i + 1}`; s += `- ${desc}`; if (ib.icon) s += ` (\`${ib.icon}\`)`; if (ib.selector) s += ` → \`${ib.selector}\``; s += `\n`; }); s += '\n'; }
  if (C.links.length) { s += `**Links (${C.links.length}):**\n`; C.links.forEach(l => { s += `- [${l.text}](${l.href}) → \`page.locator('a:has-text("${l.text}")')\`\n`; }); s += '\n'; }

  if (C.tables.length) { C.tables.forEach((tbl, i) => { const sel = pickSel(tbl, null); s += `**Table${C.tables.length > 1 ? ` ${i + 1}` : ''}:**${sel ? ` \`${sel}\`` : ''}\n`; if (tbl.headers.length) s += `- Columns: \`${tbl.headers.join('` | `')}\`\n`; s += `- Rows: ${tbl.rowCount}${tbl.hasPagination ? ' (paginated)' : ''}\n`; if (tbl.sampleRow.length) s += `- Sample: ${tbl.sampleRow.join(' | ')}\n`; s += '\n'; }); }
  if (C.cards.length) { s += `**Cards (${C.cards.length}):**\n`; C.cards.forEach((c, i) => { s += `- **${c.title || `Card ${i + 1}`}**`; if (c.subtitle) s += ` — ${c.subtitle}`; if (c.selector) s += ` → \`${c.selector}\``; s += `\n`; if (c.text) s += `  "${c.text}"\n`; if (c.actions.length) s += `  Actions: ${c.actions.map(a => `\`${a}\``).join(', ')}\n`; }); s += '\n'; }
  if (C.chips.length) { s += `**Chips:** ${C.chips.map(c => `\`${c.text}\``).join(', ')}\n\n`; }
  if (C.lists.length) { C.lists.forEach((l, i) => { s += `**List${C.lists.length > 1 ? ` ${i + 1}` : ''} (${l.totalItems}):**\n`; l.items.forEach(it => { s += `- ${it.title}`; if (it.subtitle) s += ` — ${it.subtitle}`; s += `\n`; }); s += '\n'; }); }

  if (C.tabs.length) { s += `**Tabs:** ${C.tabs.map(t => `\`${t.text}\`${t.isActive ? ' ★' : ''}`).join(', ')}\n- Switch: \`page.locator('.v-tab').nth(INDEX).click()\`\n\n`; }
  if (C.navDrawerItems.length) { s += `**Sidebar (${C.navDrawerItems.length}):**\n`; C.navDrawerItems.forEach(n => { s += `- \`page.locator('a:has-text("${n.text}")')\`${n.isActive ? ' ★' : ''}\n`; }); s += '\n'; }
  if (C.pagination.length) { s += `**Pagination:** ${C.pagination.map(p => `${p.totalPages} pages`).join(', ')}\n\n`; }
  if (C.alerts.length) { s += `**Alerts:**\n`; C.alerts.forEach(a => { s += `- [${a.type}] "${a.text}"\n`; }); s += '\n'; }
  if (C.expansionPanels.length) { s += `**Expansion Panels:**\n`; C.expansionPanels.forEach((p, i) => { const sel = pickSel(p, `.v-expansion-panel-title:nth(${i})`); s += `- **${p.title || `Panel ${i + 1}`}** (${p.isOpen ? 'open' : 'closed'}) → \`page.locator('${sel}').click()\`\n`; }); s += '\n'; }
  if (C.expansionPanels.length) {
    s += `**Expansion Panel Selector Rule:** Use \`.v-expansion-panel-title\` (or \`.crawler-sections .v-expansion-panel-title\`) with text filter. Do NOT use \`button:has-text(...)\` for these section headers.\n\n`;
  }

  // Element registry
  const meaningful = (C.elementRegistry || []).filter(e => e.customClasses.length > 0 || e.id || e.testId);
  if (meaningful.length) {
    s += `**Custom Elements & IDs (${meaningful.length}):**\n\n| Selector | Tag | Classes | Text |\n|----------|-----|---------|------|\n`;
    meaningful.slice(0, 50).forEach(e => {
      const sel = e.testId ? `[data-testid="${e.testId}"]` : e.selector || '';
      s += `| \`${sel}\` | \`${e.tag}\` | \`${e.customClasses.join(' ')}\` | ${(e.text || '').replace(/\|/g, '\\|').slice(0, 60)} |\n`;
    });
    s += '\n';
  }

  // Explored tabs
  if (data.explored?.tabs?.length) { s += `#### Explored Tabs\n\n`; for (const tab of data.explored.tabs) { s += `**Tab "${tab.tabName}" (${tab.tabIndex}):**\n`; s += summarize(tab.components); } }

  // Explored modals
  if (data.explored?.modals?.length) {
    s += `#### Discovered Modals / Dialogs\n\n`;
    for (const m of data.explored.modals) {
      const trigger = m.trigger || '';
      if (/^(Node|Panel|Menu):/.test(trigger)) {
        s += `**Trigger:** ${trigger}\n`;
      } else {
        s += `**Trigger:** \`page.locator('button:has-text("${trigger}")').click()\`\n`;
      }
      if (m.overlaySelector) { s += `**Overlay:** \`page.locator('${m.overlaySelector}')\`\n**Wait:** \`await page.locator('${m.overlaySelector}').waitFor({ state: 'visible', timeout: 10000 })\`\n`; }
      if (m.overlayClassName) s += `**Classes:** \`${m.overlayClassName}\`\n`;
      if (m.menuItems?.length) {
        s += `**Menu items:** ${m.menuItems.map(t => `\`${t}\``).join(', ')}\n`;
        s += `  Pick: \`await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /ITEM_TEXT/ }).click()\`\n`;
      }
      s += '\n' + summarize(m.components, m.overlaySelector);
      if (m.dropdowns?.length) {
        s += `**Dropdowns in modal:**\n`;
        m.dropdowns.forEach(dd => {
          s += `- **"${dd.label}"**: ${dd.options.map(o => `\`${o}\``).join(', ')}\n`;
          s += `  Open: \`page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(${dd.index}).click()\`\n`;
          s += `  Pick: \`await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()\`\n`;
        });
        s += '\n';
      }
      if (m.subExplorations?.length) {
        for (const sub of m.subExplorations) {
          s += `**After clicking "${sub.subTrigger}" inside modal:**\n`;
          s += summarize(sub.components, m.overlaySelector);
          if (sub.dropdowns?.length) {
            s += `**Dropdowns after "${sub.subTrigger}":**\n`;
            sub.dropdowns.forEach(dd => {
              s += `- **"${dd.label}"**: ${dd.options.map(o => `\`${o}\``).join(', ')}\n`;
              s += `  Open: \`page.locator('.v-select:visible,.v-autocomplete:visible,.v-combobox:visible').nth(${dd.index}).click()\`\n`;
              s += `  Pick: \`await page.locator('.v-overlay--active .v-list-item').filter({ hasText: /OPTION/ }).click()\`\n`;
            });
            s += '\n';
          }
          if (sub.panels?.length) {
            s += `**Expansion Panels after "${sub.subTrigger}":**\n`;
            for (const p of sub.panels) {
              s += `- **"${p.panelTitle}"** (index ${p.panelIndex})\n`;
              s += `  Open: \`page.locator('.v-expansion-panel-title').filter({ hasText: /${p.panelTitle}/ }).click()\`\n`;
              s += summarize(p.components, m.overlaySelector);
            }
            s += '\n';
          }
        }
      }
    }
  }

  // Explored panels
  if (data.explored?.panels?.length) { s += `#### Explored Expansion Panels\n\n`; for (const p of data.explored.panels) { s += `**Panel "${p.panelTitle}" (${p.panelIndex}):**\n`; s += summarize(p.components); } }

  s += '---\n\n';
  return s;
}

function summarize(C, overlaySel) {
  let s = '';
  const panelTitleSet = new Set((C.expansionPanels || []).map(p => (p.title || '').replace(/\s+/g, ' ').trim()).filter(Boolean));
  (C.dialogs || []).forEach(d => {
    if (d.selector) s += `- Container: \`${d.selector}\` (classes: \`${d.className}\`)\n`;
    if (d.title) s += `- Title: "${d.title}"\n`;
    if (d.inputs?.length) { s += `- Inputs:\n`; d.inputs.forEach(inp => { s += `  - ${inp.label || 'unlabeled'} (\`${inp.type}\`)\n`; }); }
    const dialogButtons = (d.buttons || []).filter(b => {
      const normalized = (b || '').replace(/\s+/g, ' ').trim();
      return !panelTitleSet.has(normalized);
    });
    if (dialogButtons.length) s += `- Buttons: ${dialogButtons.map(b => `\`${b}\``).join(', ')}\n`;
  });
  const extra = [];
  const fi = (C.inputs || []).filter(i => !i.inSelect);
  if (fi.length) extra.push(`${fi.length} input(s): ${fi.map(i => i.label || i.type).join(', ')}`);
  if (C.textareas?.length) extra.push(`${C.textareas.length} textarea(s)`);
  if (C.selects?.length) extra.push(`${C.selects.length} dropdown(s): ${C.selects.map(s => s.label || '?').join(', ')}`);
  if (C.radios?.length) extra.push(`radio group(s): ${C.radios.map(r => `${r.groupLabel || '?'} [${r.options.join(', ')}]`).join('; ')}`);
  if (C.btnToggles?.length) extra.push(`toggle(s): ${C.btnToggles.map(t => `${t.groupLabel || '?'} [${t.options.map(o => o.active ? `**${o.text}**` : o.text).join(' | ')}]`).join('; ')}`);
  if (C.checkboxes?.length) extra.push(`${C.checkboxes.length} checkbox(es)`);
  if (C.switches?.length) extra.push(`${C.switches.length} switch(es)`);
  if (C.tables?.length) extra.push(`${C.tables.length} table(s)`);
  if (C.cards?.length) extra.push(`${C.cards.length} card(s)`);
  if (C.vueFlow) extra.push(`Vue Flow: ${C.vueFlow.nodes.length} node(s), ${C.vueFlow.edges.length} edge(s)`);
  if (C.monacoEditors?.length) extra.push(`${C.monacoEditors.length} Monaco editor(s)`);
  if (C.headings?.length) extra.push(`headings: ${C.headings.map(h => h.text).join(', ')}`);
  const sb = (C.buttons || []).filter(b => {
    const normalized = (b.text || '').replace(/\s+/g, ' ').trim();
    return !stripDynamic(b.text).includes('{DATE}') && !panelTitleSet.has(normalized);
  });
  if (sb.length) extra.push(`buttons: ${sb.map(b => b.text).join(', ')}`);
  const ce = (C.elementRegistry || []).filter(e => e.customClasses.length > 0);
  if (ce.length) extra.push(`custom: ${ce.slice(0, 10).map(e => e.selector || e.customClasses[0]).join(', ')}`);
  extra.forEach(e => { s += `- ${e}\n`; });
  s += '\n';
  return s;
}

// ---------------------------------------------------------------------------
// High-level API
// ---------------------------------------------------------------------------
async function runPageInspection({ path: pagePath, name, needsLogin = false, skillFolder }) {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  try {
    if (needsLogin) { const ok = await doLogin(page); if (!ok) { console.log('❌ Login failed'); process.exit(1); } }
    const data = await inspectPage(page, pagePath, name, !needsLogin);
    if (data.authFailed) { console.log('❌ Auth failed'); process.exit(1); }
    const wsRoot = pathLib.resolve(__dirname, '..', '..');
    const folder = skillFolder || `app-selectors-${slugify(name)}`;
    const dir = pathLib.join(wsRoot, 'skills', folder);
    fs.mkdirSync(dir, { recursive: true });
    const content = generatePageSkillMd(data, name);
    fs.writeFileSync(pathLib.join(dir, 'SKILL.md'), content);
    console.log(`\n✅ ${folder}/SKILL.md written (${(content.length / 1024).toFixed(1)} KB)`);
    if (needsLogin) {
      const loginDir = pathLib.join(wsRoot, 'skills', 'app-selectors');
      fs.mkdirSync(loginDir, { recursive: true });
      fs.writeFileSync(pathLib.join(loginDir, 'SKILL.md'), generateLoginSkillMd());
    }
  } finally { await browser.close(); }
}

async function runAllInspections() {
  const PUBLIC = [
    { path: '/login', name: 'Login Page' }, { path: '/signup', name: 'Sign Up Page' },
    { path: '/forgot-password', name: 'Forgot Password Page' }, { path: '/auth/set-password', name: 'Set Password Page' },
    { path: '/register', name: 'Register Page' },
  ];
  const PROTECTED = [
    { path: '/', name: 'Dashboard' }, { path: '/manage-chatbot', name: 'Manage Chatbot' },
    { path: '/config-test', name: 'Config Test' }, { path: '/flow-designer', name: 'Flow Designer' }, { path: '/flow-designer/206', name: 'Flow Designer Canvas' },
    { path: '/knowledge-management', name: 'Knowledge Management' }, { path: '/knowledge-base', name: 'Knowledge Base' },
    { path: '/sentiment', name: 'Sentiment Dashboard' }, { path: '/organization', name: 'Organization' },
    { path: '/profile', name: 'Profile' }, { path: '/change-email', name: 'Change Email' },
    { path: '/change-password', name: 'Change Password' }, { path: '/settings', name: 'Settings' },
    { path: '/logs', name: 'Logs' },
    { path: '/add-ons', name: 'Add-Ons' },
    { path: '/add-ons/webchat-widget', name: 'Add-Ons Webchat Widget' },
    { path: '/add-ons/webchat-widget/install', name: 'Add-Ons Webchat Widget Install' },
    { path: '/add-ons/twilio', name: 'Add-Ons Twilio Detail' },
    { path: '/add-ons/twilio/install', name: 'Add-Ons Twilio Install' },
    { path: '/flow-tester', name: 'Flow Tester' },
  ];

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const wsRoot = pathLib.resolve(__dirname, '..', '..');

  console.log('\n' + '='.repeat(70) + '\nPUBLIC PAGES\n' + '='.repeat(70));
  for (const { path: p, name } of PUBLIC) {
    const data = await inspectPage(page, p, name, true);
    if (!data.authFailed) {
      const dir = pathLib.join(wsRoot, 'skills', `app-selectors-${slugify(name)}`);
      fs.mkdirSync(dir, { recursive: true });
      fs.writeFileSync(pathLib.join(dir, 'SKILL.md'), generatePageSkillMd(data, name));
    }
  }

  console.log('\n' + '='.repeat(70) + '\nLOGIN\n' + '='.repeat(70));
  const ok = await doLogin(page);
  if (!ok) { console.log('❌ Login failed'); await browser.close(); process.exit(1); }
  const loginDir = pathLib.join(wsRoot, 'skills', 'app-selectors');
  fs.mkdirSync(loginDir, { recursive: true });
  fs.writeFileSync(pathLib.join(loginDir, 'SKILL.md'), generateLoginSkillMd());

  console.log('\n' + '='.repeat(70) + '\nPROTECTED PAGES\n' + '='.repeat(70));
  for (const { path: p, name } of PROTECTED) {
    const data = await inspectPage(page, p, name, false);
    if (data.authFailed) { console.log('  🔄 Re-logging in...'); await doLogin(page); continue; }
    const dir = pathLib.join(wsRoot, 'skills', `app-selectors-${slugify(name)}`);
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(pathLib.join(dir, 'SKILL.md'), generatePageSkillMd(data, name));
  }

  await browser.close();
  console.log('\n✅ All done!\n');
}

module.exports = { runPageInspection, runAllInspections, BASE_URL, LOGIN_EMAIL, LOGIN_PASSWORD, ORG_NAME };
