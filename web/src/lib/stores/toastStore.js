import { writable } from 'svelte/store';
import { createId } from '../utils/ids.js';

function createToastStore() {
  /** @typedef {{ id: string, message: string, type: 'info' | 'success' | 'error' | 'warning' }} Toast */
  /** @type {import('svelte/store').Writable<Toast[]>} */
  const state = writable([]);
  const { subscribe, update } = state;
  /** @type {Map<string, ReturnType<typeof setTimeout>>} */
  const timers = new Map();

  /** @param {string} message @param {Toast['type']} [type] @param {number} [duration] */
  function add(message, type = 'info', duration = 3500) {
    const id = createId();
    update(toasts => [...toasts, { id, message, type }]);

    timers.set(id, setTimeout(() => remove(id), duration));
  }

  /** @param {string} id */
  function remove(id) {
    clearTimeout(timers.get(id));
    timers.delete(id);
    update(toasts => toasts.filter(t => t.id !== id));
  }

  return {
    subscribe,
    /** @param {string} msg @param {number} [dur] */
    info: (msg, dur) => add(msg, 'info', dur),
    /** @param {string} msg @param {number} [dur] */
    success: (msg, dur) => add(msg, 'success', dur),
    /** @param {string} msg @param {number} [dur] */
    error: (msg, dur) => add(msg, 'error', dur),
    /** @param {string} msg @param {number} [dur] */
    warning: (msg, dur) => add(msg, 'warning', dur),
    remove,
  };
}

export const toast = createToastStore();
