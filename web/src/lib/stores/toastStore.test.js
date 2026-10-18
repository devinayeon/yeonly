import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import { toast } from './toastStore.js';

test('manual dismissal cancels expiry and does not remove other notifications', (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  toast.info('first', 100);
  toast.error('second', 200);
  const [first, second] = get(toast);
  toast.remove(first.id);
  t.mock.timers.tick(100);
  assert.deepEqual(get(toast), [second]);
  t.mock.timers.tick(100);
  assert.deepEqual(get(toast), []);
});
