/** Generate an opaque identifier, including on HTTP LAN origins without randomUUID. */
export function createId(prefix = '') {
  const random = globalThis.crypto.randomUUID?.()
    ?? Array.from(globalThis.crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, '0')).join('');
  return `${prefix}${random}`;
}
