// web/backend/dist/vault.js
// Vault Sidebar Controller (Vanilla JS, ~5KB)
document.addEventListener('DOMContentLoaded', () => {
  const sidebar = document.getElementById('vault-sidebar');
  const toggleBtn = document.getElementById('toggle-sidebar');
  const closeBtn = document.getElementById('close-sidebar');

  if (toggleBtn) {
    toggleBtn.addEventListener('click', () => {
      sidebar.classList.toggle('hidden');
      if (!sidebar.classList.contains('hidden')) {
        loadVaultTree();
      }
    });
  }

  if (closeBtn) {
    closeBtn.addEventListener('click', () => {
      sidebar.classList.add('hidden');
    });
  }

  async function loadVaultTree() {
    try {
      const response = await fetch('/api/vault/list');
      const data = await response.json();
      renderTree(data.notes || []);
    } catch (e) {
      console.error('Failed to load vault tree:', e);
    }
  }

  function renderTree(items) {
    const tree = document.getElementById('vault-tree');
    if (!tree) return;
    tree.innerHTML = '';
    items.forEach(item => {
      const div = document.createElement('div');
      div.className = 'tree-item';
      div.textContent = item.name || item.path || 'Untitled';
      div.onclick = () => loadNote(item.path || item.name);
      tree.appendChild(div);
    });
  }

  async function loadNote(path) {
    try {
      const response = await fetch(`/api/vault/note?path=${encodeURIComponent(path)}`);
      const data = await response.json();
      showNoteModal(data);
    } catch (e) {
      console.error('Failed to load note:', e);
    }
  }

  function showNoteModal(noteData) {
    // Simple modal implementation
    alert(`Note: ${noteData.title || 'Note'}\n\n${noteData.content || ''}`);
  }

  // Expose functions for other scripts
  window.loadVaultTree = loadVaultTree;
  window.loadNote = loadNote;
});
