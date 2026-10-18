import test from 'node:test';
import assert from 'node:assert/strict';
import { copyText } from './clipboard.js';

test('clipboard failure is reported instead of showing false success', async (t) => {
  const previous = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
  let copied;
  const clipboard = { writeText: async text => { copied = text; } };
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { clipboard } });
  t.after(() => {
    if (previous) Object.defineProperty(globalThis, 'navigator', previous);
    else Reflect.deleteProperty(globalThis, 'navigator');
  });
  assert.equal(await copyText('ROOM'), true);
  assert.equal(copied, 'ROOM');
  clipboard.writeText = async () => { throw new Error('Permission denied'); };
  assert.equal(await copyText('ROOM'), false);
});
