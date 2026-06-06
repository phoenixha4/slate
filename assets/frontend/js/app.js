/**
 * app.js — Main UI controller for the Slate Todo app.
 *
 * Uses vanilla JS. The Go backend serves the embedded frontend and same-origin
 * REST API; this frontend calls it via api.js and renders everything client-side.
 */

import { api } from './api.js';
import { state, setState, subscribe } from './state.js';

// ────────────────────────────────────────────────────────────────────────────
// Colour palette for projects and labels
// ────────────────────────────────────────────────────────────────────────────
const COLORS = [
  '#7c6af7', '#ef4444', '#f97316', '#eab308', '#22c55e', '#3b82f6',
  '#6366f1', '#a855f7', '#ec4899', '#f43f5e', '#14b8a6', '#8b5cf6',
  '#64748b', '#84cc16', '#06b6d4', '#f59e0b',
];

let selectedColor = '#7c6af7';
let selectedProjectColor = '#7c6af7';

// ────────────────────────────────────────────────────────────────────────────
// Initialisation
// ────────────────────────────────────────────────────────────────────────────
async function init() {
  initChrome();
  // Load projects and labels
  const [projects, labels] = await Promise.all([api.getProjects(), api.getLabels()]);
  setState({ projects, labels, view: 'inbox' });
  await loadTasks();
  render();
  bindEvents();
  bindSearch();
  bindDialogs();
  bindInlineAdd();
  bindDetailPanel();
  bindColorSwatches();
  renderColorSwatches();
}

// ────────────────────────────────────────────────────────────────────────────
// Load data
// ────────────────────────────────────────────────────────────────────────────
async function loadTasks() {
  setState({ loading: true });
  try {
    let params = {};
    const v = state.view;
    if (v.startsWith('project:')) params.project_id = v.split(':')[1];
    if (v.startsWith('label:')) { /* label filtering done client-side after load */ }
    if (v === 'today') params.due = 'today';
    if (v === 'upcoming') params.due = 'upcoming';
    if (v === 'inbox') params.project_id = '00000000-0000-0000-0000-000000000001';
    if (!state.showDone) params.completed = 'false';
    const tasks = await api.getTasks(params);
    setState({ tasks, loading: false });
  } catch (e) {
    toast('Error loading tasks: ' + e.message);
    setState({ loading: false });
  }
}

async function loadTaskDetail(id) {
  try {
    const t = await api.getTask(id);
    setState({ selectedTask: t });
  } catch (e) {
    toast('Error loading task: ' + e.message);
  }
}

// ────────────────────────────────────────────────────────────────────────────
// Rendering
// ────────────────────────────────────────────────────────────────────────────
function render() {
  renderNav();
  renderProjects();
  renderLabels();
  renderTaskList();
  renderDetail();
  renderTitle();
  populateProjectSelects();
}

subscribe(render);

function renderNav() {
  document.querySelectorAll('[data-view]').forEach(el => {
    el.classList.toggle('active', el.dataset.view === state.view);
  });
}

function renderTitle() {
  const map = { inbox: 'Inbox', today: 'Today', upcoming: 'Upcoming', all: 'All Tasks' };
  let title = map[state.view] || 'Tasks';
  let sub = '';

  if (state.view.startsWith('project:')) {
    const p = state.projects.find(x => x.id === state.view.split(':')[1]);
    if (p) title = p.name;
  }
  if (state.view.startsWith('label:')) {
    const l = state.labels.find(x => x.id === state.view.split(':')[1]);
    if (l) title = 'Label: ' + l.name;
  }

  const today = new Date();
  const todayStr = today.toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'short' });
  if (state.view === 'today') sub = todayStr;
  if (state.view === 'upcoming') sub = 'Next 7 days';

  document.getElementById('view-title').textContent = title;
  document.getElementById('view-sub').textContent = sub;
}

function renderProjects() {
  const ul = document.getElementById('project-list');
  ul.innerHTML = state.projects.map(p => {
    const active = state.view === `project:${p.id}`;
    return `<li>
      <a href="#" class="nav-link ${active ? 'active' : ''}" data-view="project:${p.id}">
        <span class="dot" style="background:${p.color}"></span>
        ${esc(p.name)}
        <span class="badge ${p.task_count > 0 ? '' : 'hidden'}" data-variant="secondary">${p.task_count}</span>
      </a>
    </li>`;
  }).join('');

  // Rebind nav click handlers
  ul.querySelectorAll('[data-view]').forEach(el => {
    el.addEventListener('click', e => {
      e.preventDefault();
      setState({ view: el.dataset.view, selectedTask: null });
      closeDetail();
      if (window.matchMedia('(max-width: 900px)').matches) setSidebarOpen(false);
      loadTasks();
    });
  });

  // Update inbox/today badges
  const inbox = state.projects.find(p => p.id === '00000000-0000-0000-0000-000000000001');
  const inboxBadge = document.getElementById('badge-inbox');
  if (inbox && inboxBadge) {
    inboxBadge.textContent = inbox.task_count || '';
    inboxBadge.style.display = inbox.task_count > 0 ? 'inline' : 'none';
  }
}

function renderLabels() {
  const ul = document.getElementById('label-list');
  ul.innerHTML = state.labels.map(l => {
    const active = state.view === `label:${l.id}`;
    return `<li>
      <a href="#" class="nav-link ${active ? 'active' : ''}" data-view="label:${l.id}">
        <span class="badge" data-variant="secondary" style="background:${l.color};color:var(--primary-foreground)">${esc(l.name)}</span>
      </a>
    </li>`;
  }).join('');

  ul.querySelectorAll('[data-view]').forEach(el => {
    el.addEventListener('click', e => {
      e.preventDefault();
      setState({ view: el.dataset.view, selectedTask: null });
      closeDetail();
      if (window.matchMedia('(max-width: 900px)').matches) setSidebarOpen(false);
      loadTasks();
    });
  });
}

function renderTaskList() {
  const list = document.getElementById('task-list');
  let tasks = state.tasks;

  if (state.view.startsWith('label:')) {
    const lid = state.view.split(':')[1];
    tasks = tasks.filter(t => t.labels.some(l => l.id === lid));
  }

  if (state.search) {
    const q = state.search.toLowerCase();
    tasks = tasks.filter(t =>
      t.title.toLowerCase().includes(q) ||
      (t.notes || '').toLowerCase().includes(q)
    );
  }

  if (tasks.length === 0) {
    list.innerHTML = `<div class="empty-state">
      <span class="empty-state-icon"></span>
      <p>No tasks here. Add one when you are ready.</p>
    </div>`;
    return;
  }

  list.innerHTML = tasks.map(t => {
    const p = t.priority || 4;
    const done = t.completed ? 'done' : '';
    const sel = state.selectedTask?.id === t.id ? 'selected' : '';
    const labels = (t.labels || []).map(l =>
      `<span class="badge" style="background:${l.color};color:var(--primary-foreground)">${esc(l.name)}</span>`
    ).join('');
    const proj = state.projects.find(p => p.id === t.project_id);
    const projTag = proj ? `<small>${esc(proj.name)}</small>` : '';
    const due = formatDue(t.due_date);
    const dueClass = dueClassFor(t.due_date);
    const subCount = (t.subtasks || []).length;
    const subBadge = subCount > 0 ? `<span class="task-subtask-count">${subCount} sub-task${subCount > 1 ? 's' : ''}</span>` : '';
    return `
      <div class="task-card p${p} ${done} ${sel}" data-id="${t.id}">
        <button class="task-check-circle ${done}" data-check="${t.id}"></button>
        <div class="task-body">
          <div class="task-title-text">${esc(t.title)}</div>
          <div class="task-meta">
            ${projTag}
            ${labels}
            ${subBadge}
            <span class="task-due ${dueClass}">${due}</span>
          </div>
        </div>
      </div>
    `;
  }).join('');

  // Click handlers
  list.querySelectorAll('.task-card').forEach(el => {
    el.addEventListener('click', e => {
      if (e.target.closest('[data-check]')) return;
      const id = el.dataset.id;
      setState({ selectedTask: { id } }); // temporary placeholder
      loadTaskDetail(id);
    });
  });

  // Check circle handlers
  list.querySelectorAll('[data-check]').forEach(btn => {
    btn.addEventListener('click', async e => {
      e.stopPropagation();
      const id = btn.dataset.check;
      const t = state.tasks.find(x => x.id === id);
      if (!t) return;
      try {
        await api.updateTask(id, { completed: !t.completed });
        await loadTasks();
        await loadTaskDetail(id);
      } catch (err) {
        toast('Error: ' + err.message);
      }
    });
  });
}

function renderDetail() {
  const t = state.selectedTask;
  const dp = document.getElementById('detail-panel');
  const backdrop = document.getElementById('detail-backdrop');
  if (!t || !t.id) {
    dp.classList.add('dp-closed');
    backdrop?.classList.add('hidden');
    return;
  }
  dp.classList.remove('dp-closed');
  backdrop?.classList.remove('hidden');

  const proj = state.projects.find(p => p.id === t.project_id);
  document.getElementById('dp-project-badge').textContent = proj ? proj.name : 'Inbox';
  document.getElementById('dp-project-badge').style.display = 'inline';

  const titleEl = document.getElementById('dp-title');
  titleEl.textContent = t.title;
  titleEl.classList.toggle('done', t.completed);
  document.getElementById('dp-check').classList.toggle('done', t.completed);

  document.getElementById('dp-notes').value = t.notes || '';
  showNotesPreview();
  document.getElementById('dp-due').value = t.due_date || '';
  document.getElementById('dp-project-sel').value = t.project_id || '';

  // Priority
  const pp = document.getElementById('dp-priority');
  pp.querySelectorAll('.p-btn').forEach(btn => {
    btn.classList.toggle('sel', Number(btn.dataset.p) === (t.priority || 4));
  });

  // Labels
  const lblWrap = document.getElementById('dp-labels');
  lblWrap.innerHTML = state.labels.map(l => {
    const on = (t.labels || []).some(x => x.id === l.id);
    return `<button class="label-toggle ${on ? 'on' : ''}" data-lid="${l.id}" style="background:${on ? l.color : ''}">${esc(l.name)}</button>`;
  }).join('');
  lblWrap.querySelectorAll('.label-toggle').forEach(btn => {
    btn.addEventListener('click', async () => {
      const lid = btn.dataset.lid;
      const on = btn.classList.contains('on');
      const currentIds = (state.selectedTask.labels || []).map(l => l.id);
      const newIds = on ? currentIds.filter(id => id !== lid) : [...currentIds, lid];
      try {
        await api.updateTask(t.id, { label_ids: newIds });
        await loadTaskDetail(t.id);
        await loadTasks();
      } catch (e) {
        toast('Error: ' + e.message);
      }
    });
  });

  // Subtasks
  const subList = document.getElementById('dp-subtask-list');
  const subs = t.subtasks || [];
  if (subs.length === 0) subList.innerHTML = '<p class="text-muted subtask-empty">No sub-tasks yet.</p>';
  else subList.innerHTML = subs.map(s => `
    <div class="subtask-row ${s.completed ? 'done' : ''}" data-subid="${s.id}">
      <button class="task-check-circle ${s.completed ? 'done' : ''}"></button>
      <span class="subtask-title">${esc(s.title)}</span>
    </div>
  `).join('');

  subList.querySelectorAll('.subtask-row').forEach(el => {
    el.addEventListener('click', async () => {
      const sid = el.dataset.subid;
      const s = subs.find(x => x.id === sid);
      if (!s) return;
      try {
        await api.updateTask(sid, { completed: !s.completed });
        await loadTaskDetail(t.id);
        await loadTasks();
      } catch (e) {
        toast('Error: ' + e.message);
      }
    });
  });

  // Meta
  document.getElementById('dp-meta').textContent = `Created ${new Date(t.created_at).toLocaleDateString()}`;
}

function populateProjectSelects() {
  const options = state.projects.map(p => `<option value="${p.id}">${esc(p.name)}</option>`).join('');
  ['inline-project', 'dp-project-sel', 'qa-project'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.innerHTML = options;
  });
}

// ────────────────────────────────────────────────────────────────────────────
// Event binding
// ────────────────────────────────────────────────────────────────────────────
function bindEvents() {
  document.querySelectorAll('[data-sidebar-toggle]').forEach(btn => {
    btn.addEventListener('click', toggleSidebar);
  });
  document.getElementById('sidebar-backdrop')?.addEventListener('click', () => setSidebarOpen(false));

  // Sidebar navigation
  document.querySelectorAll('.nav-link[data-view]').forEach(el => {
    el.addEventListener('click', e => {
      e.preventDefault();
      const v = el.dataset.view;
      if (v.startsWith('project:') || v.startsWith('label:')) return; // handled by render
      setState({ view: v, selectedTask: null });
      closeDetail();
      if (window.matchMedia('(max-width: 900px)').matches) setSidebarOpen(false);
      loadTasks();
    });
  });

  // Toggle completed
  document.getElementById('btn-toggle-done').addEventListener('click', toggleCompleted);

  // Add task button (top bar)
  document.getElementById('btn-add-task').addEventListener('click', () => {
    openTaskDialog();
  });

  // Show inline add
  document.getElementById('btn-show-add').addEventListener('click', () => {
    document.getElementById('inline-add').classList.remove('hidden');
    document.getElementById('btn-show-add').classList.add('hidden');
    document.getElementById('inline-title').focus();
  });
}

function initChrome() {
  const savedTheme = localStorage.getItem('theme');
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  applyTheme(savedTheme || (prefersDark ? 'dark' : 'light'));

  const desktop = window.matchMedia('(min-width: 901px)').matches;
  const sidebarPinned = localStorage.getItem('sidebar_pinned');
  const sidebarOpen = localStorage.getItem('sidebar_open');
  const pinned = sidebarPinned === null ? desktop : sidebarPinned === 'true';
  setSidebarPinned(pinned, { persist: false });
  setSidebarOpen(pinned ? true : sidebarOpen === 'true', { persist: false });

  document.getElementById('btn-theme')?.addEventListener('click', toggleTheme);
  document.getElementById('sidebar-theme')?.addEventListener('click', toggleTheme);
  document.querySelector('[data-sidebar-pin]')?.addEventListener('click', toggleSidebarPinned);
  document.getElementById('btn-shortcuts')?.addEventListener('click', openShortcuts);
  document.getElementById('sidebar-shortcuts')?.addEventListener('click', openShortcuts);
  document.getElementById('sidebar-quick-add')?.addEventListener('click', openTaskDialog);
  document.getElementById('shortcuts-close')?.addEventListener('click', () => {
    document.getElementById('dlg-shortcuts')?.close();
  });
  document.getElementById('detail-backdrop')?.addEventListener('click', closeDetail);

  ['dlg-add-task', 'dlg-add-project', 'dlg-add-label', 'dlg-shortcuts'].forEach(id => {
    const dlg = document.getElementById(id);
    dlg?.addEventListener('keydown', e => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        e.preventDefault();
        dlg.querySelector('footer button:last-child')?.click();
      }
      if (e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey && e.target?.tagName !== 'TEXTAREA') {
        e.preventDefault();
        dlg.querySelector('footer button:last-child')?.click();
      }
    });
  });

  document.addEventListener('keydown', handleShortcuts);
}

function applyTheme(theme) {
  document.body.dataset.theme = theme;
  localStorage.setItem('theme', theme);
  const btn = document.getElementById('btn-theme');
  if (btn) {
    btn.textContent = theme === 'dark' ? '☼' : '◐';
    btn.title = theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode';
  }
  const sideBtn = document.getElementById('sidebar-theme');
  if (sideBtn) {
    sideBtn.innerHTML = `${theme === 'dark' ? 'Light' : 'Dark'} <kbd>D</kbd>`;
  }
}

function toggleTheme() {
  applyTheme(document.body.dataset.theme === 'dark' ? 'light' : 'dark');
}

function setSidebarOpen(open, opts = { persist: true }) {
  document.body.classList.toggle('sidebar-collapsed', !open);
  const showBackdrop = open && window.matchMedia('(max-width: 900px)').matches;
  document.getElementById('sidebar-backdrop')?.classList.toggle('hidden', !showBackdrop);
  document.querySelectorAll('[data-sidebar-toggle]').forEach(btn => {
    btn.setAttribute('aria-expanded', String(open));
  });
  if (opts.persist) localStorage.setItem('sidebar_open', String(open));
}

function toggleSidebar() {
  const open = document.body.classList.contains('sidebar-collapsed');
  if (!open) setSidebarPinned(false);
  setSidebarOpen(open);
}

function setSidebarPinned(pinned, opts = { persist: true }) {
  document.body.classList.toggle('sidebar-pinned', pinned);
  const btn = document.querySelector('[data-sidebar-pin]');
  if (btn) {
    btn.setAttribute('aria-pressed', String(pinned));
    btn.title = pinned ? 'Unpin sidebar' : 'Pin sidebar';
    btn.setAttribute('aria-label', pinned ? 'Unpin sidebar' : 'Pin sidebar');
    btn.textContent = pinned ? '⌘' : '⌖';
  }
  if (opts.persist) localStorage.setItem('sidebar_pinned', String(pinned));
}

function toggleSidebarPinned() {
  const pinned = !document.body.classList.contains('sidebar-pinned');
  setSidebarPinned(pinned);
  setSidebarOpen(pinned);
}

function closeDetail() {
  document.getElementById('detail-panel')?.classList.add('dp-closed');
  document.getElementById('detail-backdrop')?.classList.add('hidden');
  setState({ selectedTask: null });
}

function openTaskDialog() {
  populateQaLabels();
  const dlg = document.getElementById('dlg-add-task');
  if (dlg.open) return;
  dlg.showModal();
  requestAnimationFrame(() => document.getElementById('qa-title')?.focus());
}

function openShortcuts() {
  document.getElementById('dlg-shortcuts')?.showModal();
}

let pendingGoto = '';
let pendingGotoTimer;

function handleShortcuts(e) {
  const target = e.target;
  const isTyping = target && (
    target.matches?.('input, textarea, select') ||
    target.isContentEditable
  );

  if (e.key === 'Escape') {
    const openDialog = document.querySelector('dialog[open]');
    if (openDialog) {
      e.preventDefault();
      openDialog.close();
      return;
    }
    if (!document.getElementById('inline-add')?.classList.contains('hidden')) {
      e.preventDefault();
      hideInlineAdd();
      return;
    }
    if (!document.getElementById('dp-subtask-add')?.classList.contains('hidden')) {
      e.preventDefault();
      hideSubtaskAdd();
      return;
    }
    closeDetail();
    setSidebarOpen(false);
    return;
  }

  if (isTyping) return;

  if (e.key === '?') {
    e.preventDefault();
    openShortcuts();
    return;
  }
  if (e.key === '/') {
    e.preventDefault();
    document.getElementById('search-input')?.focus();
    return;
  }
  if (e.key.toLowerCase() === 'n') {
    e.preventDefault();
    openTaskDialog();
    return;
  }
  if (e.key.toLowerCase() === 's') {
    e.preventDefault();
    toggleSidebar();
    return;
  }
  if (e.key.toLowerCase() === 'p') {
    e.preventDefault();
    toggleSidebarPinned();
    return;
  }
  if (e.key.toLowerCase() === 'd') {
    e.preventDefault();
    toggleTheme();
    return;
  }
  if (e.shiftKey && e.key.toLowerCase() === 'c') {
    e.preventDefault();
    toggleCompleted();
    return;
  }

  const k = e.key.toLowerCase();
  if (k === 'g' || pendingGoto) {
    e.preventDefault();
    pendingGoto += k;
    clearTimeout(pendingGotoTimer);
    pendingGotoTimer = setTimeout(() => { pendingGoto = ''; }, 900);
    const map = { gi: 'inbox', gt: 'today', gu: 'upcoming', ga: 'all' };
    if (map[pendingGoto]) {
      setState({ view: map[pendingGoto], selectedTask: null });
      closeDetail();
      loadTasks();
      pendingGoto = '';
    }
  }
}

function bindSearch() {
  const inp = document.getElementById('search-input');
  let timer;
  inp.addEventListener('input', () => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      const q = inp.value.trim();
      if (q) {
        api.searchTasks(q).then(tasks => setState({ tasks, search: q }));
      } else {
        setState({ search: '' });
        loadTasks();
      }
    }, 300);
  });
}

// ────────────────────────────────────────────────────────────────────────────
// Dialogs
// ────────────────────────────────────────────────────────────────────────────
function bindDialogs() {
  // Quick add task
  document.getElementById('btn-add-task').addEventListener('click', () => {
    openTaskDialog();
  });
  document.getElementById('qa-cancel').addEventListener('click', () => {
    document.getElementById('dlg-add-task').close();
  });
  document.getElementById('qa-save').addEventListener('click', () => createTaskFromDialog());
  bindPriorityPicker('qa-priority');

  // New project
  document.getElementById('btn-new-project').addEventListener('click', () => {
    document.getElementById('dlg-add-project').showModal();
    selectedProjectColor = '#7c6af7';
    renderProjectSwatches();
  });
  document.getElementById('proj-cancel').addEventListener('click', () => {
    document.getElementById('dlg-add-project').close();
  });
  document.getElementById('proj-save').addEventListener('click', async () => {
    const name = document.getElementById('proj-name').value.trim();
    if (!name) { toast('Project name is required'); return; }
    try {
      await api.createProject({ name, color: selectedProjectColor });
      const projects = await api.getProjects();
      setState({ projects });
      document.getElementById('dlg-add-project').close();
      document.getElementById('proj-name').value = '';
      toast('Project created');
    } catch (e) {
      toast('Error: ' + e.message);
    }
  });

  // New label
  document.getElementById('btn-new-label').addEventListener('click', () => {
    document.getElementById('dlg-add-label').showModal();
    selectedColor = '#888899';
    renderLabelSwatches();
  });
  document.getElementById('lbl-cancel').addEventListener('click', () => {
    document.getElementById('dlg-add-label').close();
  });
  document.getElementById('lbl-save').addEventListener('click', async () => {
    const name = document.getElementById('lbl-name').value.trim();
    if (!name) { toast('Label name is required'); return; }
    try {
      await api.createLabel({ name, color: selectedColor });
      const labels = await api.getLabels();
      setState({ labels });
      document.getElementById('dlg-add-label').close();
      document.getElementById('lbl-name').value = '';
      toast('Label created');
    } catch (e) {
      toast('Error: ' + e.message);
    }
  });

  // Close dialogs on backdrop click
  ['dlg-add-task', 'dlg-add-project', 'dlg-add-label'].forEach(id => {
    const dlg = document.getElementById(id);
    dlg.addEventListener('click', e => {
      if (e.target === dlg) dlg.close();
    });
  });
}

function bindColorSwatches() {
  // Render label swatches
  const lblWrap = document.getElementById('lbl-colors');
  lblWrap.innerHTML = COLORS.map(c =>
    `<span class="color-swatch ${c === selectedColor ? 'selected' : ''}" style="background:${c}" data-color="${c}"></span>`
  ).join('');
  lblWrap.querySelectorAll('.color-swatch').forEach(el => {
    el.addEventListener('click', () => {
      selectedColor = el.dataset.color;
      renderLabelSwatches();
    });
  });
}

function renderLabelSwatches() {
  const el = document.getElementById('lbl-colors');
  if (!el) return;
  el.querySelectorAll('.color-swatch').forEach(sw => {
    sw.classList.toggle('selected', sw.dataset.color === selectedColor);
  });
}

function renderProjectSwatches() {
  const el = document.getElementById('proj-colors');
  if (!el) return;
  el.querySelectorAll('.color-swatch').forEach(sw => {
    sw.classList.toggle('selected', sw.dataset.color === selectedProjectColor);
  });
}

function renderColorSwatches() {
  const projWrap = document.getElementById('proj-colors');
  const lblWrap = document.getElementById('lbl-colors');
  const html = COLORS.map(c =>
    `<span class="color-swatch" style="background:${c}" data-color="${c}"></span>`
  ).join('');
  if (projWrap) projWrap.innerHTML = html;
  if (lblWrap) lblWrap.innerHTML = html;

  projWrap?.querySelectorAll('.color-swatch').forEach(el => {
    el.addEventListener('click', () => {
      selectedProjectColor = el.dataset.color;
      renderProjectSwatches();
    });
  });
  lblWrap?.querySelectorAll('.color-swatch').forEach(el => {
    el.addEventListener('click', () => {
      selectedColor = el.dataset.color;
      renderLabelSwatches();
    });
  });
}

function populateQaLabels() {
  const wrap = document.getElementById('qa-labels');
  wrap.innerHTML = state.labels.map(l =>
    `<button class="label-toggle" data-lid="${l.id}" style="background:${l.color}">${esc(l.name)}</button>`
  ).join('');
  wrap.querySelectorAll('.label-toggle').forEach(btn => {
    btn.addEventListener('click', () => btn.classList.toggle('on'));
  });
}

function createTaskFromDialog() {
  const title = document.getElementById('qa-title').value.trim();
  if (!title) { toast('Task name is required'); return; }
  const pp = document.getElementById('qa-priority');
  const p = Number(pp.querySelector('.sel')?.dataset.p || 4);
  const pid = document.getElementById('qa-project').value || undefined;
  const due = document.getElementById('qa-due').value || undefined;
  const notes = document.getElementById('qa-notes').value || undefined;
  const labelIds = Array.from(document.querySelectorAll('#qa-labels .on')).map(b => b.dataset.lid);

  api.createTask({
    title, notes, priority: p,
    project_id: pid || undefined,
    due_date: due || undefined,
    label_ids: labelIds.length ? labelIds : undefined,
  }).then(() => {
    document.getElementById('dlg-add-task').close();
    document.getElementById('qa-title').value = '';
    document.getElementById('qa-notes').value = '';
    document.getElementById('qa-due').value = '';
    toast('Task created');
    loadTasks();
  }).catch(e => toast('Error: ' + e.message));
}

// ────────────────────────────────────────────────────────────────────────────
// Inline add
// ─────────────────────────────────────────────────────────────────────────────
function bindInlineAdd() {
  bindPriorityPicker('inline-priority');

  document.getElementById('btn-inline-cancel').addEventListener('click', hideInlineAdd);

  document.getElementById('inline-title').addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      e.preventDefault();
      saveInlineTask();
    }
    if (e.key === 'Escape') {
      e.preventDefault();
      hideInlineAdd();
    }
  });
  document.getElementById('inline-notes').addEventListener('keydown', e => {
    if (e.key === 'Escape') {
      e.preventDefault();
      hideInlineAdd();
    }
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault();
      saveInlineTask();
    }
  });

  document.getElementById('btn-inline-save').addEventListener('click', saveInlineTask);
}

function hideInlineAdd() {
  document.getElementById('inline-add').classList.add('hidden');
  document.getElementById('btn-show-add').classList.remove('hidden');
}

async function saveInlineTask() {
  const title = document.getElementById('inline-title').value.trim();
  if (!title) return;
  const pp = document.getElementById('inline-priority');
  const p = Number(pp.querySelector('.sel')?.dataset.p || 4);
  const pid = document.getElementById('inline-project').value || undefined;
  const due = document.getElementById('inline-due').value || undefined;
  const notes = document.getElementById('inline-notes').value || undefined;

  try {
    await api.createTask({
      title, notes, priority: p,
      project_id: pid || undefined,
      due_date: due || undefined,
    });
    document.getElementById('inline-title').value = '';
    document.getElementById('inline-notes').value = '';
    document.getElementById('inline-due').value = '';
    hideInlineAdd();
    toast('Task created');
    await loadTasks();
  } catch (e) {
    toast('Error: ' + e.message);
  }
}

function bindPriorityPicker(containerId) {
  const el = document.getElementById(containerId);
  if (!el) return;
  el.querySelectorAll('.p-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      el.querySelectorAll('.p-btn').forEach(b => b.classList.remove('sel', 'active-p4'));
      btn.classList.add('sel');
    });
  });
}

// ─────────────────────────────────────────────────────────────────────────────
// Detail panel
// ────────────────────────────────────────────────────────────────────────────
function bindDetailPanel() {
  document.getElementById('btn-dp-close').addEventListener('click', () => {
    closeDetail();
  });

  // Toggle complete
  document.getElementById('dp-check').addEventListener('click', async () => {
    const t = state.selectedTask;
    if (!t) return;
    try {
      await api.updateTask(t.id, { completed: !t.completed });
      await loadTaskDetail(t.id);
      await loadTasks();
    } catch (e) {
      toast('Error: ' + e.message);
    }
  });

  // Title edit
  document.getElementById('dp-title').addEventListener('blur', async () => {
    const t = state.selectedTask;
    if (!t) return;
    const val = document.getElementById('dp-title').textContent.trim();
    if (val && val !== t.title) {
      try {
        await api.updateTask(t.id, { title: val });
        await loadTaskDetail(t.id);
        await loadTasks();
      } catch (e) {
        toast('Error: ' + e.message);
      }
    }
  });
  document.getElementById('dp-title').addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); e.target.blur(); }
  });

  // Notes
  document.getElementById('dp-notes-preview').addEventListener('click', showNotesEditor);
  document.getElementById('dp-notes-preview').addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      e.preventDefault();
      showNotesEditor();
    }
  });
  document.getElementById('dp-notes').addEventListener('keydown', e => {
    if (e.key === 'Escape') {
      e.preventDefault();
      showNotesPreview();
    }
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault();
      saveNotes();
    }
  });
  document.getElementById('dp-notes').addEventListener('change', saveNotes);

  async function saveNotes() {
    const t = state.selectedTask;
    if (!t) return;
    const val = document.getElementById('dp-notes').value;
    if (val !== (t.notes || '')) {
      try {
        await api.updateTask(t.id, { notes: val });
        await loadTaskDetail(t.id);
        showNotesPreview();
      } catch (e) {
        toast('Error: ' + e.message);
      }
    } else {
      showNotesPreview();
    }
  }

  // Due date
  document.getElementById('dp-due').addEventListener('change', async () => {
    const t = state.selectedTask;
    if (!t) return;
    const val = document.getElementById('dp-due').value;
    try {
      await api.updateTask(t.id, { due_date: val || '' });
      await loadTaskDetail(t.id);
      await loadTasks();
    } catch (e) {
      toast('Error: ' + e.message);
    }
  });

  // Priority
  document.getElementById('dp-priority').querySelectorAll('.p-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const t = state.selectedTask;
      if (!t) return;
      const p = Number(btn.dataset.p);
      try {
        await api.updateTask(t.id, { priority: p });
        await loadTaskDetail(t.id);
        await loadTasks();
      } catch (e) {
        toast('Error: ' + e.message);
      }
    });
  });

  // Project change
  document.getElementById('dp-project-sel').addEventListener('change', async () => {
    const t = state.selectedTask;
    if (!t) return;
    const pid = document.getElementById('dp-project-sel').value;
    try {
      await api.updateTask(t.id, { project_id: pid || '' });
      await loadTaskDetail(t.id);
      await loadTasks();
    } catch (e) {
      toast('Error: ' + e.message);
    }
  });

  // Delete task
  document.getElementById('btn-dp-delete').addEventListener('click', async () => {
    const t = state.selectedTask;
    if (!t) return;
    if (!confirm('Delete this task?')) return;
    try {
      await api.deleteTask(t.id);
      closeDetail();
      await loadTasks();
      toast('Task deleted');
    } catch (e) {
      toast('Error: ' + e.message);
    }
  });

  // Add sub-task
  document.getElementById('btn-dp-add-sub').addEventListener('click', showSubtaskAdd);
  document.getElementById('btn-dp-subtask-save').addEventListener('click', saveSubtask);
  document.getElementById('btn-dp-subtask-cancel').addEventListener('click', hideSubtaskAdd);
  document.getElementById('dp-subtask-title').addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      e.preventDefault();
      saveSubtask();
    }
    if (e.key === 'Escape') {
      e.preventDefault();
      hideSubtaskAdd();
    }
  });
}

function toggleCompleted() {
  setState({ showDone: !state.showDone });
  document.getElementById('btn-toggle-done').classList.toggle('active', state.showDone);
  loadTasks();
}

function showNotesEditor() {
  document.getElementById('dp-notes-preview').classList.add('hidden');
  const notes = document.getElementById('dp-notes');
  notes.classList.remove('hidden');
  notes.focus();
}

function showNotesPreview() {
  const notes = document.getElementById('dp-notes');
  const preview = document.getElementById('dp-notes-preview');
  const raw = notes.value.trim();
  preview.innerHTML = raw ? renderMarkdown(raw) : '<span class="markdown-empty">Add notes...</span>';
  preview.classList.toggle('is-empty', !raw);
  notes.classList.add('hidden');
  preview.classList.remove('hidden');
}

function showSubtaskAdd() {
  document.getElementById('dp-subtask-add').classList.remove('hidden');
  document.getElementById('dp-subtask-title').focus();
}

function hideSubtaskAdd() {
  document.getElementById('dp-subtask-title').value = '';
  document.getElementById('dp-subtask-add').classList.add('hidden');
}

async function saveSubtask() {
  const t = state.selectedTask;
  if (!t) return;
  const title = document.getElementById('dp-subtask-title').value.trim();
  if (!title) return;
  try {
    await api.createTask({ title, parent_id: t.id, priority: 4 });
    hideSubtaskAdd();
    await loadTaskDetail(t.id);
    await loadTasks();
    toast('Sub-task added');
  } catch (e) {
    toast('Error: ' + e.message);
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────
function esc(s) {
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

function renderMarkdown(raw) {
  const lines = esc(raw).split(/\n{2,}/).map(block => {
    const trimmed = block.trim();
    if (!trimmed) return '';
    const listLines = trimmed.split('\n').filter(line => /^[-*]\s+/.test(line.trim()));
    if (listLines.length > 0 && listLines.length === trimmed.split('\n').length) {
      return `<ul>${listLines.map(line => `<li>${inlineMarkdown(line.trim().replace(/^[-*]\s+/, ''))}</li>`).join('')}</ul>`;
    }
    return `<p>${inlineMarkdown(trimmed).replace(/\n/g, '<br>')}</p>`;
  });
  return lines.join('');
}

function inlineMarkdown(s) {
  return s
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\*([^*]+)\*/g, '<em>$1</em>')
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
}

function formatDue(d) {
  if (!d) return '';
  const today = new Date(); today.setHours(0,0,0,0);
  const due = new Date(d); due.setHours(0,0,0,0);
  const diff = (due - today) / (1000*60*60*24);
  if (diff === 0) return 'Today';
  if (diff === 1) return 'Tomorrow';
  if (diff === -1) return 'Yesterday';
  return due.toLocaleDateString('en-GB', { day:'numeric', month:'short' });
}

function dueClassFor(d) {
  if (!d) return '';
  const today = new Date(); today.setHours(0,0,0,0);
  const due = new Date(d); due.setHours(0,0,0,0);
  const diff = (due - today) / (1000*60*60*24);
  if (diff === 0) return 'today';
  if (diff < 0) return 'overdue';
  return '';
}

function toast(msg) {
  let stack = document.querySelector('.toast-stack');
  if (!stack) {
    stack = document.createElement('div');
    stack.className = 'toast-stack';
    document.body.appendChild(stack);
  }

  const el = document.createElement('div');
  el.className = `toast ${/^error[:\s]/i.test(msg) ? 'error' : ''}`;
  el.textContent = msg;
  stack.appendChild(el);

  window.setTimeout(() => {
    el.classList.add('leaving');
    el.addEventListener('animationend', () => el.remove(), { once: true });
  }, 2600);
}

// ─────────────────────────────────────────────────────────────────────────────
init();
