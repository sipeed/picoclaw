package dashboard

// dashboardHTML is the embedded frontend served at /dashboard.
// Single-file SPA — no external dependencies except Google Fonts.
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>PicoClaw Gateway Dashboard</title>
<link href="https://fonts.googleapis.com/css2?family=Space+Mono:ital,wght@0,400;0,700;1,400&family=Syne:wght@400;600;700;800&display=swap" rel="stylesheet">
<style>
:root {
  --bg:      #080b0e;
  --bg2:     #0d1117;
  --bg3:     #131920;
  --border:  #1c2333;
  --border2: #243047;
  --accent:  #e8ff47;
  --accent2: #ff6b35;
  --accent3: #3dd9ff;
  --green:   #39ff8a;
  --red:     #ff3d6e;
  --purple:  #b06dff;
  --text:    #dce6f5;
  --muted:   #4a5568;
  --muted2:  #2d3748;
  --mono:    'Space Mono', monospace;
  --sans:    'Syne', sans-serif;
}
*{margin:0;padding:0;box-sizing:border-box}
html,body{height:100%;background:var(--bg);color:var(--text);font-family:var(--sans)}
body::after{content:'';position:fixed;inset:0;background:radial-gradient(ellipse 80% 60% at 50% -10%,rgba(61,217,255,.04),transparent);pointer-events:none;z-index:0}

/* Layout */
.layout{display:grid;grid-template-columns:200px 1fr;min-height:100vh;position:relative;z-index:1}

/* Sidebar */
.sidebar{background:var(--bg2);border-right:1px solid var(--border);display:flex;flex-direction:column}
.brand{padding:20px 16px;border-bottom:1px solid var(--border)}
.brand-name{font-size:15px;font-weight:800;letter-spacing:.1em;color:var(--accent)}
.brand-sub{font-size:9px;font-family:var(--mono);color:var(--muted);letter-spacing:.15em;margin-top:2px}
.health{display:inline-flex;align-items:center;gap:5px;margin-top:10px;padding:3px 8px;background:rgba(57,255,138,.07);border:1px solid rgba(57,255,138,.2);border-radius:20px;font-size:9px;font-family:var(--mono);color:var(--green)}
.hdot{width:5px;height:5px;border-radius:50%;background:var(--green);animation:blink 2s ease infinite}
@keyframes blink{0%,100%{opacity:1}50%{opacity:.2}}
nav{padding:16px 8px;flex:1}
.nav-group{margin-bottom:20px}
.nav-group-label{font-size:8px;letter-spacing:.2em;text-transform:uppercase;color:var(--muted);padding:0 8px;margin-bottom:4px;font-family:var(--mono)}
.nav-item{display:flex;align-items:center;gap:8px;padding:8px 10px;border-radius:6px;cursor:pointer;font-size:12px;font-weight:600;color:var(--muted);transition:all .15s;margin-bottom:1px;user-select:none}
.nav-item:hover{background:var(--bg3);color:var(--text)}
.nav-item.active{background:rgba(232,255,71,.07);color:var(--accent);border-left:2px solid var(--accent);padding-left:8px}
.nav-icon{font-size:13px;width:16px;text-align:center;opacity:.7}
.uptime{padding:12px 16px;border-top:1px solid var(--border);font-size:9px;font-family:var(--mono);color:var(--muted)}

/* Main */
.main{display:flex;flex-direction:column;min-height:100vh}
.topbar{padding:14px 28px;border-bottom:1px solid var(--border);display:flex;align-items:center;justify-content:space-between;background:var(--bg2);position:sticky;top:0;z-index:50}
.topbar-title{font-size:17px;font-weight:800;letter-spacing:-.02em}
.topbar-right{display:flex;align-items:center;gap:12px}
.btn{padding:6px 14px;border-radius:6px;font-size:11px;font-family:var(--mono);font-weight:700;cursor:pointer;transition:all .15s;border:1px solid;letter-spacing:.05em}
.btn-primary{background:rgba(232,255,71,.08);border-color:rgba(232,255,71,.25);color:var(--accent)}
.btn-primary:hover{background:rgba(232,255,71,.15);border-color:rgba(232,255,71,.4)}
.btn-danger{background:rgba(255,61,110,.08);border-color:rgba(255,61,110,.25);color:var(--red)}
.btn-danger:hover{background:rgba(255,61,110,.15)}
.btn-accent{background:rgba(61,217,255,.08);border-color:rgba(61,217,255,.25);color:var(--accent3)}
.btn-accent:hover{background:rgba(61,217,255,.15)}

.content{padding:24px 28px;flex:1}

/* Cards */
.stat-row{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:20px}
.stat{background:var(--bg2);border:1px solid var(--border);border-radius:10px;padding:14px 16px;transition:border-color .2s}
.stat:hover{border-color:var(--border2)}
.stat-lbl{font-size:9px;letter-spacing:.15em;text-transform:uppercase;font-family:var(--mono);color:var(--muted);margin-bottom:6px}
.stat-val{font-size:22px;font-weight:800;letter-spacing:-.03em;line-height:1}
.stat-sub{font-size:10px;font-family:var(--mono);color:var(--muted);margin-top:4px}
.green{color:var(--green)}.red{color:var(--red)}.blue{color:var(--accent3)}.yellow{color:var(--accent)}.purple{color:var(--purple)}

/* Panel */
.panel{background:var(--bg2);border:1px solid var(--border);border-radius:10px;overflow:hidden;margin-bottom:16px}
.panel-head{padding:12px 18px;border-bottom:1px solid var(--border);display:flex;align-items:center;justify-content:space-between;gap:12px}
.panel-title{font-size:11px;font-weight:700;letter-spacing:.1em;text-transform:uppercase}
.panel-body{padding:16px 18px}
.panel-actions{display:flex;gap:8px}

/* Table */
.tbl{width:100%;border-collapse:collapse}
.tbl th{font-size:8px;letter-spacing:.15em;text-transform:uppercase;color:var(--muted);font-family:var(--mono);padding:8px 12px;text-align:left;border-bottom:1px solid var(--border)}
.tbl td{padding:10px 12px;font-size:12px;border-bottom:1px solid rgba(28,35,51,.6);vertical-align:middle}
.tbl tr:last-child td{border-bottom:none}
.tbl tr:hover td{background:rgba(255,255,255,.015)}
.mono{font-family:var(--mono)}

/* Badges */
.badge{display:inline-flex;align-items:center;gap:4px;padding:2px 8px;border-radius:20px;font-size:9px;font-weight:700;font-family:var(--mono)}
.badge-green{background:rgba(57,255,138,.08);border:1px solid rgba(57,255,138,.2);color:var(--green)}
.badge-red{background:rgba(255,61,110,.08);border:1px solid rgba(255,61,110,.2);color:var(--red)}
.badge-blue{background:rgba(61,217,255,.08);border:1px solid rgba(61,217,255,.2);color:var(--accent3)}
.badge-muted{background:rgba(74,85,104,.15);border:1px solid rgba(74,85,104,.25);color:var(--muted)}
.badge-yellow{background:rgba(232,255,71,.08);border:1px solid rgba(232,255,71,.2);color:var(--accent)}
.dot{width:4px;height:4px;border-radius:50%;background:currentColor}

/* Code editor */
.code-editor{width:100%;min-height:320px;background:var(--bg3);border:1px solid var(--border);border-radius:8px;padding:14px;font-family:var(--mono);font-size:12px;color:var(--text);resize:vertical;outline:none;line-height:1.6}
.code-editor:focus{border-color:var(--border2)}

/* Log lines */
.log-wrap{max-height:400px;overflow-y:auto;display:flex;flex-direction:column;gap:2px}
.log-line{font-family:var(--mono);font-size:10px;padding:4px 8px;border-radius:3px;background:var(--bg3);display:flex;gap:10px;align-items:flex-start}
.log-ts{color:var(--muted);white-space:nowrap;min-width:80px}
.log-lvl{font-weight:700;min-width:36px}
.log-src{color:var(--accent3);min-width:80px}
.log-msg{color:var(--text);opacity:.8;flex:1;word-break:break-all}
.lvl-info{color:var(--accent3)}
.lvl-warn{color:var(--accent2)}
.lvl-error{color:var(--red)}
.lvl-debug{color:var(--purple)}

/* Form */
.form-row{display:grid;grid-template-columns:1fr 1fr;gap:12px;margin-bottom:12px}
.form-group{display:flex;flex-direction:column;gap:5px}
.form-group label{font-size:9px;letter-spacing:.12em;text-transform:uppercase;font-family:var(--mono);color:var(--muted)}
.form-group input,.form-group select{background:var(--bg3);border:1px solid var(--border);border-radius:6px;padding:8px 10px;font-family:var(--mono);font-size:12px;color:var(--text);outline:none;transition:border-color .15s}
.form-group input:focus,.form-group select:focus{border-color:var(--border2)}
.form-group select option{background:var(--bg3)}

/* Empty */
.empty{padding:32px;text-align:center;color:var(--muted);font-size:12px;font-family:var(--mono)}

/* Scrollbar */
::-webkit-scrollbar{width:4px;height:4px}
::-webkit-scrollbar-track{background:var(--bg2)}
::-webkit-scrollbar-thumb{background:var(--muted2);border-radius:2px}

/* Spinner */
.spin{display:inline-block;width:14px;height:14px;border:2px solid var(--border);border-top-color:var(--accent);border-radius:50%;animation:spin .7s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
.loading-row{display:flex;align-items:center;justify-content:center;gap:8px;padding:24px;color:var(--muted);font-size:11px;font-family:var(--mono)}

/* Modal */
.modal-backdrop{position:fixed;inset:0;background:rgba(0,0,0,.7);z-index:200;display:flex;align-items:center;justify-content:center}
.modal{background:var(--bg2);border:1px solid var(--border2);border-radius:12px;padding:24px;width:480px;max-width:95vw}
.modal-title{font-size:14px;font-weight:700;margin-bottom:20px}
.modal-actions{display:flex;justify-content:flex-end;gap:8px;margin-top:20px}

/* Toast */
.toast{position:fixed;bottom:24px;right:24px;z-index:300;display:flex;flex-direction:column;gap:8px}
.toast-item{padding:10px 16px;border-radius:8px;font-size:12px;font-family:var(--mono);font-weight:700;animation:slideIn .2s ease}
.toast-ok{background:rgba(57,255,138,.15);border:1px solid rgba(57,255,138,.3);color:var(--green)}
.toast-err{background:rgba(255,61,110,.15);border:1px solid rgba(255,61,110,.3);color:var(--red)}
@keyframes slideIn{from{opacity:0;transform:translateX(20px)}to{opacity:1;transform:translateX(0)}}

/* Fade */
.fade{animation:fade .3s ease}
@keyframes fade{from{opacity:0;transform:translateY(6px)}to{opacity:1;transform:translateY(0)}}
</style>
</head>
<body>
<div class="layout">

<!-- Sidebar -->
<aside class="sidebar">
  <div class="brand">
    <div class="brand-name">🦀 PICOCLAW</div>
    <div class="brand-sub">GATEWAY DASHBOARD</div>
    <div class="health"><span class="hdot"></span>HEALTH OK</div>
  </div>
  <nav>
    <div class="nav-group">
      <div class="nav-group-label">Control</div>
      <div class="nav-item active" data-page="overview" onclick="nav(this,'overview')"><span class="nav-icon">◈</span>Overview</div>
      <div class="nav-item" data-page="config" onclick="nav(this,'config')"><span class="nav-icon">⚙</span>Config</div>
    </div>
    <div class="nav-group">
      <div class="nav-group-label">Agent</div>
      <div class="nav-item" data-page="agents" onclick="nav(this,'agents')"><span class="nav-icon">⬡</span>Agents</div>
      <div class="nav-item" data-page="skills" onclick="nav(this,'skills')"><span class="nav-icon">✦</span>Skills</div>
      <div class="nav-item" data-page="cron" onclick="nav(this,'cron')"><span class="nav-icon">⏱</span>Cron Jobs</div>
    </div>
    <div class="nav-group">
      <div class="nav-group-label">Observe</div>
      <div class="nav-item" data-page="sessions" onclick="nav(this,'sessions')"><span class="nav-icon">⟳</span>Sessions</div>
      <div class="nav-item" data-page="logs" onclick="nav(this,'logs')"><span class="nav-icon">≡</span>Logs</div>
    </div>
  </nav>
  <div class="uptime" id="uptime-display">uptime: —</div>
</aside>

<!-- Main -->
<div class="main">
  <div class="topbar">
    <div class="topbar-title" id="page-title">Overview</div>
    <div class="topbar-right">
      <button class="btn btn-primary" onclick="refresh()">↻ Refresh</button>
    </div>
  </div>

  <div class="content" id="content">
    <div class="loading-row"><div class="spin"></div>loading...</div>
  </div>
</div>

</div>

<!-- Toast container -->
<div class="toast" id="toast"></div>

<script>
// ── State ──────────────────────────────────────────────────────────────────
let currentPage = 'overview';
let DATA = { status:{}, agents:[], skills:[], cron:[], sessions:[], logs:[] };

// ── Navigation ─────────────────────────────────────────────────────────────
function nav(el, page) {
  document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
  el.classList.add('active');
  currentPage = page;
  const titles = {overview:'Overview',config:'Config Editor',agents:'Agents',skills:'Skills',cron:'Cron Jobs',sessions:'Sessions',logs:'Logs'};
  document.getElementById('page-title').textContent = titles[page] || page;
  refresh();
}

// ── API ────────────────────────────────────────────────────────────────────
async function api(path, opts={}) {
  const r = await fetch('/api/' + path, opts);
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

// ── Refresh ────────────────────────────────────────────────────────────────
async function refresh() {
  try {
    switch(currentPage) {
      case 'overview':   await loadOverview(); break;
      case 'config':     await loadConfig(); break;
      case 'agents':     await loadAgents(); break;
      case 'skills':     await loadSkills(); break;
      case 'cron':       await loadCron(); break;
      case 'sessions':   await loadSessions(); break;
      case 'logs':       await loadLogs(); break;
    }
  } catch(e) { toast('Error: ' + e.message, true); }
}

// ── Overview ───────────────────────────────────────────────────────────────
async function loadOverview() {
  const [status, agents, skills, cron, sessions] = await Promise.all([
    api('status'), api('agents'), api('skills'), api('cron'), api('sessions')
  ]);
  DATA = { ...DATA, status, agents, skills, cron, sessions };
  document.getElementById('uptime-display').textContent = 'uptime: ' + (status.uptime || '—');



  const enabledCron = cron.filter(j => j.enabled).length;

  render(` + "`" + `
  <div class="stat-row fade">
    <div class="stat"><div class="stat-lbl">Status</div><div class="stat-val green">ONLINE</div><div class="stat-sub">${status.uptime || '—'}</div></div>
    <div class="stat"><div class="stat-lbl">Agents</div><div class="stat-val blue">${agents.length}</div><div class="stat-sub">configured</div></div>
    <div class="stat"><div class="stat-lbl">Skills</div><div class="stat-val yellow">${skills.length}</div><div class="stat-sub">installed</div></div>
    <div class="stat"><div class="stat-lbl">Cron Jobs</div><div class="stat-val purple">${enabledCron} / ${cron.length}</div><div class="stat-sub">enabled / total</div></div>
  </div>

  <div class="panel fade">
    <div class="panel-head"><div class="panel-title">Agents</div></div>
    <table class="tbl">
      <thead><tr><th>ID</th><th>MODEL</th><th>SKILLS</th><th>WORKSPACE</th><th>SOUL</th></tr></thead>
      <tbody>${agents.map(a => ` + "`" + `
        <tr>
          <td class="mono" style="color:var(--accent3)">${a.id}${a.default ? ' <span class="badge badge-blue">default</span>' : ''}</td>
          <td class="mono" style="font-size:11px">${a.model_name || '—'}</td>
          <td>${(a.skills||[]).slice(0,3).map(s => ` + "`" + `<span class="badge badge-yellow" style="margin-right:2px">${s}</span>` + "`" + `).join('')}${(a.skills||[]).length > 3 ? ` + "`" + `<span class="badge badge-muted">+${a.skills.length-3}</span>` + "`" + ` : ''}</td>
          <td class="mono" style="font-size:10px;color:var(--muted)">${a.workspace || '—'}</td>
          <td>${a.has_soul ? '<span class="badge badge-green">✓ SOUL.md</span>' : '<span class="badge badge-muted">none</span>'}</td>
        </tr>` + "`" + `).join('') || '<tr><td colspan="5" class="empty">No agents configured</td></tr>'}</tbody>
    </table>
  </div>

  <div class="panel fade">
    <div class="panel-head"><div class="panel-title">Active Cron Jobs</div></div>
    <table class="tbl">
      <thead><tr><th>NAME</th><th>SCHEDULE</th><th>CHANNEL</th><th>STATUS</th><th>LAST RUN</th></tr></thead>
      <tbody>${cron.map(j => ` + "`" + `
        <tr>
          <td class="mono">${j.name}</td>
          <td class="mono" style="font-size:10px;color:var(--accent)">${j.schedule?.kind === 'every' ? 'every '+(j.schedule.everyMs/60000)+'m' : j.schedule?.expr || j.schedule?.kind}</td>
          <td class="mono" style="font-size:11px">${j.payload?.channel || '—'}</td>
          <td>${j.enabled ? '<span class="badge badge-green"><span class="dot"></span>ON</span>' : '<span class="badge badge-muted">OFF</span>'}</td>
          <td class="mono" style="font-size:10px;color:var(--muted)">${j.state?.lastRunAtMs ? new Date(j.state.lastRunAtMs).toLocaleTimeString() : '—'}</td>
        </tr>` + "`" + `).join('') || '<tr><td colspan="5" class="empty">No cron jobs</td></tr>'}</tbody>
    </table>
  </div>

  <div class="panel fade">
    <div class="panel-head"><div class="panel-title">Sessions</div></div>
    <table class="tbl">
      <thead><tr><th>KEY</th><th>MODIFIED</th><th>SIZE</th></tr></thead>
      <tbody>${sessions.slice(0,8).map(s => ` + "`" + `
        <tr>
          <td class="mono" style="font-size:11px;color:var(--accent3)">${s.key}</td>
          <td class="mono" style="font-size:10px;color:var(--muted)">${new Date(s.modified).toLocaleString()}</td>
          <td class="mono" style="font-size:10px;color:var(--muted)">${(s.size_bytes/1024).toFixed(1)} KB</td>
        </tr>` + "`" + `).join('') || '<tr><td colspan="3" class="empty">No sessions</td></tr>'}</tbody>
    </table>
  </div>
  ` + "`" + `);
}

// ── Config ─────────────────────────────────────────────────────────────────
async function loadConfig() {
  const cfg = await api('config');
  render(` + "`" + `
  <div class="panel fade">
    <div class="panel-head">
      <div class="panel-title">config.json</div>
      <div class="panel-actions">
        <button class="btn btn-primary" onclick="saveConfig()">💾 Save</button>
      </div>
    </div>
    <div class="panel-body">
      <textarea class="code-editor" id="config-editor">${JSON.stringify(cfg, null, 2)}</textarea>
    </div>
  </div>
  ` + "`" + `);
}

async function saveConfig() {
  try {
    const raw = document.getElementById('config-editor').value;
    const parsed = JSON.parse(raw);
    await api('config', { method:'PUT', headers:{'Content-Type':'application/json'}, body: JSON.stringify(parsed) });
    toast('Config saved ✓');
  } catch(e) { toast('Error: ' + e.message, true); }
}

// ── Agents ─────────────────────────────────────────────────────────────────
async function loadAgents() {
  const [agents, soul] = await Promise.all([api('agents'), api('soul')]);
  DATA.agents = agents;
  render(` + "`" + `
  <div class="panel fade">
    <div class="panel-head">
      <div class="panel-title">Agents (${agents.length})</div>
      <div class="panel-actions">
        <span style="font-size:10px;font-family:var(--mono);color:var(--muted)">configure in config.json</span>
      </div>
    </div>
    <table class="tbl">
      <thead><tr><th>ID</th><th>MODEL</th><th>WORKSPACE</th><th>SKILLS</th><th>SOUL</th></tr></thead>
      <tbody>${agents.map(a => ` + "`" + `
        <tr>
          <td><span class="mono" style="color:var(--accent3);font-weight:700">${a.id}</span>${a.default ? ' <span class="badge badge-blue">default</span>' : ''}</td>
          <td class="mono" style="font-size:11px">${a.model_name || '—'}</td>
          <td class="mono" style="font-size:10px;color:var(--muted)">${a.workspace || '—'}</td>
          <td>
            ${(a.skills||[]).slice(0,4).map(s => ` + "`" + `<span class="badge badge-yellow" style="margin-right:2px">${s}</span>` + "`" + `).join('')}
            ${(a.skills||[]).length > 4 ? ` + "`" + `<span class="badge badge-muted">+${a.skills.length-4}</span>` + "`" + ` : ''}
            ${(a.skills||[]).length === 0 ? '<span class="badge badge-muted">all skills</span>' : ''}
          </td>
          <td>${a.has_soul ? '<span class="badge badge-green">✓ SOUL.md</span>' : '<span class="badge badge-muted">none</span>'}</td>
        </tr>` + "`" + `).join('') || '<tr><td colspan="5" class="empty">No agents found</td></tr>'}</tbody>
    </table>
  </div>

  <div class="panel fade">
    <div class="panel-head">
      <div class="panel-title">✦ SOUL.md — Agent Personality</div>
      <div class="panel-actions">
        <button class="btn btn-primary" onclick="saveSoul()">💾 Save SOUL.md</button>
      </div>
    </div>
    <div class="panel-body">
      <div style="font-size:10px;font-family:var(--mono);color:var(--muted);margin-bottom:10px">
        SOUL.md defines the agent's personality, tone, and behavior. Changes take effect on next conversation.
      </div>
      <textarea class="code-editor" id="soul-editor" style="min-height:400px">${escHtml(soul.content || '')}</textarea>
    </div>
  </div>
  ` + "`" + `);
}

async function saveSoul() {
  try {
    const content = document.getElementById('soul-editor').value;
    await api('soul', { method:'PUT', headers:{'Content-Type':'application/json'}, body: JSON.stringify({content}) });
    toast('SOUL.md saved ✓');
  } catch(e) { toast('Error: ' + e.message, true); }
}

// ── Skills ─────────────────────────────────────────────────────────────────
async function loadSkills() {
  const [skills, agents] = await Promise.all([api('skills'), api('agents')]);
  DATA.skills = skills;
  DATA.agents = agents;
  const agentSkills = (agents[0]?.skills || []);
  render(` + "`" + `
  <div class="panel fade">
    <div class="panel-head">
      <div class="panel-title">Skills (${skills.length})</div>
      <div class="panel-actions">
        <button class="btn btn-accent" onclick="showNewSkill()">+ New Skill</button>
      </div>
    </div>
    <table class="tbl">
      <thead><tr><th>NAME</th><th>DESCRIPTION</th><th>LINKED TO AGENT</th><th>ACTIONS</th></tr></thead>
      <tbody>${skills.map(s => ` + "`" + `
        <tr>
          <td><span class="mono" style="color:var(--accent);font-weight:700">${s.name}</span></td>
          <td style="font-size:11px;color:var(--muted);max-width:260px">${s.description || '<span style="opacity:.4">—</span>'}</td>
          <td>${agentSkills.includes(s.name) ? '<span class="badge badge-green">✓ main agent</span>' : '<span class="badge badge-muted">not linked</span>'}</td>
          <td style="display:flex;gap:6px">
            <button class="btn btn-primary" style="padding:3px 10px;font-size:9px" onclick="editSkill('${s.name}')">✎ Edit</button>
            <button class="btn btn-danger" style="padding:3px 10px;font-size:9px" onclick="deleteSkill('${s.name}')">✕</button>
          </td>
        </tr>` + "`" + `).join('') || '<tr><td colspan="4" class="empty">No skills installed</td></tr>'}</tbody>
    </table>
    <div style="padding:12px 18px;border-top:1px solid var(--border);font-size:10px;font-family:var(--mono);color:var(--muted)">
      Skills are auto-loaded from <span style="color:var(--accent3)">/workspace/skills/</span>
    </div>
  </div>
  <div id="skill-editor-panel"></div>
  ` + "`" + `);
}

async function editSkill(name) {
  const result = await api('skill?name=' + encodeURIComponent(name));
  document.getElementById('skill-editor-panel').innerHTML = ` + "`" + `
  <div class="panel fade">
    <div class="panel-head">
      <div class="panel-title">✎ Editing: ${name}/SKILL.md</div>
      <div class="panel-actions">
        <button class="btn btn-primary" onclick="saveSkill('${name}')">💾 Save</button>
        <button class="btn btn-danger" onclick="document.getElementById('skill-editor-panel').innerHTML=''">✕ Close</button>
      </div>
    </div>
    <div class="panel-body">
      <textarea class="code-editor" id="skill-editor-content" style="min-height:380px">${escHtml(result.content || '')}</textarea>
    </div>
  </div>` + "`" + `;
  document.getElementById('skill-editor-panel').scrollIntoView({behavior:'smooth'});
}

async function saveSkill(name) {
  try {
    const content = document.getElementById('skill-editor-content').value;
    await api('skill?name=' + encodeURIComponent(name), {
      method:'PUT', headers:{'Content-Type':'application/json'}, body: JSON.stringify({content})
    });
    toast('SKILL.md saved ✓');
  } catch(e) { toast('Error: ' + e.message, true); }
}

async function deleteSkill(name) {
  if (!confirm('Delete skill "' + name + '"? This cannot be undone.')) return;
  try {
    await api('skill?name=' + encodeURIComponent(name), { method:'DELETE' });
    toast('Skill deleted ✓');
    await loadSkills();
  } catch(e) { toast('Error: ' + e.message, true); }
}

function showNewSkill() {
  const backdrop = document.createElement('div');
  backdrop.className = 'modal-backdrop';
  backdrop.id = 'skill-modal';
  backdrop.innerHTML = ` + "`" + `
  <div class="modal">
    <div class="modal-title">New Skill</div>
    <div class="form-group" style="margin-bottom:12px">
      <label>Skill Name (folder name)</label>
      <input id="sk-name" placeholder="my-skill">
    </div>
    <div class="form-group">
      <label>Initial SKILL.md content (optional)</label>
      <textarea class="code-editor" id="sk-content" style="min-height:160px"># My Skill

description: "Describe what this skill does"

## Instructions

Write your skill instructions here.
</textarea>
    </div>
    <div class="modal-actions">
      <button class="btn btn-danger" onclick="document.getElementById('skill-modal').remove()">Cancel</button>
      <button class="btn btn-primary" onclick="createSkill()">Create Skill</button>
    </div>
  </div>` + "`" + `;
  document.body.appendChild(backdrop);
}

async function createSkill() {
  const name = document.getElementById('sk-name').value.trim();
  const content = document.getElementById('sk-content').value;
  if (!name) { toast('Name is required', true); return; }
  try {
    await api('skill?name=' + encodeURIComponent(name), {
      method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({content})
    });
    document.getElementById('skill-modal').remove();
    toast('Skill created ✓');
    await loadSkills();
  } catch(e) { toast('Error: ' + e.message, true); }
}

// ── Cron ───────────────────────────────────────────────────────────────────
async function loadCron() {
  const jobs = await api('cron');
  DATA.cron = jobs;
  render(` + "`" + `
  <div class="panel fade">
    <div class="panel-head">
      <div class="panel-title">Cron Jobs (${jobs.length})</div>
      <div class="panel-actions">
        <button class="btn btn-accent" onclick="showAddCron()">+ New Job</button>
      </div>
    </div>
    <table class="tbl">
      <thead><tr><th>NAME</th><th>SCHEDULE</th><th>COMMAND</th><th>CHANNEL → TO</th><th>STATUS</th><th>LAST</th><th></th></tr></thead>
      <tbody>${jobs.map(j => ` + "`" + `
        <tr>
          <td class="mono" style="font-weight:700">${j.name}</td>
          <td><span class="badge badge-yellow">${j.schedule?.kind === 'every' ? 'every '+(j.schedule.everyMs/60000)+'m' : j.schedule?.expr || j.schedule?.kind}</span></td>
          <td class="mono" style="font-size:10px;max-width:180px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${j.payload?.command || j.payload?.message || '—'}</td>
          <td class="mono" style="font-size:10px">${j.payload?.channel || '—'} → ${j.payload?.to || '—'}</td>
          <td>
            <button class="btn ${j.enabled ? 'btn-danger' : 'btn-primary'}" style="padding:3px 10px;font-size:9px"
              onclick="toggleCron('${j.id}',${!j.enabled})">${j.enabled ? 'DISABLE' : 'ENABLE'}</button>
          </td>
          <td class="mono" style="font-size:9px;color:var(--muted)">${j.state?.lastStatus || '—'}</td>
          <td><button class="btn btn-danger" style="padding:3px 10px;font-size:9px" onclick="deleteCron('${j.id}')">✕</button></td>
        </tr>` + "`" + `).join('') || '<tr><td colspan="7" class="empty">No cron jobs configured</td></tr>'}</tbody>
    </table>
  </div>
  ` + "`" + `);
}

async function toggleCron(id, enabled) {
  try {
    await api(` + "`" + `cron?id=${id}&enabled=${enabled}` + "`" + `, { method:'PUT' });
    toast(enabled ? 'Job enabled ✓' : 'Job disabled ✓');
    await loadCron();
  } catch(e) { toast('Error: ' + e.message, true); }
}

async function deleteCron(id) {
  if (!confirm('Delete this cron job?')) return;
  try {
    await api('cron?id=' + id, { method:'DELETE' });
    toast('Job deleted ✓');
    await loadCron();
  } catch(e) { toast('Error: ' + e.message, true); }
}

function showAddCron() {
  const backdrop = document.createElement('div');
  backdrop.className = 'modal-backdrop';
  backdrop.id = 'cron-modal';
  backdrop.innerHTML = ` + "`" + `
  <div class="modal">
    <div class="modal-title">New Cron Job</div>
    <div class="form-row">
      <div class="form-group"><label>Name</label><input id="c-name" placeholder="My Alert"></div>
      <div class="form-group"><label>Schedule Type</label>
        <select id="c-kind" onchange="updateCronForm()">
          <option value="every">Every N minutes</option>
          <option value="cron">Cron expression</option>
        </select>
      </div>
    </div>
    <div class="form-row" id="cron-schedule-row">
      <div class="form-group"><label>Every (minutes)</label><input id="c-every" type="number" value="30" min="1"></div>
      <div class="form-group"><label>Cron expression</label><input id="c-expr" placeholder="0 * * * *" disabled style="opacity:.4"></div>
    </div>
    <div class="form-row">
      <div class="form-group"><label>Command (exec)</label><input id="c-cmd" placeholder="/usr/bin/python3 /workspace/scripts/alert.py"></div>
      <div class="form-group"><label>Message (agent)</label><input id="c-msg" placeholder="Generate daily report"></div>
    </div>
    <div class="form-row">
      <div class="form-group"><label>Channel</label><input id="c-channel" value="telegram"></div>
      <div class="form-group"><label>To (chat_id)</label><input id="c-to" placeholder="-1003766441499|5"></div>
    </div>
    <div class="form-row">
      <div class="form-group"><label>Deliver result</label>
        <select id="c-deliver"><option value="false">false (silent)</option><option value="true">true (send to chat)</option></select>
      </div>
    </div>
    <div class="modal-actions">
      <button class="btn btn-danger" onclick="document.getElementById('cron-modal').remove()">Cancel</button>
      <button class="btn btn-primary" onclick="addCron()">Create Job</button>
    </div>
  </div>` + "`" + `;
  document.body.appendChild(backdrop);
}

function updateCronForm() {
  const kind = document.getElementById('c-kind').value;
  document.getElementById('c-every').disabled = kind !== 'every';
  document.getElementById('c-every').style.opacity = kind === 'every' ? '1' : '.4';
  document.getElementById('c-expr').disabled = kind !== 'cron';
  document.getElementById('c-expr').style.opacity = kind === 'cron' ? '1' : '.4';
}

async function addCron() {
  const name = document.getElementById('c-name').value.trim();
  const kind = document.getElementById('c-kind').value;
  const every = parseInt(document.getElementById('c-every').value) * 60000;
  const expr = document.getElementById('c-expr').value.trim();
  const cmd = document.getElementById('c-cmd').value.trim();
  const msg = document.getElementById('c-msg').value.trim();
  const channel = document.getElementById('c-channel').value.trim();
  const to = document.getElementById('c-to').value.trim();
  const deliver = document.getElementById('c-deliver').value === 'true';

  if (!name) { toast('Name is required', true); return; }

  const schedule = kind === 'every'
    ? { kind: 'every', everyMs: every }
    : { kind: 'cron', expr };

  try {
    await api('cron', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, schedule, command: cmd, message: msg, channel, to, deliver })
    });
    document.getElementById('cron-modal').remove();
    toast('Job created ✓');
    await loadCron();
  } catch(e) { toast('Error: ' + e.message, true); }
}

// ── Sessions ───────────────────────────────────────────────────────────────
async function loadSessions() {
  const sessions = await api('sessions');
  DATA.sessions = sessions;
  render(` + "`" + `
  <div class="panel fade">
    <div class="panel-head"><div class="panel-title">Sessions (${sessions.length})</div></div>
    <table class="tbl">
      <thead><tr><th>SESSION KEY</th><th>LAST MODIFIED</th><th>SIZE</th></tr></thead>
      <tbody>${sessions.map(s => ` + "`" + `
        <tr>
          <td class="mono" style="font-size:11px;color:var(--accent3)">${s.key}</td>
          <td class="mono" style="font-size:10px;color:var(--muted)">${new Date(s.modified).toLocaleString()}</td>
          <td class="mono" style="font-size:10px;color:var(--muted)">${(s.size_bytes/1024).toFixed(1)} KB</td>
        </tr>` + "`" + `).join('') || '<tr><td colspan="3" class="empty">No sessions found</td></tr>'}</tbody>
    </table>
  </div>
  ` + "`" + `);
}

// ── Logs ───────────────────────────────────────────────────────────────────
async function loadLogs() {
  const lines = await api('logs');
  render(` + "`" + `
  <div class="panel fade">
    <div class="panel-head">
      <div class="panel-title">System Logs</div>
      <div class="panel-actions">
        <button class="btn btn-primary" onclick="loadLogs()">↻ Reload</button>
      </div>
    </div>
    <div class="panel-body">
      <div class="log-wrap" id="log-wrap">
        ${lines.length === 0 ? '<div class="empty">No logs found</div>' :
          lines.map(l => {
            const lvl = (l.match(/\[(\w+)\]/)||['','INFO'])[1].toUpperCase();
            const cls = lvl==='ERROR'?'lvl-error':lvl==='WARN'?'lvl-warn':lvl==='DEBUG'?'lvl-debug':'lvl-info';
            return ` + "`" + `<div class="log-line"><span class="log-msg" style="color:var(--muted)">${escHtml(l)}</span></div>` + "`" + `;
          }).join('')
        }
      </div>
    </div>
  </div>
  ` + "`" + `);
  const w = document.getElementById('log-wrap');
  if (w) w.scrollTop = w.scrollHeight;
}

// ── Helpers ────────────────────────────────────────────────────────────────
function render(html) {
  document.getElementById('content').innerHTML = html;
}

function escHtml(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

function toast(msg, err=false) {
  const t = document.getElementById('toast');
  const el = document.createElement('div');
  el.className = 'toast-item ' + (err ? 'toast-err' : 'toast-ok');
  el.textContent = msg;
  t.appendChild(el);
  setTimeout(() => el.remove(), 3000);
}

// ── Init ───────────────────────────────────────────────────────────────────
refresh();
setInterval(() => {
  api('status').then(s => {
    document.getElementById('uptime-display').textContent = 'uptime: ' + (s.uptime||'—');
  }).catch(()=>{});
}, 10000);
</script>
</body>
</html>`
