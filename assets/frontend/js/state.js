/**
 * state.js — Minimal reactive state store.
 *
 * All mutations go through setState(). Subscribed listeners are called
 * synchronously after each update so the UI stays in sync.
 */

export const state = {
  /** Current view: 'inbox' | 'today' | 'upcoming' | 'all' | 'project:<id>' | 'label:<id>' */
  view:           'inbox',

  /** Loaded data */
  projects:       /** @type {import('./types').Project[]} */ ([]),
  labels:         /** @type {import('./types').Label[]}   */ ([]),
  tasks:          /** @type {import('./types').Task[]}    */ ([]),

  /** The task currently open in the detail panel (full object with subtasks) */
  selectedTask:   null,

  /** Whether to also show completed tasks in the list */
  showDone:       false,

  /** Live search query */
  search:         '',

  /** True while a network load is in progress */
  loading:        false,
};

const listeners = new Set();

/**
 * subscribe registers a callback that fires whenever state changes.
 * Returns an unsubscribe function.
 * @param {() => void} fn
 */
export function subscribe(fn) {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

/**
 * setState merges updates into state and notifies all listeners.
 * @param {Partial<typeof state>} updates
 */
export function setState(updates) {
  Object.assign(state, updates);
  listeners.forEach(fn => fn());
}
