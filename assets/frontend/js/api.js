/**
 * api.js — Fetch-based REST client for the Go todo backend.
 * All methods return a parsed JSON value or null (204).
 * On HTTP errors the returned promise rejects with an Error containing
 * the server's {"error":"…"} message.
 */

async function request(method, path, body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

/** Build a query string from an object, omitting null/undefined values. */
function qs(params) {
  const p = Object.fromEntries(
    Object.entries(params).filter(([, v]) => v != null && v !== '')
  );
  const s = new URLSearchParams(p).toString();
  return s ? '?' + s : '';
}

export const api = {
  // ── Projects ─────────────────────────────────────────────────────────────
  getProjects:    ()        => request('GET',    '/projects'),
  getProject:     (id)      => request('GET',    `/projects/${id}`),
  createProject:  (data)    => request('POST',   '/projects', data),
  updateProject:  (id, data)=> request('PATCH',  `/projects/${id}`, data),
  deleteProject:  (id)      => request('DELETE', `/projects/${id}`),

  // ── Labels ───────────────────────────────────────────────────────────────
  getLabels:      ()        => request('GET',    '/labels'),
  createLabel:    (data)    => request('POST',   '/labels', data),
  updateLabel:    (id, data)=> request('PATCH',  `/labels/${id}`, data),
  deleteLabel:    (id)      => request('DELETE', `/labels/${id}`),

  // ── Tasks ────────────────────────────────────────────────────────────────
  /**
   * getTasks({ project_id, due, completed }) — all params optional.
   * due: "today" | "upcoming"
   * completed: "true" | "false"
   */
  getTasks:    (params = {}) => request('GET', `/tasks${qs(params)}`),
  getTask:     (id)          => request('GET', `/tasks/${id}`),
  createTask:  (data)        => request('POST',   '/tasks', data),
  updateTask:  (id, data)    => request('PATCH',  `/tasks/${id}`, data),
  deleteTask:  (id)          => request('DELETE', `/tasks/${id}`),
  searchTasks: (q)           => request('GET', `/tasks/search?q=${encodeURIComponent(q)}`),
};
