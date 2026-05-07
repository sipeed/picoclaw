// web/backend/dist/sessions.js
// Session History Controller (~4KB)
function loadSessionHistory() {
  const history = document.getElementById('session-history');
  if (!history) return;
  
  fetch('/api/sessions/search?limit=20')
    .then(r => r.json())
    .then(sessions => {
      history.innerHTML = '<h4>Recent Sessions</h4>';
      sessions.forEach(session => {
        const div = document.createElement('div');
        div.className = 'session-item';
        div.innerHTML = `
          <div class="session-title">${session.title || 'Untitled'}</div>
          <div class="session-tags">${(session.tags || []).map(t => `#${t}`).join(' ')}</div>
          <div class="session-date">${new Date(session.timestamp).toLocaleDateString()}</div>
        `;
        div.onclick = () => loadSessionContext(session.id);
        history.appendChild(div);
      });
    })
    .catch(e => console.error('Failed to load sessions:', e));
}

function loadSessionContext(sessionId) {
  // TODO: load session context into chat
  console.log('Load session:', sessionId);
}

// Expose
window.loadSessionHistory = loadSessionHistory;
