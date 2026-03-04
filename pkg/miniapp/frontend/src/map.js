// map.js — Orchestration Room
//
// External asset: drop map.png (320×320px) next to index.html to replace
// the procedural fallback. Character positions (MAP_POSITIONS) are defined
// in canvas-pixel coordinates and remain valid regardless of which rendering
// path is used — just make sure your map.png matches them.
//
// Usage:
//   loadMapAsset(function() { drawMap(ctx); });  // call once on init
//   drawMap(ctx);                                 // call each frame

// ─── Character home positions (px, canvas 320×320) ─────────────────────────
//
//   ┌──────────────────────────────┐
//   │  [conductor desk]            │  y ≈ 20–50
//   │  👑(160,58)  👩‍💼(108,58)     │
//   │  [carpet]                    │
//   │  [WS1]  [WS2]  [WS3]        │  y ≈ 80–100
//   │  🔍40   💻144  📊248         │  y = 106
//   │      [meeting area]          │  y ≈ 130–192
//   │  [WS4]  [WS5]               │  y ≈ 200–220
//   │  🔧40   🎯144                │  y = 222
//   │              🚪(160,308)     │  door
//   └──────────────────────────────┘

var MAP_POSITIONS = {
  door:      { x: 160, y: 314 },  // entry / exit point
  conductor: { x: 160, y:  58 },
  secretary: { x: 108, y:  58 },
  heartbeat: { x: 230, y:  58 },  // pigeon messenger — periodic heartbeat agent
  meeting:   { x: 160, y: 161 },  // neutral zone for conversations
  stations: [
    { x:  40, y: 106 },  // S0  scout
    { x: 144, y: 106 },  // S1  analyst
    { x: 248, y: 106 },  // S2  coder
    { x:  40, y: 222 },  // S3  worker
    { x: 144, y: 222 },  // S4  coordinator
  ],
};

// ─── Asset loading ──────────────────────────────────────────────────────────

var _mapImage = null;

// Call once before first draw. cb() is invoked when ready (image or fallback).
function loadMapAsset(cb) {
  var img = new Image();
  img.onload  = function() { _mapImage = img; cb(); };
  img.onerror = function() { cb(); };  // no map.png → use fallback
  img.src = './map.png';
}

// ─── Public draw entry point ────────────────────────────────────────────────

function drawMap(ctx) {
  ctx.imageSmoothingEnabled = false;
  if (_mapImage) {
    ctx.drawImage(_mapImage, 0, 0, 320, 320);
  } else {
    _drawMapFallback(ctx);
  }
}

// ─── Procedural fallback ────────────────────────────────────────────────────

var _C = {
  wallDark:      '#0c1018',
  wallHighlight: '#252d3f',
  floorA:        '#171b2c',
  floorB:        '#1b2033',
  carpetBase:    '#1a2050',
  carpetBorder:  '#2a3480',
  deskBack:      '#2c3e6b',
  deskTop:       '#3a50a0',
  deskEdge:      '#4a6ac0',
  deskShadow:    '#1a2448',
  monitorFrame:  '#070b14',
  monitorBlue:   '#1040a0',
  monitorGlow:   '#4488ff',
  wsBase:        '#162818',
  wsTop:         '#1e3822',
  wsEdge:        '#2a5030',
  termGlow:      '#00dd55',
  rugFill:       '#1c2248',
  rugBorder:     '#283070',
  doorMid:       '#8a5818',
  doorLight:     '#a06820',
  doorGold:      '#c8940a',
};

function _r(ctx, color, x, y, w, h, alpha) {
  ctx.globalAlpha = alpha === undefined ? 1 : alpha;
  ctx.fillStyle = color;
  ctx.fillRect(x, y, w, h);
  ctx.globalAlpha = 1;
}

function _b(ctx, color, x, y, w, h) {
  ctx.strokeStyle = color;
  ctx.lineWidth = 1;
  ctx.strokeRect(x + 0.5, y + 0.5, w - 1, h - 1);
}

function _dot(ctx, color, x, y) {
  ctx.fillStyle = color;
  ctx.fillRect(x, y, 2, 2);
}

function _workstation(ctx, x, y) {
  _r(ctx, _C.wsBase,  x,    y,    48, 20);
  _r(ctx, _C.wsTop,   x,    y,    48,  8);
  _r(ctx, _C.wsEdge,  x,    y,     2, 20);
  _r(ctx, _C.wsEdge,  x+46, y,     2, 20);
  _r(ctx, _C.wsEdge,  x,    y,    48,  2);
  // terminal screen
  _r(ctx, _C.monitorFrame, x+16, y+2,  16, 12);
  _r(ctx, '#041008',       x+17, y+3,  14, 10);
  _r(ctx, '#003315',       x+18, y+4,  12,  8);
  _r(ctx, _C.termGlow,     x+20, y+6,   8,  3);
  _dot(ctx, '#00ff88', x+22, y+6);
}

function _drawMapFallback(ctx) {
  var T = 16;

  // floor tiles
  for (var ty = 0; ty < 20; ty++) {
    for (var tx = 0; tx < 20; tx++) {
      ctx.fillStyle = (tx + ty) % 2 === 0 ? _C.floorA : _C.floorB;
      ctx.fillRect(tx * T, ty * T, T, T);
    }
  }

  // conductor carpet
  _r(ctx, _C.carpetBase,   16, 16, 288, 50);
  _b(ctx, _C.carpetBorder, 18, 18, 284, 46);

  // conductor desk
  _r(ctx, _C.deskBack,  96, 20, 128, 30);
  _r(ctx, _C.deskTop,   96, 20, 128, 12);
  _r(ctx, _C.deskEdge,  96, 20, 128,  2);
  _r(ctx, _C.deskEdge,  96, 20,   2, 30);
  _r(ctx, _C.deskEdge, 222, 20,   2, 30);
  _r(ctx, _C.deskShadow,96,48, 128,  4);
  // monitor
  _r(ctx, _C.monitorFrame, 138, 22, 44, 14);
  _r(ctx, _C.monitorBlue,  140, 23, 40, 12);
  _r(ctx, _C.monitorGlow,  156, 26,  8,  6);
  _r(ctx, '#6699ff',       158, 27,  4,  3);

  // workstations
  _workstation(ctx,  16,  80);  // S0
  _workstation(ctx, 128,  80);  // S1  (x+24 = 152 ≈ 144 center)
  _workstation(ctx, 224,  80);  // S2
  _workstation(ctx,  16, 200);  // S3
  _workstation(ctx, 128, 200);  // S4

  // meeting rug
  _r(ctx, _C.rugFill,   64, 130, 192, 62, 0.55);
  _b(ctx, _C.rugBorder, 66, 132, 188, 58);
  _b(ctx, '#202860',    70, 136, 180, 50);

  // bulletin board (left wall)
  _r(ctx, '#2c1a06', 18, 148, 36, 44);
  _r(ctx, '#3a2508', 20, 150, 32, 40);
  _r(ctx, '#cc9900', 22, 153, 12,  8);
  _r(ctx, '#dd8800', 22, 164, 10,  6);
  _r(ctx, '#bb7700', 34, 155, 13,  8);
  _r(ctx, '#ccaa00', 33, 165, 11,  6);
  _dot(ctx, '#ff4444', 28, 153);
  _dot(ctx, '#44aaff', 41, 158);
  _dot(ctx, '#44ff88', 27, 165);

  // server rack (right wall)
  _r(ctx, '#111122', 285,  80, 18, 112);
  _r(ctx, '#181830', 287,  82, 14, 108);
  for (var i = 0; i < 10; i++) {
    var ry = 85 + i * 10;
    _r(ctx, '#0a0a12', 288, ry, 12, 8);
    var lc = ['#00ff44','#0044ff','#ff3300','#111111'][i % 4];
    _r(ctx, lc, 296, ry + 2, 3, 4);
  }

  // walls (drawn last to cover any overruns)
  _r(ctx, _C.wallDark,      0,   0, 320,  16);
  _r(ctx, _C.wallHighlight, 0,  14, 320,   2);
  _r(ctx, _C.wallDark,      0,   0,  16, 320);
  _r(ctx, _C.wallHighlight,14,   0,   2, 320);
  _r(ctx, _C.wallDark,    304,   0,  16, 320);
  _r(ctx, _C.wallHighlight,304,  0,   2, 320);
  _r(ctx, _C.wallDark,      0, 304, 144,  16);
  _r(ctx, _C.wallDark,    176, 304, 144,  16);
  _r(ctx, _C.wallHighlight, 0, 304, 144,   2);
  _r(ctx, _C.wallHighlight,176, 304, 144,   2);

  // door
  _r(ctx, '#0a0808',    144, 292,  32, 12);        // outside (dark)
  _r(ctx, _C.doorMid,   144, 280,  32, 24);
  _r(ctx, _C.doorLight, 144, 280,  32,  3);
  _r(ctx, _C.doorLight, 144, 280,   3, 24);
  _r(ctx, _C.doorLight, 173, 280,   3, 24);
  _r(ctx, '#4a2408',    146, 284,  12, 16);        // door panels
  _r(ctx, '#4a2408',    162, 284,  12, 16);
  _r(ctx, _C.doorGold,  170, 291,   5,  5);        // handle
}

// Expose map helpers for app.js runtime.
globalThis.MAP_POSITIONS = MAP_POSITIONS;
globalThis.loadMapAsset = loadMapAsset;
globalThis.drawMap = drawMap;


