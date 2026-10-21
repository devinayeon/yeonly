/** @param {'localStorage' | 'sessionStorage'} area @param {string} key */
export function readStorage(area, key) {
  try {
    return globalThis[area]?.getItem(key) ?? null;
  } catch {
    // Storage can be disabled; the current in-memory session remains usable.
    return null;
  }
}

/** @param {'localStorage' | 'sessionStorage'} area @param {string} key @param {string} value */
export function writeStorage(area, key, value) {
  try {
    globalThis[area]?.setItem(key, value);
  } catch {
    // Persistence is best-effort when the browser blocks storage or its quota is full.
  }
}

/** @param {'localStorage' | 'sessionStorage'} area @param {string} key */
export function removeStorage(area, key) {
  try {
    globalThis[area]?.removeItem(key);
  } catch {
    // Storage can be disabled; the in-memory identity remains usable.
  }
}

/** Read only string values from a stored map; malformed storage is never trusted. */
export function readHostKeys() {
  /** @type {Record<string, string>} */
  const keys = Object.create(null);
  try {
    /** @type {unknown} */
    const value = JSON.parse(readStorage('localStorage', 'yeonly_host_keys') ?? '{}');
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      for (const [key, token] of Object.entries(value)) {
        if (typeof token === 'string') keys[key] = token;
      }
    }
  } catch {
    // Ignore malformed entries and allow the next successful write to repair them.
  }
  return keys;
}
