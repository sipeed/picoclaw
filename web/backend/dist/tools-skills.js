// web/backend/dist/tools-skills.js
// Tool Skills Panel Controller (~3KB)
function loadToolSkills() {
  const panel = document.getElementById('tool-skills-panel');
  if (!panel) return;
  
  fetch('/api/tools/skills')
    .then(r => r.json())
    .then(skills => {
      panel.innerHTML = '<h4>Tool Skills</h4>';
      skills.forEach(skill => {
        const div = document.createElement('div');
        div.className = 'skill-item';
        div.innerHTML = `
          <div class="skill-name">${skill.name || 'Unknown'}</div>
          <div class="skill-usage">Used ${skill.usage_count || 0} times</div>
          <div class="skill-tags">${(skill.tags || []).map(t => `#${t}`).join(' ')}</div>
        `;
        panel.appendChild(div);
      });
    })
    .catch(e => console.error('Failed to load tool skills:', e));
}

async function searchCommunityRegistry(query) {
  try {
    const response = await fetch(`/api/tools/registry?q=${encodeURIComponent(query || '')}`);
    const results = await response.json();
    displayRegistryResults(results);
  } catch (e) {
    console.error('Failed to search registry:', e);
  }
}

function displayRegistryResults(results) {
  console.log('Registry results:', results);
  // TODO: render results in UI
}

// Expose
window.loadToolSkills = loadToolSkills;
window.searchCommunityRegistry = searchCommunityRegistry;
