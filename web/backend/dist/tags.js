// web/backend/dist/tags.js
// Tag Cloud Controller (~3KB)
function loadTagCloud() {
  const cloud = document.getElementById('tag-cloud');
  if (!cloud) return;
  
  fetch('/api/vault/tags')
    .then(r => r.json())
    .then(tags => {
      cloud.innerHTML = '<h4>Tags</h4>';
      tags.forEach(tag => {
        const badge = document.createElement('span');
        badge.className = 'tag-badge';
        badge.textContent = `#${tag.name} (${tag.count})`;
        badge.onclick = () => filterByTag(tag.name);
        cloud.appendChild(badge);
      });
    })
    .catch(e => console.error('Failed to load tags:', e));
}

function filterByTag(tagName) {
  fetch(`/api/sessions/search?tag=${encodeURIComponent(tagName)}`)
    .then(r => r.json())
    .then(data => displaySearchResults(data))
    .catch(e => console.error('Failed to filter by tag:', e));
}

function displaySearchResults(data) {
  // TODO: render search results in UI
  console.log('Search results:', data);
}

// Expose
window.loadTagCloud = loadTagCloud;
window.filterByTag = filterByTag;
