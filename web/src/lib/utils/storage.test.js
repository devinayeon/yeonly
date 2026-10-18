import test from 'node:test';
import assert from 'node:assert/strict';
import { readStorage, writeStorage, readHostKeys } from './storage.js';
import { getStoredHostKey, storeHostKey, removeStoredHostKey } from '../stores/roomStore.js';

test('invalid host storage is repaired without accepting non-string credentials', (t) => {
  let raw = '{broken';
  const previous = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: {
    getItem: () => raw, setItem: (_key, value) => { raw = value; },
  } });
  t.after(() => {
    if (previous) Object.defineProperty(globalThis, 'localStorage', previous);
    else Reflect.deleteProperty(globalThis, 'localStorage');
  });
  assert.equal(getStoredHostKey('ROOM'), '');
  storeHostKey('room', 'valid');
  assert.equal(getStoredHostKey('ROOM'), 'valid');
  raw = JSON.stringify({ ROOM: false, OTHER: 'keep' });
  assert.equal(getStoredHostKey('ROOM'), '');
  storeHostKey('room', 'repaired');
  assert.equal(getStoredHostKey('OTHER'), 'keep');
  removeStoredHostKey('room');
  assert.equal(getStoredHostKey('ROOM'), '');
  assert.equal(getStoredHostKey('OTHER'), 'keep');
  for (const invalid of ['null', '[]', '42', '"text"']) {
    raw = invalid;
    assert.equal(Object.keys(readHostKeys()).length, 0);
  }
});

test('storage access remains best-effort when the browser blocks it', (t) => {
  const previous = Object.getOwnPropertyDescriptor(globalThis, 'sessionStorage');
  Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, get() { throw new Error('Blocked'); } });
  t.after(() => {
    if (previous) Object.defineProperty(globalThis, 'sessionStorage', previous);
    else Reflect.deleteProperty(globalThis, 'sessionStorage');
  });
  assert.equal(readStorage('sessionStorage', 'key'), null);
  assert.doesNotThrow(() => writeStorage('sessionStorage', 'key', 'value'));
});
