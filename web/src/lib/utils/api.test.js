import test from 'node:test';
import assert from 'node:assert/strict';
import { fetchPublicRooms, createRoom, checkRoom } from './api.js';

test('API helpers validate payloads, preserve errors and propagate cancellation', async (t) => {
  let body = { rooms: null };
  let status = 200;
  const fetch = t.mock.method(globalThis, 'fetch', async () => new Response(JSON.stringify(body), { status }));
  const controller = new AbortController();
  assert.deepEqual(await fetchPublicRooms(controller.signal), []);
  assert.equal(fetch.mock.calls[0].arguments[1].signal, controller.signal);
  body = { rooms: [{ id: 'ROOM', name: 123 }] };
  await assert.rejects(fetchPublicRooms(), /tidak valid/);
  body = { roomId: 'ROOM', hostKey: 12 };
  await assert.rejects(createRoom('Class', false), /tidak valid/);
  body = { roomId: 'ROOM', hostKey: 'key' };
  assert.deepEqual(await createRoom('Class', false), body);
  const options = fetch.mock.calls.at(-1).arguments[1];
  assert.equal(options.method, 'POST');
  assert.deepEqual(JSON.parse(options.body), { name: 'Class', isPublic: false });
  status = 404;
  body = { error: 'Room expired' };
  await assert.rejects(checkRoom('ROOM'), /Room expired/);
});
