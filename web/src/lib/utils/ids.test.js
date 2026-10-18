import test from 'node:test';
import assert from 'node:assert/strict';
import { createId } from './ids.js';

test('session IDs remain usable on HTTP origins without randomUUID', (t) => {
  const crypto = globalThis.crypto;
  const previous = Object.getOwnPropertyDescriptor(globalThis, 'crypto');
  Object.defineProperty(globalThis, 'crypto', { configurable: true, value: {
    getRandomValues: crypto.getRandomValues.bind(crypto),
  } });
  t.after(() => Object.defineProperty(globalThis, 'crypto', previous));
  const first = createId('sess_');
  assert.match(first, /^sess_[a-f0-9]{32}$/);
  assert.notEqual(createId('sess_'), first);
});
