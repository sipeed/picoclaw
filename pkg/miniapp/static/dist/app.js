(() => {
  // src/logs_view.js
  var SAFE_LEVELS = new Set(["debug", "info", "warn", "error"]);
  function stringifyFieldValue(value) {
    if (value === null || value === undefined) {
      return "";
    }
    if (typeof value === "string") {
      return value;
    }
    if (typeof value === "number" || typeof value === "boolean") {
      return String(value);
    }
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }
  function escapeHtml(value) {
    return String(value == null ? "" : value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
  }
  function renderFields(fields) {
    if (!fields || typeof fields !== "object" || Array.isArray(fields)) {
      return "";
    }
    const keys = Object.keys(fields);
    if (keys.length === 0) {
      return "";
    }
    const parts = keys.map((key) => `${key}=${stringifyFieldValue(fields[key])}`);
    return ` <span class="log-fields">{${escapeHtml(parts.join(", "))}}</span>`;
  }
  function filterLogs(entries, component = "") {
    if (!Array.isArray(entries) || entries.length === 0) {
      return [];
    }
    if (!component) {
      return entries.slice();
    }
    return entries.filter((entry) => (entry?.component || "") === component);
  }
  function paginateLogs(entries, page = 1, pageSize = 100) {
    const list = Array.isArray(entries) ? entries : [];
    const size = Math.max(1, Number(pageSize) || 100);
    const totalPages = Math.max(1, Math.ceil(list.length / size));
    const currentPage = Math.min(Math.max(1, Number(page) || 1), totalPages);
    const end = list.length - (currentPage - 1) * size;
    const start = Math.max(0, end - size);
    return {
      items: list.slice(start, Math.max(start, end)),
      currentPage,
      totalPages,
      pageSize: size
    };
  }
  function renderLogs(entries, options = {}) {
    const component = options.component || "";
    const filtered = filterLogs(entries, component);
    const paged = paginateLogs(filtered, options.page, options.pageSize);
    let html = "";
    for (const entry of paged.items) {
      const levelRaw = String(entry?.level || "info").toLowerCase();
      const level = SAFE_LEVELS.has(levelRaw) ? levelRaw : "info";
      const ts = entry?.timestamp ? String(entry.timestamp).substring(11, 19) : "";
      const componentHTML = entry?.component ? `<span class="log-comp">${escapeHtml(entry.component)}</span>` : "";
      const fieldsHTML = renderFields(entry?.fields);
      const message = escapeHtml(entry?.message || "");
      html += '<div class="log-entry">' + `<span class="log-ts">${ts}</span>` + `<span class="log-badge ${level}">${level}</span>` + componentHTML + `<span class="log-msg">${message}${fieldsHTML}</span>` + "</div>";
    }
    return {
      html,
      totalItems: filtered.length,
      currentPage: paged.currentPage,
      totalPages: paged.totalPages,
      pageSize: paged.pageSize
    };
  }

  // src/app.js
  var tg = window.Telegram.WebApp;
  tg.ready();
  var API_BASE = location.origin;
  var initData = tg.initData || "";
  var selectedSkill = null;
  var lastSSE = { plan: 0, skills: 0, session: 0, dev: 0 };
  if (!window.ORCH_ENABLED) {
    orchTabBtn = document.querySelector('.tab[data-panel="orch"]');
    orchPanel = document.getElementById("orch");
    if (orchTabBtn)
      orchTabBtn.style.display = "none";
    if (orchPanel)
      orchPanel.style.display = "none";
    document.documentElement.style.setProperty("--tab-count", "6");
  }
  var orchTabBtn;
  var orchPanel;
  var tabs = document.querySelectorAll('.tab:not([style*="display: none"])');
  var tabIndicator = document.querySelector(".tab-indicator");
  function moveIndicator(index) {
    tabIndicator.style.transform = "translateX(" + index * 100 + "%)";
  }
  tabs.forEach((tab, index) => {
    tab.addEventListener("click", () => {
      tabs.forEach((t) => t.classList.remove("active"));
      document.querySelectorAll(".panel").forEach((p2) => p2.classList.remove("active"));
      tab.classList.add("active");
      document.getElementById(tab.dataset.panel).classList.add("active");
      moveIndicator(index);
      document.getElementById("send-bar").classList.toggle("hidden", !(tab.dataset.panel === "skills" && selectedSkill));
      var p = tab.dataset.panel;
      var fresh = lastSSE[p] && Date.now() - lastSSE[p] < 5000;
      if (p === "plan" && !fresh)
        loadPlan();
      if (p === "skills" && !fresh)
        loadSkills();
      if (p === "session" && !fresh)
        loadSession();
      if (p === "git")
        loadGit();
      if (p === "dev" && !fresh)
        loadDev();
      if (p === "config")
        connectLogsWs();
      else
        disconnectLogsWs();
      if (p === "orch")
        connectOrchWs();
      else
        disconnectOrchWs();
    });
  });
  document.querySelectorAll(".cmd-tile").forEach((tile) => {
    tile.addEventListener("click", async () => {
      const ok = await sendCommand(tile.dataset.cmd);
      if (ok)
        flashSent(tile);
    });
  });
  async function sendCommand(cmd) {
    if (!cmd.startsWith("/"))
      return false;
    try {
      const res = await fetch(API_BASE + "/miniapp/api/command?initData=" + encodeURIComponent(initData), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ command: cmd })
      });
      if (!res.ok)
        throw new Error("API error: " + res.status);
      return true;
    } catch (e) {
      return false;
    }
  }
  async function sendCustomCmd() {
    const input = document.getElementById("custom-cmd");
    const btn = input.nextElementSibling;
    const cmd = input.value.trim();
    if (!cmd)
      return;
    if (!cmd.startsWith("/"))
      return;
    const ok = await sendCommand(cmd);
    if (ok) {
      input.value = "";
      flashSent(btn);
    }
  }
  async function sendSkillCommand() {
    if (!selectedSkill)
      return;
    const msg = document.getElementById("skill-msg").value.trim();
    const cmd = msg ? "/skill " + selectedSkill + " " + msg : "/skill " + selectedSkill;
    const btn = document.getElementById("send-skill-btn");
    const ok = await sendCommand(cmd);
    if (ok)
      flashSent(btn);
  }
  async function startPlan() {
    const input = document.getElementById("plan-task");
    const btn = input.nextElementSibling;
    const task = input.value.trim();
    if (!task)
      return;
    const ok = await sendCommand("/plan " + task);
    if (ok) {
      input.value = "";
      flashSent(btn);
    }
  }
  function flashSent(el) {
    el.classList.add("sent");
    setTimeout(() => el.classList.remove("sent"), 600);
  }
  async function apiFetch(path) {
    const sep = path.includes("?") ? "&" : "?";
    const res = await fetch(API_BASE + path + sep + "initData=" + encodeURIComponent(initData));
    if (!res.ok)
      throw new Error("API error: " + res.status);
    return res.json();
  }
  function renderPlanFromData(data) {
    var loading = document.getElementById("plan-loading");
    var el = document.getElementById("plan-content");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    if (!data.has_plan) {
      el.innerHTML = `<div class="empty-state">No active plan.</div>
      <div class="card glass" style="margin-top:16px">
        <div class="card-title">Start a Plan</div>
        <div style="display:flex;gap:8px;margin-top:8px">
          <input id="plan-task" class="send-input glass glass-interactive" placeholder="Describe your task...">
          <button class="send-btn" onclick="startPlan()">Start</button>
        </div>
      </div>`;
      return;
    }
    var html = `<div class="card glass">
    <div class="card-title">Status</div>
    <div class="card-value">${escapeHtml2(data.status)}</div>
    <div style="color:var(--hint);margin-top:4px">Phase ${data.current_phase} / ${data.total_phases}</div>
  </div>`;
    if (data.status === "interviewing" || data.status === "review") {
      if (data.memory) {
        html += `<div class="memory-view glass">${renderSimpleMarkdown(data.memory)}</div>`;
      }
      if (data.status === "review") {
        html += `<div class="slide-approve-wrap">
        <div class="slide-approve-track glass glass-interactive" data-cmd="/plan start">
          <div class="slide-approve-thumb"><svg viewBox="0 0 24 24"><path d="M5 12h14m-6-6 6 6-6 6" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/></svg></div>
          <div class="slide-approve-label">Slide to Approve</div>
        </div>
      </div>
      <div class="slide-approve-wrap">
        <div class="slide-approve-track glass glass-interactive" data-cmd="/plan start clear" style="border-color:var(--warn,#ff9800)">
          <div class="slide-approve-thumb" style="background:var(--warn,#ff9800)"><svg viewBox="0 0 24 24"><path d="M5 12h14m-6-6 6 6-6 6" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/></svg></div>
          <div class="slide-approve-label">Approve &amp; Clear History</div>
        </div>
      </div>`;
      }
    } else {
      if (data.phases && data.phases.length > 0) {
        html += renderPhases(data.phases, data.current_phase);
      }
    }
    el.innerHTML = html;
    if (data.status === "review")
      setupSlideApprove();
  }
  var slideApproveAC = null;
  function setupSlideApprove() {
    if (slideApproveAC)
      slideApproveAC.abort();
    slideApproveAC = new AbortController;
    var signal = slideApproveAC.signal;
    var tracks = document.querySelectorAll(".slide-approve-track");
    if (!tracks.length)
      return;
    tracks.forEach(function(track) {
      var thumb = track.querySelector(".slide-approve-thumb");
      var label = track.querySelector(".slide-approve-label");
      var cmd = track.getAttribute("data-cmd") || "/plan start";
      var dragging = false;
      var startX = 0;
      var thumbStartLeft = 0;
      function getMaxLeft() {
        return track.offsetWidth - thumb.offsetWidth - 6;
      }
      function markAllApproved() {
        tracks.forEach(function(t) {
          t.classList.add("approved");
          t.querySelector(".slide-approve-label").textContent = "Approved!";
          t.querySelector(".slide-approve-thumb").classList.add("hidden");
        });
      }
      function onStart(e) {
        if (track.classList.contains("approved"))
          return;
        dragging = true;
        thumb.classList.add("dragging");
        var clientX = e.touches ? e.touches[0].clientX : e.clientX;
        startX = clientX;
        thumbStartLeft = thumb.offsetLeft - 3;
        e.preventDefault();
      }
      function onMove(e) {
        if (!dragging)
          return;
        var clientX = e.touches ? e.touches[0].clientX : e.clientX;
        var dx = clientX - startX;
        var newLeft = Math.max(0, Math.min(thumbStartLeft + dx, getMaxLeft()));
        thumb.style.left = newLeft + 3 + "px";
        e.preventDefault();
      }
      function onEnd(e) {
        if (!dragging)
          return;
        dragging = false;
        thumb.classList.remove("dragging");
        var currentLeft = thumb.offsetLeft - 3;
        var maxLeft = getMaxLeft();
        if (currentLeft >= maxLeft * 0.8) {
          markAllApproved();
          sendCommand(cmd);
        } else {
          thumb.style.left = "3px";
        }
      }
      thumb.addEventListener("touchstart", onStart, { passive: false, signal });
      thumb.addEventListener("mousedown", onStart, { signal });
      document.addEventListener("touchmove", onMove, { passive: false, signal });
      document.addEventListener("mousemove", onMove, { signal });
      document.addEventListener("touchend", onEnd, { signal });
      document.addEventListener("mouseup", onEnd, { signal });
    });
  }
  function renderSimpleMarkdown(text) {
    var lines = text.split(`
`);
    var out = [];
    for (var i = 0;i < lines.length; i++) {
      var line = lines[i];
      if (/^### /.test(line)) {
        out.push('<div class="md-h3">' + escapeHtml2(line.slice(4)) + "</div>");
      } else if (/^## /.test(line)) {
        out.push('<div class="md-h2">' + escapeHtml2(line.slice(3)) + "</div>");
      } else if (/^# /.test(line)) {
        out.push('<div class="md-h1">' + escapeHtml2(line.slice(2)) + "</div>");
      } else if (/^- \[x\] /.test(line)) {
        out.push('<div class="md-checkbox"><span class="md-checkbox-icon checked"></span>' + escapeHtml2(line.slice(6)) + "</div>");
      } else if (/^- \[ \] /.test(line)) {
        out.push('<div class="md-checkbox"><span class="md-checkbox-icon"></span>' + escapeHtml2(line.slice(6)) + "</div>");
      } else if (/^> /.test(line)) {
        out.push('<div class="md-quote">' + escapeHtml2(line.slice(2)) + "</div>");
      } else if (/^- /.test(line)) {
        out.push('<div class="md-bullet">' + escapeHtml2(line.slice(2)) + "</div>");
      } else if (line.trim() === "") {
        out.push("<br>");
      } else {
        out.push("<div>" + escapeHtml2(line) + "</div>");
      }
    }
    return out.join("");
  }
  async function loadTab(loadingId, contentId, label, fetchFn, renderFn) {
    var loading = document.getElementById(loadingId);
    var el = document.getElementById(contentId);
    loading.classList.remove("hidden");
    loading.textContent = "Loading " + label + "...";
    el.classList.add("hidden");
    try {
      renderFn(await fetchFn());
    } catch (e) {
      loading.textContent = "Failed to load " + label + ".";
    }
  }
  function loadPlan() {
    return loadTab("plan-loading", "plan-content", "plan", function() {
      return apiFetch("/miniapp/api/plan");
    }, renderPlanFromData);
  }
  function renderPhases(phases, currentPhase) {
    return phases.map((phase) => {
      const doneCount = phase.steps.filter((s) => s.done).length;
      const total = phase.steps.length;
      let indicatorClass, indicator;
      if (phase.number < currentPhase || total > 0 && doneCount === total) {
        indicatorClass = "done";
        indicator = "✓";
      } else if (phase.number === currentPhase) {
        indicatorClass = "current";
        indicator = String(phase.number);
      } else {
        indicatorClass = "pending";
        indicator = String(phase.number);
      }
      const progressHtml = total > 0 ? `<span class="phase-progress">${doneCount}/${total}</span>` : "";
      const stepsHtml = phase.steps.map((step) => {
        const doneClass = step.done ? "done" : "";
        const stepClass = step.done ? "step step-done" : "step";
        return `<div class="${stepClass}" data-phase="${phase.number}" data-step="${step.index}" data-done="${step.done}">
        <div class="step-check ${doneClass}"></div>
        <div class="step-text ${doneClass}">${escapeHtml2(step.description)}</div>
      </div>`;
      }).join("");
      return `<div class="phase">
      <div class="phase-header">
        <div class="phase-indicator ${indicatorClass}">${indicator}</div>
        <span class="phase-title">${escapeHtml2(phase.title || "Phase " + phase.number)}</span>
        ${progressHtml}
      </div>
      ${stepsHtml}
    </div>`;
    }).join("");
  }
  document.getElementById("plan-content").addEventListener("click", function(e) {
    const step = e.target.closest(".step");
    if (!step)
      return;
    if (step.dataset.done === "true")
      return;
    const phase = step.dataset.phase;
    const stepIdx = step.dataset.step;
    sendCommand("/plan done " + stepIdx);
  });
  function renderSkillsFromData(data) {
    var loading = document.getElementById("skills-loading");
    var el = document.getElementById("skills-list");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    if (!data || data.length === 0) {
      el.innerHTML = '<div class="empty-state">No skills installed.</div>';
      return;
    }
    el.innerHTML = data.map((s) => `<div class="skill-item glass glass-interactive" data-skill="${escapeAttr(s.name)}">
    <div class="skill-body">
      <div class="skill-name">${escapeHtml2(s.name)}</div>
      <div class="skill-desc">${escapeHtml2(s.description || "No description")}</div>
      <span class="skill-source">${escapeHtml2(s.source)}</span>
    </div>
    <span class="skill-arrow">›</span>
  </div>`).join("");
    if (selectedSkill) {
      var prev = el.querySelector('[data-skill="' + CSS.escape(selectedSkill) + '"]');
      if (prev)
        prev.classList.add("selected");
    }
  }
  document.getElementById("skills-list").addEventListener("click", function(e) {
    var item = e.target.closest(".skill-item");
    if (!item)
      return;
    var el = document.getElementById("skills-list");
    if (selectedSkill === item.dataset.skill) {
      item.classList.remove("selected");
      selectedSkill = null;
      document.getElementById("send-bar").classList.add("hidden");
      return;
    }
    el.querySelectorAll(".skill-item").forEach(function(i) {
      i.classList.remove("selected");
    });
    item.classList.add("selected");
    selectedSkill = item.dataset.skill;
    document.getElementById("send-bar").classList.remove("hidden");
    document.getElementById("skill-msg").placeholder = "Message for /" + selectedSkill + "...";
    document.getElementById("skill-msg").focus();
  });
  function loadSkills() {
    return loadTab("skills-loading", "skills-list", "skills", function() {
      return apiFetch("/miniapp/api/skills");
    }, renderSkillsFromData);
  }
  function formatAge(sec) {
    if (sec < 60)
      return sec + "s ago";
    if (sec < 3600)
      return Math.floor(sec / 60) + "m ago";
    return Math.floor(sec / 3600) + "h ago";
  }
  function shortSessionKey(key) {
    var parts = key.split(":");
    if (parts.length > 2)
      return parts.slice(2).join(":");
    return key;
  }
  function renderActiveSessions(sessions) {
    if (!sessions || sessions.length === 0) {
      return `<div class="card glass">
      <div class="card-title">Active Sessions</div>
      <div style="color:var(--hint);font-size:13px">No active sessions</div>
    </div>`;
    }
    return `<div class="card glass"><div class="card-title">Active Sessions</div>
    ${sessions.map((s) => {
      var touchDir = s.touch_dir || "—";
      return `<div style="padding:6px 0;border-bottom:1px solid var(--secondary-bg)">
        <div style="display:flex;align-items:center;gap:6px">
          <span style="color:var(--done);font-size:10px">●</span>
          <span style="font-weight:600;font-size:13px">${escapeHtml2(shortSessionKey(s.session_key))}</span>
          <span style="margin-left:auto;color:var(--hint);font-size:12px">${formatAge(s.age_sec)}</span>
        </div>
        <div style="color:var(--hint);font-size:12px;padding-left:16px">touch: ${escapeHtml2(touchDir)}</div>
      </div>`;
    }).join("")}
  </div>`;
  }
  function renderSessionFromData(sessions, stats) {
    var loading = document.getElementById("session-loading");
    var el = document.getElementById("session-content");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    var html = renderActiveSessions(sessions);
    if (!stats || stats.status === "stats not enabled") {
      html += '<div class="empty-state">Stats tracking not enabled.<br>Start gateway with --stats flag.</div>';
      el.innerHTML = html;
      return;
    }
    var since = stats.since ? new Date(stats.since).toLocaleDateString() : "N/A";
    var today = stats.today || {};
    html += `<div class="card glass">
    <div class="card-title">Today</div>
    <div class="stat-row"><span class="stat-label">Prompts</span><span class="stat-value">${today.prompts || 0}</span></div>
    <div class="stat-row"><span class="stat-label">Requests</span><span class="stat-value">${today.requests || 0}</span></div>
    <div class="stat-row"><span class="stat-label">Tokens</span><span class="stat-value">${formatTokens(today.total_tokens || 0)}</span></div>
  </div>
  <div class="card glass">
    <div class="card-title">All Time (since ${escapeHtml2(since)})</div>
    <div class="stat-row"><span class="stat-label">Prompts</span><span class="stat-value">${stats.total_prompts || 0}</span></div>
    <div class="stat-row"><span class="stat-label">Requests</span><span class="stat-value">${stats.total_requests || 0}</span></div>
    <div class="stat-row"><span class="stat-label">Total Tokens</span><span class="stat-value">${formatTokens(stats.total_tokens || 0)}</span></div>
    <div class="stat-row"><span class="stat-label">Prompt Tokens</span><span class="stat-value">${formatTokens(stats.total_prompt_tokens || 0)}</span></div>
    <div class="stat-row"><span class="stat-label">Completion Tokens</span><span class="stat-value">${formatTokens(stats.total_completion_tokens || 0)}</span></div>
  </div>`;
    el.innerHTML = html;
  }
  var cachedContextInfo = null;
  function renderContextCard(ctx) {
    if (!ctx)
      return "";
    cachedContextInfo = ctx;
    var wd = ctx.work_dir || "—";
    var pwd = ctx.plan_work_dir || "—";
    var ws = ctx.workspace || "—";
    var filesHtml = "";
    if (ctx.bootstrap && ctx.bootstrap.length) {
      filesHtml = ctx.bootstrap.map(function(b) {
        var path = b.path ? escapeHtml2(b.path) : "—";
        var scope = b.scope === "global" ? "global" : "project";
        var found = b.path ? "var(--text)" : "var(--hint)";
        return `<div style="display:flex;gap:8px;padding:2px 0;font-size:12px">
        <span style="min-width:90px;font-weight:600;color:${found}">${escapeHtml2(b.name)}</span>
        <span style="color:var(--hint);flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" title="${path}">${path}</span>
        <span style="color:var(--hint);font-size:11px">${scope}</span>
      </div>`;
      }).join("");
    }
    return `<div class="card glass">
    <div class="card-title">Context</div>
    <div style="font-size:12px">
      <div class="stat-row"><span class="stat-label">workDir</span><span class="stat-value" style="font-size:12px;overflow:hidden;text-overflow:ellipsis" title="${escapeHtml2(wd)}">${escapeHtml2(wd)}</span></div>
      <div class="stat-row"><span class="stat-label">planWorkDir</span><span class="stat-value" style="font-size:12px;overflow:hidden;text-overflow:ellipsis" title="${escapeHtml2(pwd)}">${escapeHtml2(pwd)}</span></div>
      <div class="stat-row"><span class="stat-label">workspace</span><span class="stat-value" style="font-size:12px;overflow:hidden;text-overflow:ellipsis" title="${escapeHtml2(ws)}">${escapeHtml2(ws)}</span></div>
    </div>
    <div style="margin-top:8px">${filesHtml}</div>
    <div style="margin-top:8px;text-align:center">
      <button onclick="toggleSystemPrompt()" style="background:var(--secondary-bg);color:var(--text);border:none;padding:6px 12px;border-radius:8px;font-size:12px;cursor:pointer" id="prompt-toggle-btn">Show System Prompt</button>
    </div>
    <pre id="system-prompt-view" style="display:none;margin-top:8px;font-size:11px;max-height:400px;overflow:auto;background:var(--secondary-bg);padding:8px;border-radius:6px;white-space:pre-wrap;word-break:break-word"></pre>
  </div>`;
  }
  function toggleSystemPrompt() {
    var view = document.getElementById("system-prompt-view");
    var btn = document.getElementById("prompt-toggle-btn");
    if (!view || !btn)
      return;
    if (view.style.display === "none") {
      btn.textContent = "Loading...";
      apiFetch("/miniapp/api/prompt").then(function(data) {
        view.textContent = data.prompt || "(empty)";
        view.style.display = "block";
        btn.textContent = "Hide System Prompt";
      }).catch(function() {
        btn.textContent = "Show System Prompt";
      });
    } else {
      view.style.display = "none";
      btn.textContent = "Show System Prompt";
    }
  }
  function renderContextFromData(ctx) {
    var el = document.getElementById("context-content");
    if (el)
      el.innerHTML = renderContextCard(ctx);
  }
  function loadSession() {
    return loadTab("session-loading", "session-content", "session", function() {
      return Promise.all([
        apiFetch("/miniapp/api/session"),
        apiFetch("/miniapp/api/sessions").catch(function() {
          return [];
        }),
        apiFetch("/miniapp/api/context").catch(function() {
          return null;
        }),
        apiFetch("/miniapp/api/sessions/graph").catch(function() {
          return null;
        })
      ]);
    }, function(results) {
      renderSessionFromData(results[1], results[0]);
      renderContextFromData(results[2]);
      renderSessionGraph(results[3]);
    });
  }
  function renderSessionGraph(graph) {
    var el = document.getElementById("session-graph");
    if (!el)
      return;
    if (!graph || !graph.nodes || graph.nodes.length === 0) {
      el.classList.add("hidden");
      return;
    }
    el.classList.remove("hidden");
    var childrenMap = {};
    var roots = [];
    graph.nodes.forEach(function(n) {
      childrenMap[n.key] = [];
    });
    graph.edges.forEach(function(e) {
      if (childrenMap[e.from])
        childrenMap[e.from].push(e.to);
    });
    var nodeMap = {};
    graph.nodes.forEach(function(n) {
      nodeMap[n.key] = n;
      var isChild = graph.edges.some(function(e) {
        return e.to === n.key;
      });
      if (!isChild)
        roots.push(n.key);
    });
    function renderTreeNode(key) {
      var n = nodeMap[key];
      if (!n)
        return "";
      var icon = n.status === "completed" ? "✓" : "●";
      var iconClass = n.status === "completed" ? "completed" : "active";
      var label = n.label || n.short_key || n.key;
      var kids = childrenMap[key] || [];
      var childHtml = "";
      if (kids.length > 0) {
        childHtml = '<ul class="session-tree-children">' + kids.map(renderTreeNode).join("") + "</ul>";
      }
      return '<li class="session-tree-node">' + '<span class="session-tree-icon ' + iconClass + '">' + icon + "</span>" + '<span class="session-tree-label">' + escapeHtml2(label) + "</span>" + '<span class="session-tree-meta">turns=' + n.turn_count + "</span>" + childHtml + "</li>";
    }
    var html = '<div class="card glass"><div class="card-title">Session Graph</div>' + '<ul class="session-tree">' + roots.map(renderTreeNode).join("") + "</ul></div>";
    el.innerHTML = html;
  }
  function formatTokens(n) {
    if (n >= 1e6)
      return (n / 1e6).toFixed(1) + "M";
    if (n >= 1000)
      return (n / 1000).toFixed(1) + "K";
    return String(n);
  }
  function escapeHtml2(s) {
    if (!s)
      return "";
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }
  function escapeAttr(s) {
    if (!s)
      return "";
    return s.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
  var gitSelectedRepo = null;
  function loadGit() {
    gitSelectedRepo = null;
    return loadTab("git-loading", "git-content", "git", function() {
      return Promise.all([
        apiFetch("/miniapp/api/git"),
        apiFetch("/miniapp/api/worktrees").catch(function() {
          return [];
        })
      ]);
    }, function(results) {
      renderGitRepos(results[0], results[1]);
    });
  }
  function renderWorktrees(worktrees) {
    var items = Array.isArray(worktrees) ? worktrees : [];
    var html = '<div class="card glass"><div class="card-title">Worktrees</div>';
    if (items.length === 0) {
      html += '<div class="empty-state" style="padding:12px 0 4px">No active worktrees.</div>';
      html += "</div>";
      return html;
    }
    html += '<div class="worktree-list">';
    items.forEach(function(wt) {
      var dirtyClass = wt.has_uncommitted ? " dirty" : "";
      var dirtyBadge = wt.has_uncommitted ? '<span class="worktree-dirty">DIRTY</span>' : '<span class="worktree-clean">CLEAN</span>';
      var last = "(no commits)";
      if (wt.last_commit_hash) {
        last = wt.last_commit_hash + " " + (wt.last_commit_subject || "");
        if (wt.last_commit_age)
          last += " (" + wt.last_commit_age + ")";
      }
      html += '<div class="worktree-item' + dirtyClass + '">' + '<div class="worktree-main">' + '<div class="worktree-name-row">' + '<span class="worktree-name">' + escapeHtml2(wt.name) + "</span>" + dirtyBadge + "</div>" + '<div class="worktree-branch">' + escapeHtml2(wt.branch || "?") + "</div>" + '<div class="worktree-last">' + escapeHtml2(last) + "</div>" + "</div>" + '<div class="worktree-actions">' + '<button class="worktree-btn merge" data-wt-action="merge" data-wt-name="' + escapeAttr(wt.name) + '">Merge</button>' + '<button class="worktree-btn dispose" data-wt-action="dispose" data-wt-name="' + escapeAttr(wt.name) + '" data-wt-dirty="' + (wt.has_uncommitted ? "1" : "0") + '">Dispose</button>' + "</div>" + "</div>";
    });
    html += "</div></div>";
    return html;
  }
  function renderGitRepos(repos, worktrees) {
    var loading = document.getElementById("git-loading");
    var el = document.getElementById("git-content");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    var html = renderWorktrees(worktrees);
    if (!repos || repos.length === 0) {
      html += '<div class="empty-state" style="margin-top:12px">No git repositories found.</div>';
      el.innerHTML = html;
      return;
    }
    html += '<div style="padding:10px 4px 8px;font-size:12px;color:var(--hint)">Repositories</div>';
    html += repos.map(function(r) {
      return '<div class="git-repo-item glass glass-interactive" data-repo="' + escapeAttr(r.name) + '">' + '<div class="git-repo-body">' + '<div class="git-repo-name">' + escapeHtml2(r.name) + "</div>" + '<div class="git-repo-branch">' + escapeHtml2(r.branch || "?") + "</div>" + "</div>" + '<span class="git-repo-arrow">›</span>' + "</div>";
    }).join("");
    el.innerHTML = html;
  }
  async function postWorktreeAction(action, name, force) {
    var res = await fetch(API_BASE + "/miniapp/api/worktrees?initData=" + encodeURIComponent(initData), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action, name, force: !!force })
    });
    var data = {};
    try {
      data = await res.json();
    } catch (e) {}
    if (!res.ok) {
      throw new Error(data.error || "API error: " + res.status);
    }
    return data;
  }
  document.getElementById("git-content").addEventListener("click", async function(e) {
    var wtBtn = e.target.closest("[data-wt-action]");
    if (wtBtn) {
      var action = wtBtn.dataset.wtAction;
      var name = wtBtn.dataset.wtName;
      var isDirty = wtBtn.dataset.wtDirty === "1";
      var force = false;
      if (action === "merge") {
        if (!confirm('Merge "' + name + '" into base branch?'))
          return;
      } else if (action === "dispose") {
        if (isDirty) {
          if (!confirm('"' + name + '" has uncommitted changes. Force dispose and auto-commit before removal?'))
            return;
          force = true;
        } else if (!confirm('Dispose worktree "' + name + '"?')) {
          return;
        }
      }
      var originalText = wtBtn.textContent;
      wtBtn.disabled = true;
      wtBtn.textContent = action === "merge" ? "Merging..." : "Disposing...";
      try {
        await postWorktreeAction(action, name, force);
        await loadGit();
      } catch (err) {
        alert(err.message || "Action failed");
        wtBtn.disabled = false;
        wtBtn.textContent = originalText;
      }
      return;
    }
    var item = e.target.closest(".git-repo-item");
    if (!item)
      return;
    loadGitDetail(item.dataset.repo);
  });
  function loadGitDetail(name) {
    gitSelectedRepo = name;
    return loadTab("git-loading", "git-content", name, function() {
      return apiFetch("/miniapp/api/git?repo=" + encodeURIComponent(name));
    }, renderGitDetail);
  }
  function renderGitDetail(repo) {
    var loading = document.getElementById("git-loading");
    var el = document.getElementById("git-content");
    loading.classList.add("hidden");
    el.classList.remove("hidden");
    var html = '<button class="git-back-btn" onclick="loadGit()">← ' + escapeHtml2(repo.name || gitSelectedRepo) + "</button>";
    html += '<div class="card glass"><div class="card-title">' + escapeHtml2(repo.name) + " &mdash; " + escapeHtml2(repo.branch || "?") + "</div>";
    if (repo.modified && repo.modified.length > 0) {
      html += '<div style="padding:4px 12px 8px;font-size:12px;color:var(--hint)">Changes (' + repo.modified.length + ")</div>";
      repo.modified.forEach(function(f) {
        html += '<div class="git-commit">' + '<span class="git-status git-status-' + (f.status === "??" ? "u" : f.status.toLowerCase()) + '">' + escapeHtml2(f.status) + "</span>" + '<span class="git-subject">' + escapeHtml2(f.path) + "</span>" + "</div>";
      });
    }
    if (repo.commits && repo.commits.length > 0) {
      html += '<div style="padding:4px 12px 8px;font-size:12px;color:var(--hint)">Commits</div>';
      repo.commits.forEach(function(c) {
        html += '<div class="git-commit">' + '<span class="git-hash">' + escapeHtml2(c.hash) + "</span>" + '<span class="git-subject">' + escapeHtml2(c.subject) + "</span>" + '<span class="git-meta">' + escapeHtml2(c.date) + "</span>" + "</div>";
      });
    } else {
      html += '<div style="padding:12px;color:var(--hint)">No commits found.</div>';
    }
    html += "</div>";
    el.innerHTML = html;
  }
  var devActiveId = "";
  function renderDevFromData(data) {
    var dot = document.getElementById("dev-dot");
    var headerTarget = document.getElementById("dev-header-target");
    var targetsList = document.getElementById("dev-targets-list");
    var iframeWrap = document.getElementById("dev-iframe-wrap");
    var iframe = document.getElementById("dev-iframe");
    var targets = data.targets || [];
    devActiveId = data.active_id || "";
    if (data.active) {
      dot.classList.add("on");
      headerTarget.textContent = data.target ? data.target.replace(/^https?:\/\//, "") : "";
      iframeWrap.classList.remove("hidden");
      var iframeSrc = location.origin + "/miniapp/dev/";
      if (iframe.src !== iframeSrc)
        iframe.src = iframeSrc;
    } else {
      dot.classList.remove("on");
      headerTarget.textContent = "";
      iframeWrap.classList.add("hidden");
      iframe.src = "";
    }
    if (targets.length === 0) {
      targetsList.innerHTML = '<div class="empty-state">No targets registered.<br>Ask the agent to start a dev server.</div>';
      return;
    }
    targetsList.innerHTML = targets.map(function(t) {
      var isActive = t.id === devActiveId;
      var activeClass = isActive ? " active" : "";
      var dotClass = isActive ? " on" : "";
      var displayUrl = t.target.replace(/^https?:\/\//, "");
      return '<div class="dev-target-item glass glass-interactive' + activeClass + '" data-dev-id="' + escapeAttr(t.id) + '">' + '<span class="dev-target-dot' + dotClass + '"></span>' + '<span class="dev-target-name">' + escapeHtml2(t.name) + "</span>" + '<span class="dev-target-url">' + escapeHtml2(displayUrl) + "</span>" + '<span class="dev-target-delete" data-del-id="' + escapeAttr(t.id) + '" data-del-name="' + escapeAttr(t.name) + '">&times;</span>' + "</div>";
    }).join("");
  }
  document.getElementById("dev-targets-list").addEventListener("click", function(e) {
    var delBtn = e.target.closest(".dev-target-delete");
    if (delBtn) {
      e.stopPropagation();
      var id = delBtn.dataset.delId;
      var name = delBtn.dataset.delName;
      if (confirm('Remove "' + name + '"?')) {
        postDevUnregister(id);
      }
      return;
    }
    var card = e.target.closest("[data-dev-id]");
    if (!card)
      return;
    postDevAction(card.dataset.devId);
  });
  async function postDevAction(id) {
    var action = id === devActiveId ? "deactivate" : "activate";
    var body = action === "activate" ? { action: "activate", id } : { action: "deactivate" };
    try {
      var res = await fetch(API_BASE + "/miniapp/api/dev?initData=" + encodeURIComponent(initData), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body)
      });
      var data = await res.json();
      if (!data.error)
        renderDevFromData(data);
    } catch (e) {}
  }
  async function postDevUnregister(id) {
    try {
      var res = await fetch(API_BASE + "/miniapp/api/dev?initData=" + encodeURIComponent(initData), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "unregister", id })
      });
      var data = await res.json();
      if (!data.error)
        renderDevFromData(data);
    } catch (e) {}
  }
  function loadDev() {
    apiFetch("/miniapp/api/dev").then(renderDevFromData).catch(function() {});
  }
  var eventSource = null;
  function connectSSE() {
    if (eventSource)
      eventSource.close();
    eventSource = new EventSource(API_BASE + "/miniapp/api/events?initData=" + encodeURIComponent(initData));
    eventSource.addEventListener("plan", function(e) {
      try {
        lastSSE.plan = Date.now();
        renderPlanFromData(JSON.parse(e.data));
      } catch (err) {}
    });
    eventSource.addEventListener("session", function(e) {
      try {
        lastSSE.session = Date.now();
        var d = JSON.parse(e.data);
        renderSessionFromData(d.sessions, d.stats);
        if (d.graph)
          renderSessionGraph(d.graph);
      } catch (err) {}
    });
    eventSource.addEventListener("skills", function(e) {
      try {
        lastSSE.skills = Date.now();
        renderSkillsFromData(JSON.parse(e.data));
      } catch (err) {}
    });
    eventSource.addEventListener("dev", function(e) {
      try {
        lastSSE.dev = Date.now();
        renderDevFromData(JSON.parse(e.data));
      } catch (err) {}
    });
    eventSource.addEventListener("context", function(e) {
      try {
        renderContextFromData(JSON.parse(e.data));
      } catch (err) {}
    });
    eventSource.addEventListener("prompt", function(e) {
      try {
        var d = JSON.parse(e.data);
        var view = document.getElementById("system-prompt-view");
        if (view && view.style.display !== "none") {
          view.textContent = d.prompt || "(empty)";
        }
      } catch (err) {}
    });
    eventSource.onerror = function() {};
  }
  connectSSE();
  loadPlan();
  var logsWs = null;
  var logsComponent = "";
  var logsEntries = [];
  var logsReconnectTimer = null;
  var logsPage = 1;
  var LOGS_PAGE_SIZE = 60;
  function connectLogsWs() {
    if (logsWs && logsWs.readyState <= 1)
      return;
    var wsProto = location.protocol === "https:" ? "wss:" : "ws:";
    var wsUrl = wsProto + "//" + location.host + "/miniapp/api/logs/ws?initData=" + encodeURIComponent(initData);
    if (logsComponent)
      wsUrl += "&component=" + encodeURIComponent(logsComponent);
    logsWs = new WebSocket(wsUrl);
    var statusDot = document.getElementById("logs-status");
    logsWs.onopen = function() {
      statusDot.classList.add("on");
    };
    logsWs.onmessage = function(e) {
      var msg = JSON.parse(e.data);
      if (msg.type === "init") {
        logsEntries = msg.entries || [];
        logsPage = 1;
      } else if (msg.type === "entry") {
        logsEntries.push(msg.entry);
        if (logsEntries.length > 200)
          logsEntries.shift();
      }
      renderLogs2();
    };
    logsWs.onclose = function() {
      statusDot.classList.remove("on");
      logsWs = null;
      var activeTab = document.querySelector(".tab.active");
      if (activeTab && activeTab.dataset.panel === "config") {
        logsReconnectTimer = setTimeout(connectLogsWs, 3000);
      }
    };
    logsWs.onerror = function() {};
  }
  function disconnectLogsWs() {
    if (logsReconnectTimer) {
      clearTimeout(logsReconnectTimer);
      logsReconnectTimer = null;
    }
    if (logsWs) {
      logsWs.close();
      logsWs = null;
    }
    document.getElementById("logs-status").classList.remove("on");
  }
  function renderLogs2() {
    var container = document.getElementById("logs-content");
    if (!container)
      return;
    var wasScrolledToBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 30;
    var view = renderLogs(logsEntries, {
      component: logsComponent,
      page: logsPage,
      pageSize: LOGS_PAGE_SIZE
    });
    if (logsPage > view.totalPages) {
      logsPage = view.totalPages;
      view = renderLogs(logsEntries, {
        component: logsComponent,
        page: logsPage,
        pageSize: LOGS_PAGE_SIZE
      });
    }
    if (!view.html) {
      container.innerHTML = '<div class="empty-state">No logs.</div>';
    } else {
      container.innerHTML = view.html;
    }
    updateLogsPager(view);
    if (wasScrolledToBottom)
      container.scrollTop = container.scrollHeight;
  }
  function updateLogsPager(view) {
    var info = document.getElementById("logs-page-info");
    var prev = document.getElementById("logs-page-prev");
    var next = document.getElementById("logs-page-next");
    if (!info || !prev || !next)
      return;
    info.textContent = view.currentPage + "/" + view.totalPages + " (" + view.totalItems + ")";
    prev.disabled = view.currentPage <= 1;
    next.disabled = view.currentPage >= view.totalPages;
  }
  document.querySelector(".log-filter-chips").addEventListener("click", function(e) {
    var chip = e.target.closest(".log-filter-chip");
    if (!chip)
      return;
    document.querySelectorAll(".log-filter-chip").forEach(function(c) {
      c.classList.remove("active");
    });
    chip.classList.add("active");
    logsComponent = chip.dataset.component || "";
    logsPage = 1;
    logsEntries = [];
    renderLogs2();
    disconnectLogsWs();
    connectLogsWs();
  });
  var logsPrevButton = document.getElementById("logs-page-prev");
  if (logsPrevButton) {
    logsPrevButton.addEventListener("click", function() {
      if (logsPage <= 1)
        return;
      logsPage--;
      renderLogs2();
    });
  }
  var logsNextButton = document.getElementById("logs-page-next");
  if (logsNextButton) {
    logsNextButton.addEventListener("click", function() {
      logsPage++;
      renderLogs2();
    });
  }
  var orchCanvas = null;
  var orchCtx = null;
  var orchInited = false;
  var orchWs = null;
  var orchReconnectTimer = null;
  var _orchLastTs = null;
  var _orchBOB = [0, -1, -2, -1];
  var _orchFRAME_MS = { idle: 450, waiting: 650, toolcall: 90, talking: 280, entering: 220, exiting: 220 };
  var _orchWALK = 55;
  var _orchConductor;
  var _orchSecretary;
  var _orchHeartbeat;
  var _orchSubagents;
  var _orchSlots;
  var _orchFreeSlots;
  function _orchMakeChar(id, emoji, home) {
    return {
      id,
      emoji,
      x: home.x,
      y: home.y,
      home,
      target: null,
      state: "idle",
      frame: 0,
      frameTimer: 0,
      bubble: null,
      alive: false,
      _onArrive: null
    };
  }
  function _orchInitChars() {
    _orchConductor = _orchMakeChar("conductor", "\uD83D\uDC51", MAP_POSITIONS.conductor);
    _orchSecretary = _orchMakeChar("secretary", "\uD83D\uDC69‍\uD83D\uDCBC", MAP_POSITIONS.secretary);
    _orchHeartbeat = _orchMakeChar("heartbeat", "\uD83D\uDD4A️", MAP_POSITIONS.heartbeat || { x: 230, y: 58 });
    _orchConductor.alive = true;
    _orchSecretary.alive = false;
    _orchHeartbeat.alive = true;
    _orchConductor.statusText = null;
    _orchHeartbeat.facing = 1;
    _orchHeartbeat.flipTimer = 0;
    var ps = [
      { id: "s0", emoji: "\uD83D\uDD0D" },
      { id: "s1", emoji: "\uD83D\uDCCA" },
      { id: "s2", emoji: "\uD83D\uDCBB" },
      { id: "s3", emoji: "\uD83D\uDD27" },
      { id: "s4", emoji: "\uD83C\uDFAF" }
    ];
    _orchSubagents = ps.map(function(p, i) {
      var c = _orchMakeChar(p.id, p.emoji, MAP_POSITIONS.stations[i]);
      c.x = MAP_POSITIONS.door.x;
      c.y = MAP_POSITIONS.door.y;
      return c;
    });
    _orchSlots = {};
    _orchFreeSlots = _orchSubagents.slice();
  }
  function _orchAllChars() {
    return [_orchConductor, _orchSecretary, _orchHeartbeat].concat(_orchSubagents);
  }
  function _orchSyncBadge(id, state, alive) {
    var el = document.getElementById("orch-badge-" + id);
    if (!el)
      return;
    el.className = "orch-badge" + (alive ? " alive" : "") + (state === "talking" ? " talking" : "") + (state === "toolcall" ? " toolcall" : "") + (state === "waiting" ? " waiting" : "");
  }
  function _orchSetState(c, state, tool) {
    c.state = state;
    _orchSyncBadge(c.id, state, c.alive);
    if (c === _orchConductor) {
      if (state === "waiting")
        c.statusText = "\uD83E\uDD14";
      else if (state === "toolcall")
        c.statusText = "⌨";
      else if (state === "user_waiting")
        c.statusText = "⏳";
      else if (state === "plan_interviewing")
        c.statusText = "\uD83D\uDCCB";
      else if (state === "plan_review")
        c.statusText = "\uD83D\uDD0D";
      else if (state === "plan_executing")
        c.statusText = "▶️";
      else if (state === "plan_completed")
        c.statusText = "✅";
      else
        c.statusText = null;
      var inPlan = state.indexOf("plan_") === 0;
      if (_orchSecretary.alive !== inPlan) {
        _orchSecretary.alive = inPlan;
        _orchSyncBadge("secretary", _orchSecretary.state, _orchSecretary.alive);
      }
    }
  }
  function _orchMoveTo(c, pos, cb) {
    c.target = pos;
    c._onArrive = cb || null;
  }
  function _orchSay(c, text, ttl) {
    c.bubble = { text, ttl: ttl || 2200 };
  }
  function _orchCharForId(id) {
    if (id === "heartbeat")
      return _orchHeartbeat;
    if (_orchSlots[id])
      return _orchSlots[id];
    return _orchConductor;
  }
  function _orchSpawn(id) {
    if (/^subagent-/.test(id)) {
      var c = _orchFreeSlots.shift();
      if (!c)
        return;
      _orchSlots[id] = c;
      c.alive = true;
      c.x = MAP_POSITIONS.door.x;
      c.y = MAP_POSITIONS.door.y;
      _orchSetState(c, "entering");
      _orchMoveTo(c, c.home, function() {
        _orchSetState(c, "idle");
      });
    } else {
      var ch = _orchCharForId(id);
      ch.alive = true;
      _orchSetState(ch, "waiting");
    }
  }
  function _orchGC(id) {
    if (/^subagent-/.test(id)) {
      var c = _orchSlots[id];
      if (!c)
        return;
      delete _orchSlots[id];
      _orchFreeSlots.push(c);
      _orchSetState(c, "exiting");
      _orchMoveTo(c, MAP_POSITIONS.door, function() {
        c.alive = false;
        _orchSetState(c, "idle");
      });
    } else {
      var ch = _orchCharForId(id);
      if (ch === _orchHeartbeat) {
        _orchSetState(ch, "idle");
      } else if (ch === _orchConductor) {
        _orchSetState(ch, "user_waiting");
      } else {
        ch.alive = false;
        _orchSetState(ch, "idle");
      }
    }
  }
  function _orchConverse(fromId, toId, text) {
    var from = _orchCharForId(fromId), to = _orchCharForId(toId);
    if (!from || !to || from === to)
      return;
    var label = (text || "").slice(0, 18);
    var mid = { x: (from.x + to.x) / 2, y: (from.y + to.y) / 2 };
    _orchSetState(from, "talking");
    _orchSetState(to, "talking");
    _orchMoveTo(from, { x: mid.x - 18, y: mid.y }, function() {
      _orchSay(from, label, 2400);
    });
    _orchMoveTo(to, { x: mid.x + 18, y: mid.y }, function() {
      setTimeout(function() {
        _orchMoveTo(from, from.home, function() {
          _orchSetState(from, "idle");
        });
        _orchMoveTo(to, to.home, function() {
          _orchSetState(to, "idle");
        });
      }, 2600);
    });
  }
  function _orchUpdate(dt) {
    _orchAllChars().forEach(function(c) {
      if (!c.alive && c.state !== "entering")
        return;
      if (c === _orchHeartbeat) {
        if (c.state === "idle") {
          c.frame = 0;
        } else {
          c.frameTimer += dt;
          var pDur = c.state === "toolcall" ? 130 : 380;
          if (c.frameTimer >= pDur) {
            c.frame = (c.frame + 1) % 4;
            c.frameTimer -= pDur;
          }
        }
      } else {
        c.frameTimer += dt;
        var dur = _orchFRAME_MS[c.state] || 450;
        if (c.frameTimer >= dur) {
          c.frame = (c.frame + 1) % 4;
          c.frameTimer -= dur;
        }
      }
      if (c.target) {
        var dx = c.target.x - c.x, dy = c.target.y - c.y, dist = Math.sqrt(dx * dx + dy * dy);
        if (dist > 1.5) {
          var spd = _orchWALK * dt / 1000;
          c.x += dx / dist * spd;
          c.y += dy / dist * spd;
        } else {
          c.x = c.target.x;
          c.y = c.target.y;
          c.target = null;
          if (c._onArrive) {
            c._onArrive();
            c._onArrive = null;
          }
        }
      }
      if (c.bubble) {
        c.bubble.ttl -= dt;
        if (c.bubble.ttl <= 0)
          c.bubble = null;
      }
      if (c === _orchHeartbeat) {
        if (c.target) {
          var pdx = c.target.x - c.x;
          if (Math.abs(pdx) > 1)
            c.facing = pdx > 0 ? 1 : -1;
        } else {
          var flipRate = c.state === "toolcall" ? 280 : c.state === "waiting" ? 600 : 2800;
          c.flipTimer += dt;
          if (c.flipTimer >= flipRate) {
            c.flipTimer -= flipRate;
            c.facing = -c.facing;
          }
        }
      }
    });
  }
  function _orchDrawStatus(c) {
    if (!c.statusText)
      return;
    var yOff = _orchBOB[c.frame], cx = Math.floor(c.x), cy = Math.floor(c.y + yOff) - 20;
    orchCtx.font = "11px serif";
    orchCtx.textAlign = "center";
    orchCtx.textBaseline = "middle";
    orchCtx.fillText(c.statusText, cx, cy);
  }
  function _orchDrawBubble(c) {
    if (!c.bubble)
      return;
    var yOff = _orchBOB[c.frame], bx = c.x, by = c.y + yOff - 18;
    orchCtx.font = "7px Silkscreen,monospace";
    var tw = orchCtx.measureText(c.bubble.text).width, pw = tw + 8, ph = 12;
    var lx = Math.max(4, Math.min(316 - pw, bx - pw / 2));
    orchCtx.fillStyle = "#facc15";
    orchCtx.fillRect(Math.floor(lx), Math.floor(by - ph), Math.ceil(pw), Math.ceil(ph));
    orchCtx.fillRect(Math.floor(bx) - 1, Math.floor(by), 3, 3);
    orchCtx.fillStyle = "#0a0a00";
    orchCtx.textAlign = "left";
    orchCtx.textBaseline = "middle";
    orchCtx.fillText(c.bubble.text, Math.floor(lx + 4), Math.floor(by - ph / 2));
  }
  function _orchDrawChar(c) {
    if (!c.alive && c.state !== "entering" && c.state !== "exiting")
      return;
    var yOff = _orchBOB[c.frame], cx = Math.floor(c.x), cy = Math.floor(c.y + yOff);
    if (c.state === "toolcall") {
      orchCtx.fillStyle = "rgba(251,146,60,0.35)";
      orchCtx.beginPath();
      orchCtx.arc(cx, cy, 13, 0, Math.PI * 2);
      orchCtx.fill();
    } else if (c.state === "waiting") {
      orchCtx.fillStyle = "rgba(96,165,250,0.25)";
      orchCtx.beginPath();
      orchCtx.arc(cx, cy, 11, 0, Math.PI * 2);
      orchCtx.fill();
    } else if (c.state === "user_waiting" || c.state === "plan_review") {
      orchCtx.fillStyle = "rgba(167,139,250,0.18)";
      orchCtx.beginPath();
      orchCtx.arc(cx, cy, 10, 0, Math.PI * 2);
      orchCtx.fill();
    } else if (c.state === "plan_executing") {
      orchCtx.fillStyle = "rgba(74,222,128,0.18)";
      orchCtx.beginPath();
      orchCtx.arc(cx, cy, 10, 0, Math.PI * 2);
      orchCtx.fill();
    }
    orchCtx.font = "18px serif";
    orchCtx.textAlign = "center";
    orchCtx.textBaseline = "middle";
    if (c.facing === -1) {
      orchCtx.save();
      orchCtx.translate(cx, cy);
      orchCtx.scale(-1, 1);
      orchCtx.fillText(c.emoji, 0, 0);
      orchCtx.restore();
    } else {
      orchCtx.fillText(c.emoji, cx, cy);
    }
    orchCtx.font = "6px Silkscreen,monospace";
    orchCtx.textAlign = "center";
    orchCtx.textBaseline = "top";
    orchCtx.fillStyle = c.state === "talking" ? "#facc15" : "#3a4a7a";
    orchCtx.fillText(c.id.toUpperCase(), cx, cy + 11);
    _orchDrawStatus(c);
    _orchDrawBubble(c);
  }
  function _orchRender(ts) {
    if (_orchLastTs === null)
      _orchLastTs = ts;
    var dt = Math.min(ts - _orchLastTs, 80);
    _orchLastTs = ts;
    _orchUpdate(dt);
    orchCtx.imageSmoothingEnabled = false;
    drawMap(orchCtx);
    _orchAllChars().forEach(_orchDrawChar);
    requestAnimationFrame(_orchRender);
  }
  function orchInit() {
    if (orchInited)
      return;
    orchInited = true;
    orchCanvas = document.getElementById("orch-canvas");
    orchCtx = orchCanvas.getContext("2d");
    orchCtx.imageSmoothingEnabled = false;
    _orchInitChars();
    loadMapAsset(function() {
      _orchLastTs = null;
      requestAnimationFrame(_orchRender);
    });
  }
  function connectOrchWs() {
    orchInit();
    if (orchWs && orchWs.readyState <= 1)
      return;
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    var url = proto + "//" + location.host + "/miniapp/api/orchestration/ws?initData=" + encodeURIComponent(initData);
    orchWs = new WebSocket(url);
    orchWs.onopen = function() {
      document.getElementById("orch-status-dot").classList.add("on");
      document.getElementById("orch-status-text").textContent = "Live";
    };
    orchWs.onmessage = function(e) {
      var msg;
      try {
        msg = JSON.parse(e.data);
      } catch (_) {
        return;
      }
      if (msg.type === "init") {
        (msg.agents || []).forEach(function(info) {
          _orchSpawn(info.id);
          if (info.state && info.state !== "idle") {
            var c2 = _orchCharForId(info.id);
            if (c2)
              _orchSetState(c2, info.state);
          }
        });
      } else if (msg.type === "event") {
        var ev = msg.event || {};
        if (ev.type === "agent_spawn")
          _orchSpawn(ev.id);
        if (ev.type === "agent_state") {
          var c = _orchCharForId(ev.id);
          if (c)
            _orchSetState(c, ev.state, ev.tool);
        }
        if (ev.type === "agent_gc")
          _orchGC(ev.id);
        if (ev.type === "conversation")
          _orchConverse(ev.from, ev.to, ev.text);
      }
    };
    orchWs.onclose = function() {
      document.getElementById("orch-status-dot").classList.remove("on");
      document.getElementById("orch-status-text").textContent = "Disconnected";
      orchWs = null;
      var at = document.querySelector(".tab.active");
      if (at && at.dataset.panel === "orch")
        orchReconnectTimer = setTimeout(connectOrchWs, 3000);
    };
    orchWs.onerror = function() {};
  }
  function disconnectOrchWs() {
    if (orchReconnectTimer) {
      clearTimeout(orchReconnectTimer);
      orchReconnectTimer = null;
    }
    if (orchWs) {
      orchWs.close();
      orchWs = null;
    }
    var dot = document.getElementById("orch-status-dot");
    var txt = document.getElementById("orch-status-text");
    if (dot)
      dot.classList.remove("on");
    if (txt)
      txt.textContent = "Offline";
  }
  async function saveLogSnapshot() {
    try {
      var res = await fetch(API_BASE + "/miniapp/api/logs/snapshot?initData=" + encodeURIComponent(initData), {
        method: "POST"
      });
      if (!res.ok)
        throw new Error("API error: " + res.status);
      var data = await res.json();
      if (data.download_url) {
        var a = document.createElement("a");
        a.href = API_BASE + data.download_url + "?initData=" + encodeURIComponent(initData);
        a.download = "";
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
      }
    } catch (e) {}
  }
  window.sendCustomCmd = sendCustomCmd;
  window.sendSkillCommand = sendSkillCommand;
  window.startPlan = startPlan;
  window.toggleSystemPrompt = toggleSystemPrompt;
  window.loadGit = loadGit;
  window.saveLogSnapshot = saveLogSnapshot;
})();
