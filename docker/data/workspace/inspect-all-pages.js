const { chromium } = require('playwright');
const fs = require('fs');

const PUBLIC_PAGES = [
  { path: '/login', name: 'Login Page' },
  { path: '/signup', name: 'Sign Up Page' },
  { path: '/forgot-password', name: 'Forgot Password Page' },
  { path: '/auth/set-password', name: 'Set Password Page' },
  { path: '/register', name: 'Register Page' }
];

const PROTECTED_PAGES = [
  { path: '/', name: 'Dashboard' },
  { path: '/manage-chatbot', name: 'Manage Chatbot' },
  { path: '/config-test', name: 'Config Test' },
  { path: '/flow-designer', name: 'Flow Designer' },
  { path: '/knowledge-management', name: 'Knowledge Management' },
  { path: '/knowledge-base', name: 'Knowledge Base' },
  { path: '/sentiment', name: 'Sentiment Dashboard' },
  { path: '/organization', name: 'Organization' },
  { path: '/profile', name: 'Profile' },
  { path: '/change-email', name: 'Change Email' },
  { path: '/change-password', name: 'Change Password' },
  { path: '/settings', name: 'Settings' },
  { path: '/logs', name: 'Logs' },
  { path: '/add-ons', name: 'Add-Ons' },
  { path: '/flow-tester', name: 'Flow Tester' }
];

const BASE_URL = 'https://dashboard.int3nt.info';
const LOGIN_EMAIL = 'heidi@intnt.ai';
const LOGIN_PASSWORD = 'testing2026!';
const ORG_NAME = 'Testing2026!';

const DESTRUCTIVE_PATTERNS = /delete|remove|confirm|save|submit|export|download|logout|sign\s*out|sign\s*up|sign\s*in|discard|back\s+to|login|log\s*in|send\s+email|set\s+password|reset|forgot|cancel/i;
const MODAL_TRIGGER_PATTERNS = /add|create|new|edit|invite|upload|import|schedule|filter|sort|column|show|manage|configure|change|select version/i;
const MAX_MODAL_EXPLORATIONS = 10;

function stripDynamic(text) {
  return text
    .replace(/\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},\s+\d{4}/g, '{DATE}')
    .replace(/\{DATE\}\s*[-–]\s*\{DATE\}/g, '{DATE_RANGE}')
    .replace(/\b\d{4}-\d{2}-\d{2}\b/g, '{DATE}')
    .trim();
}

// ---------------------------------------------------------------------------
// Comprehensive DOM scan — runs inside the browser context
// ---------------------------------------------------------------------------
async function extractAllComponents(page) {
  return await page.evaluate(() => {
    function isVisible(el) {
      if (!el) return false;
      const rect = el.getBoundingClientRect();
      if (rect.width === 0 || rect.height === 0) return false;
      const style = getComputedStyle(el);
      return style.display !== 'none' && style.visibility !== 'hidden' && parseFloat(style.opacity) > 0;
    }

    function getText(el) {
      return el?.textContent?.trim().replace(/\s+/g, ' ') || '';
    }

    function getAriaLabel(el) {
      return el?.getAttribute('aria-label') || null;
    }

    function getTestId(el) {
      return el?.getAttribute('data-testid') || el?.getAttribute('data-test') || null;
    }

    function getInputLabel(input) {
      let label = null;
      const vField = input.closest('.v-field');
      if (vField) {
        const fl = vField.querySelector('.v-field__label');
        if (fl) label = getText(fl);
      }
      if (!label) {
        const vInput = input.closest('.v-input');
        if (vInput) {
          const vl = vInput.querySelector('.v-label');
          if (vl) label = getText(vl);
        }
      }
      if (!label && input.id) {
        const lbl = document.querySelector(`label[for="${input.id}"]`);
        if (lbl) label = getText(lbl);
      }
      if (!label) {
        const wrap = input.closest('label');
        if (wrap) label = getText(wrap).replace(input.value || '', '').trim();
      }
      if (!label) label = input.getAttribute('aria-label') || input.placeholder || null;
      if (!label) {
        const container = input.closest('.v-input') || input.parentElement;
        if (container) {
          const prev = container.previousElementSibling;
          if (prev && getText(prev).length < 60) label = getText(prev);
        }
      }
      return label ? label.replace(/\*/g, '').trim() : null;
    }

    const C = {
      headings: [],
      staticTexts: [],
      breadcrumbs: [],
      inputs: [],
      textareas: [],
      selects: [],
      checkboxes: [],
      switches: [],
      radios: [],
      fileInputs: [],
      buttons: [],
      iconButtons: [],
      links: [],
      tables: [],
      cards: [],
      chips: [],
      tabs: [],
      lists: [],
      alerts: [],
      snackbars: [],
      progressBars: [],
      dialogs: [],
      bottomSheets: [],
      menus: [],
      expansionPanels: [],
      navDrawerItems: [],
      toolbarItems: [],
      pagination: [],
      images: [],
      avatarCount: 0,
      badges: [],
      hasDividers: false,
      tooltipTriggers: [],
    };

    // ── HEADINGS ──
    const seenHeadingText = new Set();
    document.querySelectorAll('h1, h2, h3, h4, h5, h6, .text-h1, .text-h2, .text-h3, .text-h4, .text-h5, .text-h6').forEach(el => {
      if (!isVisible(el)) return;
      const text = getText(el);
      if (!text || text.length > 200 || seenHeadingText.has(text)) return;
      seenHeadingText.add(text);
      const tag = el.tagName.toLowerCase();
      const vuetifyClass = Array.from(el.classList).find(c => /^text-h\d$/.test(c)) || '';
      C.headings.push({ tag: tag.match(/^h\d$/) ? tag : vuetifyClass, text });
    });

    // ── STATIC TEXTS ──
    const seenStaticText = new Set();
    const textSelectors = '.v-card-text, .v-card-subtitle, .v-card-title, .v-list-subheader, p, .text-body-1, .text-body-2, .text-subtitle-1, .text-subtitle-2, .text-caption, .text-overline, .v-alert__content, .v-banner__text, span.text-medium-emphasis, .v-toolbar-title, .v-empty-state__headline, .v-empty-state__text';
    document.querySelectorAll(textSelectors).forEach(el => {
      if (!isVisible(el)) return;
      const text = getText(el);
      if (!text || text.length < 3 || text.length > 500) return;
      if (seenHeadingText.has(text) || seenStaticText.has(text)) return;
      seenStaticText.add(text);
      const cls = Array.from(el.classList).find(c => c.startsWith('text-') || c.startsWith('v-')) || el.tagName.toLowerCase();
      C.staticTexts.push({ type: cls, text });
    });

    // ── BREADCRUMBS ──
    document.querySelectorAll('.v-breadcrumbs-item').forEach(el => {
      if (!isVisible(el)) return;
      const t = getText(el);
      if (t) C.breadcrumbs.push(t);
    });

    // ── INPUTS ──
    document.querySelectorAll('input').forEach((input, i) => {
      if (input.type === 'hidden' || input.type === 'file') return;
      if (!isVisible(input)) return;
      C.inputs.push({
        index: i,
        type: input.type || 'text',
        label: getInputLabel(input),
        placeholder: input.placeholder || null,
        ariaLabel: getAriaLabel(input),
        testId: getTestId(input),
        disabled: input.disabled,
        readOnly: input.readOnly,
        inSelect: !!input.closest('.v-select, .v-autocomplete, .v-combobox'),
      });
    });

    // ── TEXTAREAS ──
    document.querySelectorAll('textarea').forEach((ta, i) => {
      if (!isVisible(ta)) return;
      C.textareas.push({
        index: i,
        label: getInputLabel(ta),
        placeholder: ta.placeholder || null,
        ariaLabel: getAriaLabel(ta),
        rows: ta.rows,
        disabled: ta.disabled,
      });
    });

    // ── FILE INPUTS ──
    document.querySelectorAll('input[type="file"]').forEach((fi, i) => {
      C.fileInputs.push({ index: i, label: getInputLabel(fi) || getAriaLabel(fi), accept: fi.accept || null, multiple: fi.multiple });
    });

    // ── SELECTS / DROPDOWNS ──
    document.querySelectorAll('.v-select, .v-autocomplete, .v-combobox').forEach((sel, i) => {
      if (!isVisible(sel)) return;
      const vInput = sel.closest('.v-input');
      const label = vInput?.querySelector('.v-label, .v-field__label')?.textContent?.trim() || null;
      const value = sel.querySelector('.v-select__selection-text, .v-autocomplete__selection')?.textContent?.trim() || null;
      const type = sel.classList.contains('v-autocomplete') ? 'autocomplete' : sel.classList.contains('v-combobox') ? 'combobox' : 'select';
      C.selects.push({ index: i, type, label, currentValue: value, disabled: sel.classList.contains('v-input--disabled') });
    });

    // ── CHECKBOXES ──
    document.querySelectorAll('.v-checkbox, input[type="checkbox"]').forEach((el, i) => {
      const wrapper = el.closest('.v-checkbox') || el.closest('.v-selection-control');
      const input = el.tagName === 'INPUT' ? el : el.querySelector('input[type="checkbox"]');
      if (!input || !isVisible(wrapper || el)) return;
      const label = wrapper?.querySelector('.v-label')?.textContent?.trim() || getAriaLabel(input);
      C.checkboxes.push({ index: i, label, checked: input.checked, disabled: input.disabled });
    });

    // ── SWITCHES ──
    document.querySelectorAll('.v-switch').forEach((sw, i) => {
      if (!isVisible(sw)) return;
      const label = sw.querySelector('.v-label')?.textContent?.trim() || null;
      const input = sw.querySelector('input');
      C.switches.push({ index: i, label, checked: input?.checked || false, disabled: sw.classList.contains('v-input--disabled') });
    });

    // ── RADIO GROUPS ──
    document.querySelectorAll('.v-radio-group').forEach((rg, i) => {
      if (!isVisible(rg)) return;
      const groupLabel = rg.querySelector(':scope > .v-label, :scope > .v-input__control > .v-label')?.textContent?.trim() || null;
      const options = Array.from(rg.querySelectorAll('.v-radio .v-label, .v-selection-control .v-label')).map(l => l.textContent.trim()).filter(Boolean);
      const selected = rg.querySelector('.v-selection-control--checked .v-label')?.textContent?.trim() || null;
      if (options.length) C.radios.push({ index: i, groupLabel, options, selected });
    });

    // ── BUTTONS ──
    const seenBtnText = new Set();
    document.querySelectorAll('button, [role="button"]').forEach((btn, i) => {
      if (!isVisible(btn)) return;
      const text = getText(btn);
      const ariaLabel = getAriaLabel(btn);
      const hasIcon = !!btn.querySelector('.v-icon, .mdi, i[class*="mdi"]');

      if ((!text || text.length === 0) && hasIcon) {
        const iconEl = btn.querySelector('.v-icon, [class*="mdi-"]');
        const iconClass = iconEl ? Array.from(iconEl.classList).find(c => c.startsWith('mdi-')) || '' : '';
        C.iconButtons.push({ index: i, icon: iconClass, ariaLabel, title: btn.title || null, disabled: btn.disabled });
      } else if (text && text.length > 0 && text.length < 200) {
        const key = text.replace(/\s+/g, ' ');
        const isDuplicate = seenBtnText.has(key);
        seenBtnText.add(key);
        C.buttons.push({
          text: key,
          ariaLabel,
          testId: getTestId(btn),
          disabled: btn.disabled,
          duplicate: isDuplicate,
        });
      }
    });

    // ── LINKS (outside nav drawer) ──
    const seenLinks = new Set();
    document.querySelectorAll('a[href]').forEach(a => {
      if (!isVisible(a)) return;
      if (a.closest('.v-navigation-drawer')) return;
      const text = getText(a);
      const href = a.getAttribute('href');
      if (!text || text.length > 200) return;
      const key = `${text}|${href}`;
      if (seenLinks.has(key)) return;
      seenLinks.add(key);
      C.links.push({ text, href });
    });

    // ── TABLES ──
    document.querySelectorAll('table, .v-data-table, .v-table').forEach((table, i) => {
      if (!isVisible(table)) return;
      const headers = Array.from(table.querySelectorAll('th')).map(th => getText(th)).filter(Boolean);
      const rows = table.querySelectorAll('tbody tr');
      let sampleRow = [];
      if (rows.length > 0) {
        sampleRow = Array.from(rows[0].querySelectorAll('td')).map(td => {
          const t = getText(td);
          return t.length > 60 ? t.slice(0, 57) + '...' : t;
        });
      }
      if (headers.length === 0 && rows.length === 0) return;
      C.tables.push({ index: i, headers, rowCount: rows.length, sampleRow, hasPagination: !!table.querySelector('.v-data-table-footer, .v-pagination') });
    });

    // ── CARDS ──
    document.querySelectorAll('.v-card').forEach((card, i) => {
      if (!isVisible(card)) return;
      if (card.closest('.v-dialog, .v-navigation-drawer, .v-menu__content, .v-bottom-sheet')) return;
      const title = card.querySelector('.v-card-title')?.textContent?.trim() || null;
      const subtitle = card.querySelector('.v-card-subtitle')?.textContent?.trim() || null;
      const cardText = card.querySelector('.v-card-text')?.textContent?.trim() || null;
      const actions = Array.from(card.querySelectorAll('.v-card-actions button, .v-card-actions .v-btn')).map(b => getText(b)).filter(Boolean);
      if (!title && !subtitle && !cardText && actions.length === 0) return;
      C.cards.push({
        index: i,
        title,
        subtitle,
        text: cardText && cardText.length > 200 ? cardText.slice(0, 197) + '...' : cardText,
        actions,
      });
    });

    // ── CHIPS ──
    document.querySelectorAll('.v-chip').forEach((chip, i) => {
      if (!isVisible(chip)) return;
      const text = getText(chip);
      if (!text) return;
      C.chips.push({ index: i, text, closable: !!chip.querySelector('.v-chip__close') });
    });

    // ── TABS ──
    document.querySelectorAll('.v-tab').forEach((tab, i) => {
      if (!isVisible(tab)) return;
      const text = getText(tab);
      const isActive = tab.classList.contains('v-tab--selected') || tab.getAttribute('aria-selected') === 'true';
      C.tabs.push({ index: i, text, isActive });
    });

    // ── LISTS (standalone, not inside nav/menus) ──
    document.querySelectorAll('.v-list').forEach((list, i) => {
      if (!isVisible(list)) return;
      if (list.closest('.v-navigation-drawer, .v-menu__content, .v-select__content, .v-autocomplete__content')) return;
      const items = Array.from(list.querySelectorAll('.v-list-item')).map(item => {
        const title = item.querySelector('.v-list-item-title')?.textContent?.trim() || getText(item);
        const subtitle = item.querySelector('.v-list-item-subtitle')?.textContent?.trim() || null;
        return { title, subtitle };
      }).filter(it => it.title);
      if (items.length === 0) return;
      C.lists.push({ index: i, items: items.slice(0, 25), totalItems: items.length });
    });

    // ── ALERTS ──
    document.querySelectorAll('.v-alert').forEach((alert, i) => {
      if (!isVisible(alert)) return;
      const text = getText(alert);
      const type = ['error', 'success', 'warning', 'info'].find(t => alert.classList.toString().includes(t)) || 'default';
      C.alerts.push({ index: i, type, text: text.slice(0, 300) });
    });

    // ── SNACKBARS ──
    document.querySelectorAll('.v-snackbar').forEach((sb, i) => {
      if (!isVisible(sb)) return;
      C.snackbars.push({ index: i, text: getText(sb).slice(0, 300) });
    });

    // ── PROGRESS BARS ──
    document.querySelectorAll('.v-progress-linear, .v-progress-circular').forEach((pb, i) => {
      if (!isVisible(pb)) return;
      const type = pb.classList.contains('v-progress-circular') ? 'circular' : 'linear';
      C.progressBars.push({ index: i, type });
    });

    // ── VISIBLE OVERLAYS: dialogs, drawers, sheets, custom panels ──
    // Broad scan — catches Vuetify AND custom overlay components
    const overlaySelectors = [
      '.v-overlay--active .v-dialog', '.v-dialog--active',
      '.v-bottom-sheet',
      '[class*="drawer"]:not(.v-navigation-drawer)',
      '[class*="modal"]', '[class*="sheet"]', '[class*="popup"]',
      '[class*="sidebar"]:not(.v-navigation-drawer)',
      '.v-overlay--active > .v-overlay__content > *',
    ];
    const seenOverlays = new Set();
    document.querySelectorAll(overlaySelectors.join(', ')).forEach((el, i) => {
      if (!isVisible(el)) return;
      const rect = el.getBoundingClientRect();
      if (rect.width < 50 || rect.height < 50) return;
      const cls = el.className?.toString?.() || '';
      if (seenOverlays.has(cls)) return;
      seenOverlays.add(cls);

      const content = el.querySelector('.v-card, .v-sheet, [class*="content"]') || el;
      const title = content.querySelector('.v-card-title, .v-toolbar-title, h1, h2, h3')?.textContent?.trim() || null;
      const bodyText = content.querySelector('.v-card-text, .v-card__text')?.textContent?.trim() || null;
      const inputs = Array.from(el.querySelectorAll('input:not([type="hidden"])')).filter(isVisible).map(inp => ({
        type: inp.type, label: getInputLabel(inp) || inp.placeholder || null,
      }));
      const buttons = Array.from(el.querySelectorAll('button')).filter(isVisible).map(b => getText(b)).filter(Boolean);

      const classes = cls.split(/\s+/).filter(c => c);
      const customClass = classes.find(c =>
        c.length > 2 &&
        !c.match(/^(v-|mdi-|d-|ma-|pa-|mt-|mb-|ml-|mr-|mx-|my-|px-|py-|text-|font-|bg-|rounded|elevation)/)
      );

      C.dialogs.push({
        index: i,
        title,
        bodyText: bodyText?.slice(0, 300) || null,
        inputs,
        buttons,
        className: cls,
        id: el.id || null,
        selector: el.id ? `#${el.id}` : customClass ? `.${customClass}` : classes[0] ? `.${classes[0]}` : null,
      });
    });

    // ── MENUS (visible) ──
    document.querySelectorAll('.v-overlay--active .v-list').forEach((menu, i) => {
      const overlay = menu.closest('.v-overlay');
      if (!overlay || overlay.closest('.v-dialog, .v-bottom-sheet, .v-navigation-drawer, [class*="drawer"]')) return;
      if (!isVisible(menu)) return;
      const items = Array.from(menu.querySelectorAll('.v-list-item')).map(item => getText(item)).filter(Boolean);
      if (items.length) C.menus.push({ index: i, items });
    });

    // ── EXPANSION PANELS ──
    document.querySelectorAll('.v-expansion-panel').forEach((panel, i) => {
      if (!isVisible(panel)) return;
      const title = panel.querySelector('.v-expansion-panel-title')?.textContent?.trim() || null;
      const isOpen = panel.classList.contains('v-expansion-panel--active');
      const content = isOpen ? panel.querySelector('.v-expansion-panel-text')?.textContent?.trim() : null;
      C.expansionPanels.push({ index: i, title, isOpen, contentPreview: content?.slice(0, 200) || null });
    });

    // ── NAVIGATION DRAWER ──
    document.querySelectorAll('.v-navigation-drawer .v-list-item, nav .v-list-item').forEach(item => {
      if (!isVisible(item)) return;
      const text = item.querySelector('.v-list-item-title')?.textContent?.trim() || getText(item);
      const href = item.getAttribute('href') || item.querySelector('a')?.getAttribute('href') || null;
      const isActive = item.classList.contains('v-list-item--active');
      if (text) C.navDrawerItems.push({ text, href, isActive });
    });
    const seenNav = new Set();
    C.navDrawerItems = C.navDrawerItems.filter(n => { if (seenNav.has(n.text)) return false; seenNav.add(n.text); return true; });

    // ── TOOLBAR / APP BAR ──
    document.querySelectorAll('.v-app-bar, .v-toolbar:not(.v-app-bar .v-toolbar)').forEach(tb => {
      if (!isVisible(tb)) return;
      const title = tb.querySelector('.v-toolbar-title')?.textContent?.trim() || null;
      const btns = Array.from(tb.querySelectorAll('button, .v-btn')).map(b => getText(b) || getAriaLabel(b)).filter(Boolean);
      C.toolbarItems.push({ title, buttons: btns });
    });

    // ── PAGINATION ──
    document.querySelectorAll('.v-pagination').forEach((pg, i) => {
      if (!isVisible(pg)) return;
      C.pagination.push({ index: i, totalPages: pg.querySelectorAll('.v-pagination__item').length });
    });

    // ── IMAGES ──
    document.querySelectorAll('img').forEach(img => {
      if (!isVisible(img)) return;
      const alt = img.alt || null;
      if (!alt) return;
      C.images.push({ alt });
    });

    // ── AVATARS (count) ──
    C.avatarCount = Array.from(document.querySelectorAll('.v-avatar')).filter(isVisible).length;

    // ── BADGES ──
    document.querySelectorAll('.v-badge').forEach(badge => {
      if (!isVisible(badge)) return;
      const content = badge.querySelector('.v-badge__badge')?.textContent?.trim() || null;
      if (content) C.badges.push({ content });
    });

    // ── DIVIDERS ──
    C.hasDividers = Array.from(document.querySelectorAll('.v-divider')).some(isVisible);

    // ── TOOLTIP TRIGGERS ──
    document.querySelectorAll('[title]:not(iframe)').forEach(el => {
      if (!isVisible(el)) return;
      const tip = el.title;
      if (tip && tip.length < 200) C.tooltipTriggers.push({ tooltip: tip, tag: el.tagName.toLowerCase() });
    });
    C.tooltipTriggers = C.tooltipTriggers.slice(0, 20);

    // ── ELEMENT REGISTRY: all visible elements with custom classes, IDs, or test IDs ──
    const registry = [];
    const seenReg = new Set();
    document.querySelectorAll('*').forEach(el => {
      if (!isVisible(el)) return;
      const cls = el.className?.toString?.() || '';
      const id = el.id || '';
      const testId = el.getAttribute('data-testid') || el.getAttribute('data-test') || '';
      const role = el.getAttribute('role') || '';

      const classes = cls.split(/\s+/).filter(c => c);
      const customClasses = classes.filter(c =>
        c.length > 2 &&
        !c.match(/^(v-|mdi-|d-|ma-|pa-|mt-|mb-|ml-|mr-|mx-|my-|px-|py-|pt-|pb-|pl-|pr-|ga-|text-|font-|bg-|rounded|elevation|float-|position-|overflow-|align-|justify-|flex-|order-|col-|row-|container|spacer|fill-|w-|h-|min-|max-|gap-|border|opacity|cursor|transition|--v-)/)
      );

      if (!customClasses.length && !id && !testId) return;

      const key = `${el.tagName}|${customClasses.join(' ')}|${id}`;
      if (seenReg.has(key)) return;
      seenReg.add(key);

      const rect = el.getBoundingClientRect();
      if (rect.width < 5 || rect.height < 5) return;

      const tag = el.tagName.toLowerCase();
      const text = getText(el);

      registry.push({
        tag,
        customClasses,
        allClasses: cls,
        id: id || null,
        testId: testId || null,
        role: role || null,
        selector: id ? `#${id}` : customClasses[0] ? `.${customClasses[0]}` : null,
        text: text.length > 100 ? text.slice(0, 97) + '...' : (text || null),
        size: `${Math.round(rect.width)}x${Math.round(rect.height)}`,
      });
    });
    C.elementRegistry = registry;

    return C;
  });
}

// ---------------------------------------------------------------------------
// Exploration helpers — use Playwright API to reveal hidden UI
// ---------------------------------------------------------------------------

async function scrollFullPage(page) {
  await page.evaluate(async () => {
    const target = document.querySelector('.v-main') || document.documentElement;
    const total = target.scrollHeight;
    const step = window.innerHeight;
    for (let y = 0; y < total; y += step) {
      target.scrollTo(0, y);
      await new Promise(r => setTimeout(r, 300));
    }
    target.scrollTo(0, 0);
  });
  await page.waitForTimeout(800);
}

async function exploreTabs(page) {
  const results = [];
  const tabCount = await page.locator('.v-tab:visible').count();
  if (tabCount <= 1) return results;

  for (let i = 0; i < tabCount; i++) {
    try {
      const tab = page.locator('.v-tab:visible').nth(i);
      const tabText = (await tab.textContent()).trim();
      const isActive = await tab.evaluate(el =>
        el.classList.contains('v-tab--selected') || el.getAttribute('aria-selected') === 'true'
      );
      if (!isActive) {
        await tab.click();
        await page.waitForTimeout(1500);
      }
      const components = await extractAllComponents(page);
      results.push({ tabName: tabText, tabIndex: i, components });
    } catch (e) { /* tab click failed */ }
  }
  return results;
}

async function snapshotOverlays(page) {
  return await page.evaluate(() => {
    const overlays = [];
    document.querySelectorAll('*').forEach(el => {
      const rect = el.getBoundingClientRect();
      if (rect.width < 80 || rect.height < 50) return;
      const style = getComputedStyle(el);
      if (style.display === 'none' || style.visibility === 'hidden' || parseFloat(style.opacity) === 0) return;
      const z = parseInt(style.zIndex) || 0;
      const cls = (el.className?.toString?.() || '');
      const clsLower = cls.toLowerCase();
      const isOverlayLike =
        (style.position === 'fixed' && z >= 4 && rect.width > 100) ||
        /dialog|drawer|modal|sheet|overlay--active|popup|sidebar/.test(clsLower);
      if (!isOverlayLike) return;
      overlays.push({ className: cls, id: el.id || '', tag: el.tagName });
    });
    return overlays;
  });
}

async function identifyNewOverlay(page, beforeSet) {
  return await page.evaluate((beforeKeys) => {
    const bSet = new Set(beforeKeys);
    const results = [];
    document.querySelectorAll('*').forEach(el => {
      const rect = el.getBoundingClientRect();
      if (rect.width < 80 || rect.height < 50) return;
      const style = getComputedStyle(el);
      if (style.display === 'none' || style.visibility === 'hidden' || parseFloat(style.opacity) === 0) return;
      const z = parseInt(style.zIndex) || 0;
      const cls = (el.className?.toString?.() || '');
      const clsLower = cls.toLowerCase();
      const isOverlayLike =
        (style.position === 'fixed' && z >= 4 && rect.width > 100) ||
        /dialog|drawer|modal|sheet|overlay--active|popup|sidebar/.test(clsLower);
      if (!isOverlayLike) return;
      const key = `${el.tagName}|${cls}|${el.id || ''}`;
      if (bSet.has(key)) return;

      const classes = cls.split(/\s+/).filter(c => c);
      const customClass = classes.find(c =>
        c.length > 2 &&
        !c.match(/^(v-|mdi-|d-|ma-|pa-|mt-|mb-|ml-|mr-|mx-|my-|px-|py-|text-|font-|bg-|rounded|elevation|--v-)/)
      );

      // Walk children to find the real content container if this is a Vuetify wrapper
      let contentSelector = null;
      let contentClassName = null;
      if (!customClass) {
        const children = el.querySelectorAll('*');
        for (const child of children) {
          const childCls = child.className?.toString?.() || '';
          const childClasses = childCls.split(/\s+/).filter(c => c);
          const cc = childClasses.find(c =>
            c.length > 2 &&
            !c.match(/^(v-|mdi-|d-|ma-|pa-|mt-|mb-|ml-|mr-|mx-|my-|px-|py-|text-|font-|bg-|rounded|elevation|--v-)/)
          );
          if (cc) {
            const childRect = child.getBoundingClientRect();
            if (childRect.width > 80 && childRect.height > 50) {
              contentSelector = child.id ? `#${child.id}` : `.${cc}`;
              contentClassName = childCls;
              break;
            }
          }
        }
      }

      results.push({
        tag: el.tagName.toLowerCase(),
        className: cls,
        id: el.id || null,
        selector: contentSelector || (el.id ? `#${el.id}` : customClass ? `.${customClass}` : classes[0] ? `.${classes[0]}` : el.tagName.toLowerCase()),
        contentClassName: contentClassName || cls,
        allClasses: cls,
      });
    });

    // Prefer the result that has a custom class selector (most specific)
    results.sort((a, b) => {
      const aCustom = !a.selector.startsWith('.v-');
      const bCustom = !b.selector.startsWith('.v-');
      if (aCustom && !bCustom) return -1;
      if (!aCustom && bCustom) return 1;
      return 0;
    });

    return results.length ? results[0] : null;
  }, beforeSet);
}

async function exploreModals(page, currentUrl) {
  const results = [];
  const allButtons = await page.locator('button:visible, [role="button"]:visible').all();
  let explored = 0;

  for (const btn of allButtons) {
    if (explored >= MAX_MODAL_EXPLORATIONS) break;
    try {
      const text = (await btn.textContent()).trim().replace(/\s+/g, ' ');
      if (!text || text.length > 100) continue;
      if (DESTRUCTIVE_PATTERNS.test(text)) continue;
      if (!MODAL_TRIGGER_PATTERNS.test(text)) continue;
      const isDisabled = await btn.evaluate(el => el.disabled || el.classList.contains('v-btn--disabled'));
      if (isDisabled) continue;
      if (await btn.evaluate(el => !!el.closest('.v-navigation-drawer'))) continue;

      // Snapshot overlays BEFORE click
      const before = await snapshotOverlays(page);
      const beforeKeys = before.map(o => `${o.tag}|${o.className}|${o.id}`);

      const urlBefore = page.url();
      console.log(`    🔍 Clicking "${text}"...`);
      await btn.click();
      await page.waitForTimeout(2000);

      if (page.url() !== urlBefore) {
        console.log(`    ↩ Navigation detected — going back`);
        await page.goto(urlBefore, { waitUntil: 'networkidle', timeout: 15000 }).catch(() => {});
        await page.waitForTimeout(1500);
        continue;
      }

      // Identify new overlay by diffing
      const newOverlay = await identifyNewOverlay(page, beforeKeys);

      if (newOverlay) {
        console.log(`    ✓ Overlay discovered: ${newOverlay.selector} (classes: ${newOverlay.allClasses})`);
        const components = await extractAllComponents(page);

        console.log(`    📋 Exploring dropdowns inside overlay...`);
        const overlayDropdowns = await exploreDropdowns(page);

        results.push({
          trigger: text,
          overlaySelector: newOverlay.selector,
          overlayClassName: newOverlay.allClasses,
          overlayId: newOverlay.id,
          components,
          dropdowns: overlayDropdowns,
        });
        explored++;

        // Close the overlay — try multiple strategies
        const closeSelectors = [
          `${newOverlay.selector} button:has-text("Cancel")`,
          `${newOverlay.selector} button:has-text("Close")`,
          '.v-overlay--active button:has-text("Cancel")',
          '.v-overlay--active button:has-text("Close")',
          '.v-overlay--active button.v-btn--icon:has(.mdi-close)',
        ];
        let closed = false;
        for (const sel of closeSelectors) {
          try {
            const closeBtn = page.locator(sel).first();
            if (await closeBtn.isVisible({ timeout: 300 }).catch(() => false)) {
              await closeBtn.click();
              closed = true;
              break;
            }
          } catch (e) { /* try next */ }
        }
        if (!closed) await page.keyboard.press('Escape');
        await page.waitForTimeout(1000);
      }
    } catch (e) {
      await page.keyboard.press('Escape').catch(() => {});
      await page.waitForTimeout(500);
    }
  }
  return results;
}

async function exploreExpansionPanels(page) {
  const results = [];
  const panelCount = await page.locator('.v-expansion-panel:visible').count();
  for (let i = 0; i < panelCount; i++) {
    try {
      const panel = page.locator('.v-expansion-panel:visible').nth(i);
      const title = (await panel.locator('.v-expansion-panel-title').textContent().catch(() => '')).trim();
      const isOpen = await panel.evaluate(el => el.classList.contains('v-expansion-panel--active'));
      if (!isOpen) {
        await panel.locator('.v-expansion-panel-title').click();
        await page.waitForTimeout(1000);
      }
      const components = await extractAllComponents(page);
      results.push({ panelTitle: title, panelIndex: i, components });
    } catch (e) { /* panel click failed */ }
  }
  return results;
}

async function exploreDropdowns(page) {
  const results = [];
  const selectSelector = '.v-select:visible, .v-autocomplete:visible, .v-combobox:visible';
  const selectCount = await page.locator(selectSelector).count();

  for (let i = 0; i < selectCount; i++) {
    try {
      const select = page.locator(selectSelector).nth(i);
      if (!(await select.isVisible().catch(() => false))) continue;
      const isDisabled = await select.evaluate(el => el.classList.contains('v-input--disabled'));
      if (isDisabled) continue;

      const label = await select.evaluate(el => {
        const vInput = el.closest('.v-input');
        return vInput?.querySelector('.v-label, .v-field__label')?.textContent?.trim() || null;
      });

      const selectClasses = await select.evaluate(el => el.className?.toString?.() || '');

      await select.click();
      await page.waitForTimeout(1000);

      const optionsInfo = await page.evaluate(() => {
        const overlays = Array.from(document.querySelectorAll('.v-overlay--active'));
        for (let idx = overlays.length - 1; idx >= 0; idx--) {
          const overlay = overlays[idx];
          const list = overlay.querySelector('.v-list');
          if (!list) continue;
          const rect = list.getBoundingClientRect();
          if (rect.width < 20 || rect.height < 20) continue;

          const items = Array.from(list.querySelectorAll('.v-list-item')).map(item => {
            const title = item.querySelector('.v-list-item-title')?.textContent?.trim()
                          || item.textContent?.trim() || '';
            const cls = item.className?.toString?.() || '';
            const customClasses = cls.split(/\s+/).filter(c =>
              c.length > 2 && !c.match(/^(v-|mdi-|d-|ma-|pa-|mt-|mb-|ml-|mr-|mx-|my-|px-|py-|text-|font-|bg-|rounded|elevation|--v-)/)
            );
            return { text: title, className: cls, customClasses };
          }).filter(item => item.text);

          const listCls = list.className?.toString?.() || '';
          return {
            items,
            listClassName: listCls,
            itemClassName: items.length ? items[0].className : '',
          };
        }
        return null;
      });

      if (optionsInfo && optionsInfo.items.length > 0) {
        results.push({
          index: i,
          label: label || `Dropdown ${i + 1}`,
          selectClassName: selectClasses,
          options: optionsInfo.items.map(item => item.text),
          optionDetails: optionsInfo.items,
          listClassName: optionsInfo.listClassName,
          itemClassName: optionsInfo.itemClassName,
        });
        console.log(`    📋 Dropdown "${label || i}": ${optionsInfo.items.length} option(s) — ${optionsInfo.items.map(o => o.text).join(', ')}`);
      }

      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);
    } catch (e) {
      await page.keyboard.press('Escape').catch(() => {});
      await page.waitForTimeout(300);
    }
  }
  return results;
}

// ---------------------------------------------------------------------------
// Login flow
// ---------------------------------------------------------------------------

async function isOnLoginPage(page) {
  const url = page.url();
  return url.includes('/login') || url.includes('/signup');
}

async function doLogin(page) {
  console.log(`  Logging in as ${LOGIN_EMAIL}...`);
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });

  const emailInput = page.locator('.v-text-field').nth(0).locator('input');
  const passwordInput = page.locator('.v-text-field').nth(1).locator('input');
  const loginButton = page.getByRole('button', { name: /login/i });

  await emailInput.fill(LOGIN_EMAIL);
  await passwordInput.fill(LOGIN_PASSWORD);
  await loginButton.click();

  try {
    await page.waitForURL('**/dashboard.int3nt.info/?select_org', { timeout: 15000 });
    console.log('  ✓ Login successful, redirecting to organization selection...');
  } catch (e) {
    console.log('  ⚠️  Login failed or timed out — check credentials or UI changes');
    return false;
  }

  await page.waitForTimeout(2000);

  if (page.url().includes('select_org')) {
    console.log(`  Selecting org "${ORG_NAME}"...`);
    try {
      const nameRegex = new RegExp(`^\\s*${ORG_NAME.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*$`, 'i');
      const orgCard = page.locator('.organization-card').filter({
          has: page.locator('.organization-name', { hasText: nameRegex })
      }).first();

      if (await orgCard.isVisible({ timeout: 5000 })) {
        await orgCard.click();
        await page.waitForTimeout(2000);
        await page.waitForURL(url => !url.includes('select_org'), { timeout: 10000 });
        console.log('  ✓ Org selected, URL:', page.url());
      } else {
        throw new Error('Specific org card not visible');
      }
    } catch (e) {
      console.log(`  ⚠️  Could not find org "${ORG_NAME}", clicking first available card...`);
      try {
        await page.locator('.organization-card').first().click({ timeout: 5000 });
        await page.waitForTimeout(2000);
        await page.waitForURL(url => !url.includes('select_org'), { timeout: 10000 });
        console.log('  ✓ First org card clicked, URL:', page.url());
      } catch (e2) {
        console.log('  ❌ Org selection failed:', e2.message);
        return false;
      }
    }
  }

  return true;
}

// ---------------------------------------------------------------------------
// Per-page inspection — phases: scroll → scan → tabs → modals → panels
// ---------------------------------------------------------------------------

async function inspectPage(page, url, pageName) {
  console.log(`\n${'─'.repeat(60)}`);
  console.log(`📍 ${pageName}  (${url})`);

  try {
    await page.goto(`${BASE_URL}${url}`, { waitUntil: 'networkidle', timeout: 20000 });
    await page.waitForTimeout(2000);
  } catch (e) {
    console.log(`  ⚠️  Navigation timeout — continuing`);
    await page.waitForTimeout(1000);
  }

  const publicPaths = PUBLIC_PAGES.map(p => p.path);
  if (await isOnLoginPage(page) && !publicPaths.includes(url)) {
    console.log(`  ⛔ Redirected to login`);
    return { url, pageName, authFailed: true, components: null, explored: null };
  }

  const isPublic = PUBLIC_PAGES.some(p => p.path === url);

  console.log('  ① Scrolling page...');
  await scrollFullPage(page);

  console.log('  ② Full component scan...');
  const components = await extractAllComponents(page);

  let tabsExplored = [];
  let modalsExplored = [];
  let panelsExplored = [];
  let dropdownsExplored = [];

  if (!isPublic) {
    console.log('  ③ Exploring tabs...');
    tabsExplored = await exploreTabs(page);

    console.log('  ④ Exploring modals...');
    modalsExplored = await exploreModals(page, `${BASE_URL}${url}`);

    console.log('  ⑤ Exploring expansion panels...');
    panelsExplored = await exploreExpansionPanels(page);

    console.log('  ⑥ Exploring dropdown options...');
    dropdownsExplored = await exploreDropdowns(page);
  } else {
    console.log('  ③–⑥ Skipping interactive exploration on public page');
  }

  const explored = { tabs: tabsExplored, modals: modalsExplored, panels: panelsExplored, dropdowns: dropdownsExplored };

  const summary = [];
  for (const [k, v] of Object.entries(components)) {
    const n = Array.isArray(v) ? v.length : (typeof v === 'number' ? v : (v ? 1 : 0));
    if (n > 0) summary.push(`${k}:${n}`);
  }
  console.log(`  → ${summary.join('  ')}`);
  console.log(`  → Explored ${tabsExplored.length} tab(s), ${modalsExplored.length} modal(s), ${panelsExplored.length} panel(s), ${dropdownsExplored.length} dropdown(s)`);

  return { url, pageName, authFailed: false, components, explored };
}

// ---------------------------------------------------------------------------
// SKILL.md generation
// ---------------------------------------------------------------------------

function generatePublicSkillMd(publicData) {
  let md = `---
name: app-selectors-public
description: >
  Comprehensive DOM selectors and component map for public pages on dashboard.int3nt.info.
  Use this skill when writing or fixing Playwright tests. Contains verified
  selectors for every page: headings, text, inputs, buttons, dropdowns,
  checkboxes, switches, tables, cards, chips, tabs, modals, sheets, expansion
  panels, navigation, alerts, and more.
---

# dashboard.int3nt.info — Public Pages Component Map

> Generated: ${new Date().toISOString()}
> Never use auto-generated IDs (\`input-v-N\`) — they change on re-render.
> Never match dynamic-text buttons by exact text — use patterns or position.

## Global Login Flow

\`\`\`
Step 1 — Navigate to login:
  await page.goto('${BASE_URL}/login', { waitUntil: 'networkidle' });

Step 2 — Fill credentials:
  await page.locator('.v-field__input').nth(0).fill('EMAIL');
  await page.locator('.v-field__input').nth(1).fill('PASSWORD');
  await page.locator('button:has-text("Login")').click();

Step 3 — Wait for redirect (REQUIRED):
  await page.waitForURL(/\\?select_org/, { timeout: 15000 });

Step 4 — Select organization:
  await page.locator('.organization-card').first().waitFor({ state: 'visible', timeout: 10000 });
  await page.locator('.organization-card').filter({ hasText: '${ORG_NAME}' }).click();
  await page.waitForURL(/dashboard\\.int3nt\\.info\\/(?!\\?select_org)/, { timeout: 15000 });
\`\`\`

---

## Public Pages

`;

  for (const d of publicData) {
    md += generatePageSection(d);
  }

  return md;
}

function generateProtectedSkillMd(protectedData) {
  let md = `---
name: app-selectors-protected
description: >
  Comprehensive DOM selectors and component map for protected pages on dashboard.int3nt.info.
  Use this skill when writing or fixing Playwright tests. Contains verified
  selectors for every page.
---

# dashboard.int3nt.info — Protected Pages Component Map

> Generated: ${new Date().toISOString()}
> Never use auto-generated IDs (\`input-v-N\`) — they change on re-render.
> Never match dynamic-text buttons by exact text — use patterns or position.

## Protected Pages (Login + Org Required)

`;

  for (const d of protectedData) {
    md += generatePageSection(d);
  }

  return md;
}

function generatePageSection(data) {
  let s = `### ${data.pageName}\n**URL:** \`${data.url}\`\n\n`;

  if (data.authFailed) {
    s += `> ⚠️ Auth expired — re-run script to capture selectors\n\n---\n\n`;
    return s;
  }

  const C = data.components;

  // ── PAGE STRUCTURE ──
  if (C.headings.length) {
    s += `**Headings:**\n`;
    C.headings.forEach(h => { s += `- \`${h.tag}\` — "${h.text}"\n`; });
    s += '\n';
  }

  if (C.breadcrumbs.length) {
    s += `**Breadcrumbs:** ${C.breadcrumbs.join(' › ')}\n\n`;
  }

  if (C.toolbarItems.length) {
    C.toolbarItems.forEach(tb => {
      if (tb.title) s += `**Toolbar Title:** "${tb.title}"\n`;
      if (tb.buttons.length) s += `**Toolbar Buttons:** ${tb.buttons.map(b => `\`${b}\``).join(', ')}\n`;
    });
    s += '\n';
  }

  // ── TEXT CONTENT ──
  if (C.staticTexts.length) {
    s += `**Text Content (${C.staticTexts.length}):**\n`;
    C.staticTexts.forEach(t => { s += `- [${t.type}] "${t.text}"\n`; });
    s += '\n';
  }

  // ── FORM ELEMENTS ──
  const formInputs = C.inputs.filter(inp => !inp.inSelect);
  if (formInputs.length) {
    s += `**Input Fields (${formInputs.length}):**\n\n`;
    s += `| # | Label | Type | Playwright Selector |\n`;
    s += `|---|-------|------|---------------------|\n`;
    formInputs.forEach((inp, i) => {
      const label = inp.label || (inp.type === 'password' ? 'Password' : `Input ${i + 1}`);
      s += `| ${i + 1} | ${label} | \`${inp.type}\` | \`page.locator('.v-field__input').nth(${i})\` |\n`;
    });
    s += '\n';
  }

  if (C.textareas.length) {
    s += `**Textareas (${C.textareas.length}):**\n\n`;
    s += `| # | Label | Playwright Selector |\n`;
    s += `|---|-------|---------------------|\n`;
    C.textareas.forEach((ta, i) => {
      s += `| ${i + 1} | ${ta.label || ta.placeholder || `Textarea ${i + 1}`} | \`page.locator('textarea').nth(${i})\` |\n`;
    });
    s += '\n';
  }

  if (C.selects.length) {
    const exploredDropdowns = data.explored?.dropdowns || [];

    s += `**Dropdowns / Selects (${C.selects.length}):**\n`;
    C.selects.forEach((sel, i) => {
      s += `- **${sel.label || `Dropdown ${i + 1}`}** (${sel.type})`;
      if (sel.currentValue) s += ` — current: "${sel.currentValue}"`;
      if (sel.disabled) s += ` [disabled]`;
      s += `\n`;
      s += `  - Open: \`page.locator('.v-select, .v-autocomplete, .v-combobox').nth(${sel.index}).click()\`\n`;

      const matchedDd = exploredDropdowns.find(dd =>
        dd.label === sel.label || dd.index === sel.index
      );
      if (matchedDd && matchedDd.options.length > 0) {
        s += `  - Options: ${matchedDd.options.map(o => `\`${o}\``).join(', ')}\n`;
        const itemCls = matchedDd.itemClassName || '';
        const customItemClasses = itemCls.split(/\s+/).filter(c =>
          c.length > 2 && !c.match(/^(v-|mdi-|d-|ma-|pa-|mt-|mb-|ml-|mr-|mx-|my-|px-|py-|text-|font-|bg-|rounded|elevation|--v-)/)
        );
        const itemSelector = customItemClasses.length
          ? `.${customItemClasses[0]}:has-text("OPTION")`
          : '.v-list-item:has-text("OPTION")';
        s += `  - Pick option: \`page.locator('${itemSelector}').click()\`\n`;
      } else {
        s += `  - Pick option: \`page.locator('.v-list-item:has-text("OPTION")').click()\`\n`;
      }
    });
    s += '\n';
  }

  if (C.checkboxes.length) {
    s += `**Checkboxes (${C.checkboxes.length}):**\n`;
    C.checkboxes.forEach((cb, i) => {
      s += `- ${cb.label || `Checkbox ${i + 1}`} — ${cb.checked ? '☑ checked' : '☐ unchecked'}`;
      if (cb.disabled) s += ` [disabled]`;
      s += `\n`;
      s += `  \`page.locator('.v-checkbox').nth(${i}).click()\`\n`;
    });
    s += '\n';
  }

  if (C.switches.length) {
    s += `**Switches (${C.switches.length}):**\n`;
    C.switches.forEach((sw, i) => {
      s += `- ${sw.label || `Switch ${i + 1}`} — ${sw.checked ? 'ON' : 'OFF'}`;
      if (sw.disabled) s += ` [disabled]`;
      s += `\n`;
      s += `  \`page.locator('.v-switch').nth(${i}).click()\`\n`;
    });
    s += '\n';
  }

  if (C.radios.length) {
    s += `**Radio Groups (${C.radios.length}):**\n`;
    C.radios.forEach((rg, i) => {
      s += `- **${rg.groupLabel || `Radio Group ${i + 1}`}:** ${rg.options.join(', ')}`;
      if (rg.selected) s += ` — selected: "${rg.selected}"`;
      s += `\n`;
    });
    s += '\n';
  }

  if (C.fileInputs.length) {
    s += `**File Inputs (${C.fileInputs.length}):**\n`;
    C.fileInputs.forEach((fi, i) => {
      s += `- ${fi.label || `File ${i + 1}`}`;
      if (fi.accept) s += ` (accept: ${fi.accept})`;
      s += `\n`;
    });
    s += '\n';
  }

  // ── ACTION ELEMENTS ──
  const stableButtons = [];
  const dynamicButtons = [];
  C.buttons.forEach(btn => {
    const norm = stripDynamic(btn.text);
    if (norm.includes('{DATE}')) dynamicButtons.push({ ...btn, pattern: norm });
    else stableButtons.push(btn);
  });

  if (stableButtons.length) {
    s += `**Buttons (${stableButtons.length}):**\n`;
    const seen = new Set();
    stableButtons.forEach((btn, i) => {
      const dup = seen.has(btn.text);
      seen.add(btn.text);
      const sel = dup
        ? `page.locator('button:has-text("${btn.text}")').nth(N)`
        : `page.locator('button:has-text("${btn.text}")')`;
      s += `- \`${sel}\``;
      if (btn.disabled) s += ` [disabled]`;
      if (btn.duplicate) s += ` *(duplicate text — use .nth())*`;
      s += `\n`;
    });
    s += '\n';
  }

  if (dynamicButtons.length) {
    s += `**⚠️ Dynamic Buttons (DO NOT match by text):**\n`;
    dynamicButtons.forEach(btn => {
      s += `- "${btn.text}" → pattern: \`${btn.pattern}\`\n`;
      s += `  Use position: \`page.locator('button:visible').nth(N)\`\n`;
    });
    s += '\n';
  }

  if (C.iconButtons.length) {
    s += `**Icon Buttons (${C.iconButtons.length}):**\n`;
    C.iconButtons.forEach((ib, i) => {
      const desc = ib.ariaLabel || ib.title || ib.icon || `Icon btn ${i + 1}`;
      s += `- ${desc}`;
      if (ib.icon) s += ` (\`${ib.icon}\`)`;
      s += `\n`;
    });
    s += '\n';
  }

  if (C.links.length) {
    s += `**Links (${C.links.length}):**\n`;
    C.links.forEach(l => { s += `- [${l.text}](${l.href}) — \`page.locator('a:has-text("${l.text}")')\`\n`; });
    s += '\n';
  }

  // ── DATA DISPLAY ──
  if (C.tables.length) {
    C.tables.forEach((tbl, i) => {
      s += `**Table${C.tables.length > 1 ? ` ${i + 1}` : ''}:**\n`;
      if (tbl.headers.length) s += `- Columns: \`${tbl.headers.join('` | `')}\`\n`;
      s += `- Rows: ${tbl.rowCount}`;
      if (tbl.hasPagination) s += ` (paginated)`;
      s += `\n`;
      if (tbl.sampleRow.length) s += `- Sample row: ${tbl.sampleRow.join(' | ')}\n`;
      s += '\n';
    });
  }

  if (C.cards.length) {
    s += `**Cards (${C.cards.length}):**\n`;
    C.cards.forEach((card, i) => {
      s += `- **${card.title || `Card ${i + 1}`}**`;
      if (card.subtitle) s += ` — ${card.subtitle}`;
      s += `\n`;
      if (card.text) s += `  Text: "${card.text}"\n`;
      if (card.actions.length) s += `  Actions: ${card.actions.map(a => `\`${a}\``).join(', ')}\n`;
    });
    s += '\n';
  }

  if (C.chips.length) {
    s += `**Chips (${C.chips.length}):** ${C.chips.map(c => `\`${c.text}\`${c.closable ? ' ✕' : ''}`).join(', ')}\n\n`;
  }

  if (C.lists.length) {
    C.lists.forEach((list, i) => {
      s += `**List${C.lists.length > 1 ? ` ${i + 1}` : ''} (${list.totalItems} items):**\n`;
      list.items.forEach(item => {
        s += `- ${item.title}`;
        if (item.subtitle) s += ` — ${item.subtitle}`;
        s += `\n`;
      });
      s += '\n';
    });
  }

  // ── NAVIGATION ──
  if (C.tabs.length) {
    s += `**Tabs (${C.tabs.length}):** ${C.tabs.map(t => `\`${t.text}\`${t.isActive ? ' ★' : ''}`).join(', ')}\n`;
    s += `- Switch: \`page.locator('.v-tab').nth(INDEX).click()\`\n\n`;
  }

  if (C.navDrawerItems.length) {
    s += `**Sidebar Navigation (${C.navDrawerItems.length}):**\n`;
    C.navDrawerItems.forEach(n => {
      s += `- \`page.locator('a:has-text("${n.text}")')\``;
      if (n.isActive) s += ` ★`;
      s += `\n`;
    });
    s += '\n';
  }

  if (C.pagination.length) {
    s += `**Pagination:** ${C.pagination.map(p => `${p.totalPages} page(s)`).join(', ')}\n\n`;
  }

  // ── FEEDBACK ──
  if (C.alerts.length) {
    s += `**Alerts (${C.alerts.length}):**\n`;
    C.alerts.forEach(a => { s += `- [${a.type}] "${a.text}"\n`; });
    s += '\n';
  }

  if (C.snackbars.length) {
    s += `**Snackbars:** ${C.snackbars.map(sb => `"${sb.text}"`).join(', ')}\n\n`;
  }

  if (C.progressBars.length) {
    s += `**Progress Indicators:** ${C.progressBars.map(p => p.type).join(', ')}\n\n`;
  }

  // ── EXPANSION PANELS ──
  if (C.expansionPanels.length) {
    s += `**Expansion Panels (${C.expansionPanels.length}):**\n`;
    C.expansionPanels.forEach((p, i) => {
      s += `- **${p.title || `Panel ${i + 1}`}** — ${p.isOpen ? 'open' : 'closed'}`;
      if (p.contentPreview) s += `: "${p.contentPreview}"`;
      s += `\n`;
      s += `  \`page.locator('.v-expansion-panel-title').nth(${i}).click()\`\n`;
    });
    s += '\n';
  }

  // ── IMAGES / AVATARS / BADGES ──
  if (C.images.length) {
    s += `**Images:** ${C.images.map(img => `"${img.alt}"`).join(', ')}\n\n`;
  }
  if (C.avatarCount > 0) {
    s += `**Avatars:** ${C.avatarCount} visible\n\n`;
  }
  if (C.badges.length) {
    s += `**Badges:** ${C.badges.map(b => `"${b.content}"`).join(', ')}\n\n`;
  }

  // ── TOOLTIP TRIGGERS ──
  if (C.tooltipTriggers.length) {
    s += `**Tooltips (${C.tooltipTriggers.length}):**\n`;
    C.tooltipTriggers.forEach(t => { s += `- "${t.tooltip}" (on \`${t.tag}\`)\n`; });
    s += '\n';
  }

  // ── ELEMENT REGISTRY ──
  if (C.elementRegistry?.length) {
    const meaningful = C.elementRegistry.filter(e => e.customClasses.length > 0 || e.id || e.testId);
    if (meaningful.length) {
      s += `**Custom Elements & IDs (${meaningful.length}):**\n\n`;
      s += `| Selector | Tag | Classes | Text |\n`;
      s += `|----------|-----|---------|------|\n`;
      meaningful.slice(0, 50).forEach(e => {
        const sel = e.testId ? `[data-testid="${e.testId}"]` : e.selector || '';
        const cls = e.customClasses.join(' ');
        const txt = (e.text || '').replace(/\|/g, '\\|').slice(0, 60);
        s += `| \`${sel}\` | \`${e.tag}\` | \`${cls}\` | ${txt} |\n`;
      });
      s += '\n';
    }
  }

  // ── EXPLORED: TABS ──
  if (data.explored?.tabs?.length) {
    s += `#### Explored Tabs\n\n`;
    for (const tab of data.explored.tabs) {
      s += `**Tab: "${tab.tabName}"** (index ${tab.tabIndex}):\n`;
      s += summarizeExploredComponents(tab.components);
    }
  }

  // ── EXPLORED: MODALS ──
  if (data.explored?.modals?.length) {
    s += `#### Discovered Modals / Dialogs\n\n`;
    for (const modal of data.explored.modals) {
      s += `**Triggered by:** \`page.locator('button:has-text("${modal.trigger}")').click()\`\n`;
      if (modal.overlaySelector) {
        s += `**Overlay selector:** \`page.locator('${modal.overlaySelector}')\`\n`;
        s += `**Wait for overlay:** \`await page.locator('${modal.overlaySelector}').waitFor({ state: 'visible', timeout: 10000 })\`\n`;
      }
      if (modal.overlayClassName) {
        s += `**CSS classes:** \`${modal.overlayClassName}\`\n`;
      }
      s += '\n';
      s += summarizeExploredComponents(modal.components, modal.overlaySelector);

      if (modal.dropdowns?.length) {
        modal.dropdowns.forEach(dd => {
          s += `- **Dropdown: "${dd.label}"** — ${dd.options.length} option(s): ${dd.options.map(o => `\`${o}\``).join(', ')}\n`;
          s += `  - Open: \`page.locator('.v-select, .v-autocomplete, .v-combobox').nth(${dd.index}).click()\`\n`;
          const itemCls = dd.itemClassName || '';
          const customItemCls = itemCls.split(/\s+/).filter(c =>
            c.length > 2 && !c.match(/^(v-|mdi-|d-|ma-|pa-|mt-|mb-|ml-|mr-|mx-|my-|px-|py-|text-|font-|bg-|rounded|elevation|--v-)/)
          );
          const optSel = customItemCls.length ? `.${customItemCls[0]}` : '.v-list-item';
          s += `  - Pick: \`page.locator('${optSel}:has-text("OPTION")').click()\`\n`;
        });
        s += '\n';
      }
    }
  }

  // ── EXPLORED: EXPANSION PANELS ──
  if (data.explored?.panels?.length) {
    s += `#### Explored Expansion Panels\n\n`;
    for (const panel of data.explored.panels) {
      s += `**Panel: "${panel.panelTitle}"** (index ${panel.panelIndex}):\n`;
      s += summarizeExploredComponents(panel.components);
    }
  }

  s += '---\n\n';
  return s;
}

function summarizeExploredComponents(C, overlaySelector) {
  let s = '';
  const scope = overlaySelector || '';

  const dialogsFound = C.dialogs || [];
  if (dialogsFound.length) {
    dialogsFound.forEach(d => {
      if (d.selector) s += `- Container: \`${d.selector}\` (classes: \`${d.className}\`)\n`;
      if (d.title) s += `- Title: "${d.title}"\n`;
      if (d.bodyText) s += `- Text: "${d.bodyText}"\n`;
      if (d.inputs.length) {
        s += `- Inputs:\n`;
        d.inputs.forEach(inp => { s += `  - ${inp.label || 'unlabeled'} (\`${inp.type}\`)\n`; });
      }
      if (d.buttons.length) s += `- Buttons: ${d.buttons.map(b => `\`${b}\``).join(', ')}\n`;
    });
  }

  const extra = [];
  const formInputs = (C.inputs || []).filter(i => !i.inSelect);
  if (formInputs.length) extra.push(`${formInputs.length} input(s): ${formInputs.map(i => i.label || i.placeholder || i.type).join(', ')}`);
  if (C.textareas?.length) extra.push(`${C.textareas.length} textarea(s)`);
  if (C.selects?.length) extra.push(`${C.selects.length} dropdown(s): ${C.selects.map(sel => sel.label || 'unlabeled').join(', ')}`);
  if (C.checkboxes?.length) extra.push(`${C.checkboxes.length} checkbox(es): ${C.checkboxes.map(c => c.label || 'unlabeled').join(', ')}`);
  if (C.switches?.length) extra.push(`${C.switches.length} switch(es): ${C.switches.map(sw => sw.label || 'unlabeled').join(', ')}`);
  if (C.radios?.length) extra.push(`${C.radios.length} radio group(s)`);
  if (C.tables?.length) extra.push(`${C.tables.length} table(s)`);
  if (C.cards?.length) extra.push(`${C.cards.length} card(s): ${C.cards.map(c => c.title || 'untitled').join(', ')}`);
  if (C.chips?.length) extra.push(`chips: ${C.chips.map(c => c.text).join(', ')}`);
  if (C.tabs?.length) extra.push(`tabs: ${C.tabs.map(t => t.text).join(', ')}`);
  if (C.alerts?.length) extra.push(`${C.alerts.length} alert(s)`);
  if (C.headings?.length) extra.push(`headings: ${C.headings.map(h => h.text).join(', ')}`);

  const stableButtons = (C.buttons || []).filter(b => !stripDynamic(b.text).includes('{DATE}'));
  if (stableButtons.length) extra.push(`buttons: ${stableButtons.map(b => b.text).join(', ')}`);

  // Show custom elements found in this overlay
  const customEls = (C.elementRegistry || []).filter(e => e.customClasses.length > 0);
  if (customEls.length) {
    extra.push(`custom elements: ${customEls.slice(0, 15).map(e => e.selector || e.customClasses.join('.')).join(', ')}`);
  }

  if (extra.length) {
    extra.forEach(e => { s += `- ${e}\n`; });
  }

  s += '\n';
  return s;
}

// ---------------------------------------------------------------------------
// Main runner
// ---------------------------------------------------------------------------

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const allData = [];

  console.log('\n' + '='.repeat(70));
  console.log('STEP 1: PUBLIC PAGES');
  console.log('='.repeat(70));
  for (const { path, name } of PUBLIC_PAGES) {
    const data = await inspectPage(page, path, name);
    if (data) allData.push(data);
  }

  console.log('\n' + '='.repeat(70));
  console.log('STEP 2: LOGIN');
  console.log('='.repeat(70));
  const loggedIn = await doLogin(page);
  console.log(loggedIn ? '✓ Login successful' : '❌ Login failed');

  console.log('\n' + '='.repeat(70));
  console.log('STEP 3: PROTECTED PAGES');
  console.log('='.repeat(70));
  for (const { path, name } of PROTECTED_PAGES) {
    const data = await inspectPage(page, path, name);
    if (data) allData.push(data);

    if (data?.authFailed) {
      console.log('  🔄 Re-logging in...');
      await doLogin(page);
    }
  }

  await browser.close();

  const publicPaths = PUBLIC_PAGES.map(p => p.path);
  const publicData = allData.filter(d => publicPaths.includes(d.url));
  const protectedData = allData.filter(d => !publicPaths.includes(d.url));

  const publicSkillDir = __dirname + '/skills/app-selectors-public';
  fs.mkdirSync(publicSkillDir, { recursive: true });
  const publicContent = generatePublicSkillMd(publicData);
  fs.writeFileSync(`${publicSkillDir}/SKILL.md`, publicContent);

  const protectedSkillDir = __dirname + '/skills/app-selectors-protected';
  fs.mkdirSync(protectedSkillDir, { recursive: true });
  const protectedContent = generateProtectedSkillMd(protectedData);
  fs.writeFileSync(`${protectedSkillDir}/SKILL.md`, protectedContent);

  console.log(`\n✅ Public SKILL.md written: ${(publicContent.length / 1024).toFixed(1)} KB`);
  console.log(`✅ Protected SKILL.md written: ${(protectedContent.length / 1024).toFixed(1)} KB`);

  console.log('\n| Page | Hdg | Txt | Inp | Btn | Sel | Chk | Sw | Tab | Tbl | Card | Chip | Modal | Auth |');
  console.log('|------|-----|-----|-----|-----|-----|-----|-----|-----|-----|------|------|-------|------|');
  for (const d of allData) {
    if (d.authFailed) {
      console.log(`| ${d.pageName} | — | — | — | — | — | — | — | — | — | — | — | — | ❌ |`);
      continue;
    }
    const c = d.components;
    const m = d.explored?.modals?.length || 0;
    console.log(
      `| ${d.pageName} | ${c.headings.length} | ${c.staticTexts.length} | ${c.inputs.length} | ${c.buttons.length} ` +
      `| ${c.selects.length} | ${c.checkboxes.length} | ${c.switches.length} | ${c.tabs.length} ` +
      `| ${c.tables.length} | ${c.cards.length} | ${c.chips.length} | ${m} | ✅ |`
    );
  }
  console.log('\n✅ Done!\n');
})();
