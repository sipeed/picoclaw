import { useEffect, useRef } from 'preact/hooks';

// Orch canvas character system - ported from vanilla JS
// All character/animation state is kept in module-level vars (same as original)

const BOB = [0, -1, -2, -1];
const FRAME_MS: Record<string, number> = {
  idle: 450,
  waiting: 650,
  toolcall: 90,
  talking: 280,
  entering: 220,
  exiting: 220,
};
const WALK = 55;

interface Char {
  id: string;
  emoji: string;
  x: number;
  y: number;
  home: { x: number; y: number };
  target: { x: number; y: number } | null;
  state: string;
  frame: number;
  frameTimer: number;
  bubble: { text: string; ttl: number } | null;
  alive: boolean;
  _onArrive: (() => void) | null;
  statusText?: string | null;
  facing?: number;
  flipTimer?: number;
}

let conductor: Char;
let secretary: Char;
let heartbeat: Char;
let subagents: Char[];
let slots: Record<string, Char> = {};
let freeSlots: Char[] = [];
let inited = false;
let orchWs: WebSocket | null = null;
let orchReconnectTimer: any = null;
let lastTs: number | null = null;

function makeChar(
  id: string,
  emoji: string,
  home: { x: number; y: number },
): Char {
  return {
    id,
    emoji,
    x: home.x,
    y: home.y,
    home,
    target: null,
    state: 'idle',
    frame: 0,
    frameTimer: 0,
    bubble: null,
    alive: false,
    _onArrive: null,
  };
}

function initChars() {
  const MAP = window.MAP_POSITIONS;
  conductor = makeChar('conductor', '\u{1F451}', MAP.conductor);
  secretary = makeChar('secretary', '\u{1F469}\u{200D}\u{1F4BC}', MAP.secretary);
  heartbeat = makeChar('heartbeat', '\u{1F54A}\uFE0F', MAP.heartbeat || { x: 230, y: 58 });
  conductor.alive = true;
  secretary.alive = false;
  heartbeat.alive = true;
  conductor.statusText = null;
  heartbeat.facing = 1;
  heartbeat.flipTimer = 0;

  const ps = [
    { id: 's0', emoji: '\u{1F50D}' },
    { id: 's1', emoji: '\u{1F4CA}' },
    { id: 's2', emoji: '\u{1F4BB}' },
    { id: 's3', emoji: '\u{1F527}' },
    { id: 's4', emoji: '\u{1F3AF}' },
  ];
  subagents = ps.map((p, i) => {
    const c = makeChar(p.id, p.emoji, MAP.stations[i]);
    c.x = MAP.door.x;
    c.y = MAP.door.y;
    return c;
  });
  slots = {};
  freeSlots = subagents.slice();
}

function allChars(): Char[] {
  return [conductor, secretary, heartbeat, ...subagents];
}

function syncBadge(id: string, state: string, alive: boolean) {
  const el = document.getElementById('orch-badge-' + id);
  if (!el) return;
  el.className =
    'orch-badge' +
    (alive ? ' alive' : '') +
    (state === 'talking' ? ' talking' : '') +
    (state === 'toolcall' ? ' toolcall' : '') +
    (state === 'waiting' ? ' waiting' : '');
}

function setState(c: Char, state: string) {
  c.state = state;
  syncBadge(c.id, state, c.alive);
  if (c === conductor) {
    if (state === 'waiting') c.statusText = '\u{1F914}';
    else if (state === 'toolcall') c.statusText = '\u2328';
    else if (state === 'user_waiting') c.statusText = '\u23F3';
    else if (state === 'plan_interviewing') c.statusText = '\u{1F4CB}';
    else if (state === 'plan_review') c.statusText = '\u{1F50D}';
    else if (state === 'plan_executing') c.statusText = '\u25B6\uFE0F';
    else if (state === 'plan_completed') c.statusText = '\u2705';
    else c.statusText = null;

    const inPlan = state.indexOf('plan_') === 0;
    if (secretary.alive !== inPlan) {
      secretary.alive = inPlan;
      syncBadge('secretary', secretary.state, secretary.alive);
    }
  }
}

function moveTo(c: Char, pos: { x: number; y: number }, cb?: () => void) {
  c.target = pos;
  c._onArrive = cb || null;
}

function say(c: Char, text: string, ttl = 2200) {
  c.bubble = { text, ttl };
}

function charForId(id: string): Char {
  if (id === 'heartbeat') return heartbeat;
  if (slots[id]) return slots[id];
  return conductor;
}

function spawn(id: string) {
  if (/^subagent-/.test(id)) {
    const c = freeSlots.shift();
    if (!c) return;
    slots[id] = c;
    c.alive = true;
    c.x = window.MAP_POSITIONS.door.x;
    c.y = window.MAP_POSITIONS.door.y;
    setState(c, 'entering');
    moveTo(c, c.home, () => setState(c, 'idle'));
  } else {
    const ch = charForId(id);
    ch.alive = true;
    setState(ch, 'waiting');
  }
}

function gc(id: string) {
  if (/^subagent-/.test(id)) {
    const c = slots[id];
    if (!c) return;
    delete slots[id];
    freeSlots.push(c);
    setState(c, 'exiting');
    moveTo(c, window.MAP_POSITIONS.door, () => {
      c.alive = false;
      setState(c, 'idle');
    });
  } else {
    const ch = charForId(id);
    if (ch === heartbeat) {
      setState(ch, 'idle');
    } else if (ch === conductor) {
      setState(ch, 'user_waiting');
    } else {
      ch.alive = false;
      setState(ch, 'idle');
    }
  }
}

function converse(fromId: string, toId: string, text: string) {
  const from = charForId(fromId);
  const to = charForId(toId);
  if (!from || !to || from === to) return;
  const label = (text || '').slice(0, 18);
  const mid = { x: (from.x + to.x) / 2, y: (from.y + to.y) / 2 };
  setState(from, 'talking');
  setState(to, 'talking');
  moveTo(from, { x: mid.x - 18, y: mid.y }, () => say(from, label, 2400));
  moveTo(to, { x: mid.x + 18, y: mid.y }, () => {
    setTimeout(() => {
      moveTo(from, from.home, () => setState(from, 'idle'));
      moveTo(to, to.home, () => setState(to, 'idle'));
    }, 2600);
  });
}

function update(dt: number) {
  allChars().forEach((c) => {
    if (!c.alive && c.state !== 'entering') return;
    if (c === heartbeat) {
      if (c.state === 'idle') {
        c.frame = 0;
      } else {
        c.frameTimer += dt;
        const pDur = c.state === 'toolcall' ? 130 : 380;
        if (c.frameTimer >= pDur) {
          c.frame = (c.frame + 1) % 4;
          c.frameTimer -= pDur;
        }
      }
    } else {
      c.frameTimer += dt;
      const dur = FRAME_MS[c.state] || 450;
      if (c.frameTimer >= dur) {
        c.frame = (c.frame + 1) % 4;
        c.frameTimer -= dur;
      }
    }
    if (c.target) {
      const dx = c.target.x - c.x;
      const dy = c.target.y - c.y;
      const dist = Math.sqrt(dx * dx + dy * dy);
      if (dist > 1.5) {
        const spd = (WALK * dt) / 1000;
        c.x += (dx / dist) * spd;
        c.y += (dy / dist) * spd;
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
      if (c.bubble.ttl <= 0) c.bubble = null;
    }
    if (c === heartbeat) {
      if (c.target) {
        const pdx = c.target.x - c.x;
        if (Math.abs(pdx) > 1) c.facing = pdx > 0 ? 1 : -1;
      } else {
        const flipRate =
          c.state === 'toolcall' ? 280 : c.state === 'waiting' ? 600 : 2800;
        c.flipTimer = (c.flipTimer || 0) + dt;
        if (c.flipTimer >= flipRate) {
          c.flipTimer -= flipRate;
          c.facing = -(c.facing || 1);
        }
      }
    }
  });
}

function drawStatus(ctx: CanvasRenderingContext2D, c: Char) {
  if (!c.statusText) return;
  const yOff = BOB[c.frame];
  const cx = Math.floor(c.x);
  const cy = Math.floor(c.y + yOff) - 20;
  ctx.font = '11px serif';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText(c.statusText, cx, cy);
}

function drawBubble(ctx: CanvasRenderingContext2D, c: Char) {
  if (!c.bubble) return;
  const yOff = BOB[c.frame];
  const bx = c.x;
  const by = c.y + yOff - 18;
  ctx.font = '7px Silkscreen,monospace';
  const tw = ctx.measureText(c.bubble.text).width;
  const pw = tw + 8;
  const ph = 12;
  const lx = Math.max(4, Math.min(316 - pw, bx - pw / 2));
  ctx.fillStyle = '#facc15';
  ctx.fillRect(Math.floor(lx), Math.floor(by - ph), Math.ceil(pw), Math.ceil(ph));
  ctx.fillRect(Math.floor(bx) - 1, Math.floor(by), 3, 3);
  ctx.fillStyle = '#0a0a00';
  ctx.textAlign = 'left';
  ctx.textBaseline = 'middle';
  ctx.fillText(c.bubble.text, Math.floor(lx + 4), Math.floor(by - ph / 2));
}

function drawChar(ctx: CanvasRenderingContext2D, c: Char) {
  if (!c.alive && c.state !== 'entering' && c.state !== 'exiting') return;
  const yOff = BOB[c.frame];
  const cx = Math.floor(c.x);
  const cy = Math.floor(c.y + yOff);

  if (c.state === 'toolcall') {
    ctx.fillStyle = 'rgba(251,146,60,0.35)';
    ctx.beginPath();
    ctx.arc(cx, cy, 13, 0, Math.PI * 2);
    ctx.fill();
  } else if (c.state === 'waiting') {
    ctx.fillStyle = 'rgba(96,165,250,0.25)';
    ctx.beginPath();
    ctx.arc(cx, cy, 11, 0, Math.PI * 2);
    ctx.fill();
  } else if (c.state === 'user_waiting' || c.state === 'plan_review') {
    ctx.fillStyle = 'rgba(167,139,250,0.18)';
    ctx.beginPath();
    ctx.arc(cx, cy, 10, 0, Math.PI * 2);
    ctx.fill();
  } else if (c.state === 'plan_executing') {
    ctx.fillStyle = 'rgba(74,222,128,0.18)';
    ctx.beginPath();
    ctx.arc(cx, cy, 10, 0, Math.PI * 2);
    ctx.fill();
  }

  ctx.font = '18px serif';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  if (c.facing === -1) {
    ctx.save();
    ctx.translate(cx, cy);
    ctx.scale(-1, 1);
    ctx.fillText(c.emoji, 0, 0);
    ctx.restore();
  } else {
    ctx.fillText(c.emoji, cx, cy);
  }
  ctx.font = '6px Silkscreen,monospace';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'top';
  ctx.fillStyle = c.state === 'talking' ? '#facc15' : '#3a4a7a';
  ctx.fillText(c.id.toUpperCase(), cx, cy + 11);
  drawStatus(ctx, c);
  drawBubble(ctx, c);
}

interface OrchCanvasProps {
  active: boolean;
}

export function OrchCanvas({ active }: OrchCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animRef = useRef<number>(0);

  useEffect(() => {
    if (!active) {
      // Disconnect WS when not active
      if (orchReconnectTimer) {
        clearTimeout(orchReconnectTimer);
        orchReconnectTimer = null;
      }
      if (orchWs) {
        orchWs.close();
        orchWs = null;
      }
      if (animRef.current) {
        cancelAnimationFrame(animRef.current);
        animRef.current = 0;
      }
      return;
    }

    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Init characters once
    if (!inited) {
      inited = true;
      initChars();
    }

    // Start render loop
    lastTs = null;
    function renderLoop(ts: number) {
      if (lastTs === null) lastTs = ts;
      const dt = Math.min(ts - lastTs, 80);
      lastTs = ts;
      update(dt);
      ctx!.imageSmoothingEnabled = false;
      window.drawMap(ctx!);
      allChars().forEach((c) => drawChar(ctx!, c));
      animRef.current = requestAnimationFrame(renderLoop);
    }

    window.loadMapAsset(() => {
      lastTs = null;
      animRef.current = requestAnimationFrame(renderLoop);
    });

    // Connect orchestration WebSocket
    connectOrchWs();

    return () => {
      if (animRef.current) {
        cancelAnimationFrame(animRef.current);
        animRef.current = 0;
      }
    };
  }, [active]);

  return (
    <div class="card glass" style={{ marginTop: '16px', padding: '0' }}>
      <div class="orch-room-row">
        <div class="orch-side" id="orch-panel-left">
          <div class="orch-badge alive" id="orch-badge-conductor">
            <div class="orch-badge-emoji">{'\u{1F451}'}</div>
            <div class="orch-badge-label">CNDR</div>
            <div class="orch-badge-dot" />
          </div>
          <div class="orch-badge" id="orch-badge-secretary">
            <div class="orch-badge-emoji">{'\u{1F469}\u{200D}\u{1F4BC}'}</div>
            <div class="orch-badge-label">SEC</div>
            <div class="orch-badge-dot" />
          </div>
          <div class="orch-badge alive" id="orch-badge-heartbeat">
            <div class="orch-badge-emoji">{'\u{1F54A}\uFE0F'}</div>
            <div class="orch-badge-label">HB</div>
            <div class="orch-badge-dot" />
          </div>
        </div>
        <div class="orch-canvas-wrap">
          <canvas ref={canvasRef} id="orch-canvas" width="320" height="320" />
        </div>
        <div class="orch-side" id="orch-panel-right">
          {[
            { id: 's0', emoji: '\u{1F50D}', label: 'SCOUT' },
            { id: 's1', emoji: '\u{1F4CA}', label: 'ANLY' },
            { id: 's2', emoji: '\u{1F4BB}', label: 'CODE' },
            { id: 's3', emoji: '\u{1F527}', label: 'WRKR' },
            { id: 's4', emoji: '\u{1F3AF}', label: 'CORD' },
          ].map((s) => (
            <div class="orch-badge" id={`orch-badge-${s.id}`} key={s.id}>
              <div class="orch-badge-emoji">{s.emoji}</div>
              <div class="orch-badge-label">{s.label}</div>
              <div class="orch-badge-dot" />
            </div>
          ))}
        </div>
      </div>
      <div class="orch-status">
        <span class="orch-dot" id="orch-status-dot" />
        <span id="orch-status-text">Connecting...</span>
      </div>
    </div>
  );
}

function connectOrchWs() {
  if (orchWs && orchWs.readyState <= 1) return;
  const initData = window.Telegram?.WebApp?.initData || '';
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const url =
    proto +
    '//' +
    location.host +
    '/miniapp/api/orchestration/ws?initData=' +
    encodeURIComponent(initData);
  orchWs = new WebSocket(url);

  orchWs.onopen = () => {
    const dot = document.getElementById('orch-status-dot');
    const txt = document.getElementById('orch-status-text');
    if (dot) dot.classList.add('on');
    if (txt) txt.textContent = 'Live';
  };

  orchWs.onmessage = (e) => {
    let msg: any;
    try {
      msg = JSON.parse(e.data);
    } catch {
      return;
    }
    if (msg.type === 'init') {
      (msg.agents || []).forEach((info: any) => {
        spawn(info.id);
        if (info.state && info.state !== 'idle') {
          const c = charForId(info.id);
          if (c) setState(c, info.state);
        }
      });
    } else if (msg.type === 'event') {
      const ev = msg.event || {};
      if (ev.type === 'agent_spawn') spawn(ev.id);
      if (ev.type === 'agent_state') {
        const c = charForId(ev.id);
        if (c) setState(c, ev.state);
      }
      if (ev.type === 'agent_gc') gc(ev.id);
      if (ev.type === 'conversation') converse(ev.from, ev.to, ev.text);
    }
  };

  orchWs.onclose = () => {
    const dot = document.getElementById('orch-status-dot');
    const txt = document.getElementById('orch-status-text');
    if (dot) dot.classList.remove('on');
    if (txt) txt.textContent = 'Disconnected';
    orchWs = null;
  };

  orchWs.onerror = () => {};
}
