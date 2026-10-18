import { cubicOut } from 'svelte/easing';

/** Small, reversible transitions shared by overlays and transient feedback.
 * @param {Element} _node
 * @param {{duration?: number, y?: number, scale?: number}} [options]
 */
export function reveal(_node, { duration = 180, y = 6, scale = 1 } = {}) {
  const reduced = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  return {
    duration: reduced ? 0 : duration,
    easing: cubicOut,
    css: (/** @type {number} */ t) => `opacity: ${t}; transform: translateY(${(1 - t) * y}px) scale(${scale + (1 - scale) * t});`,
  };
}
