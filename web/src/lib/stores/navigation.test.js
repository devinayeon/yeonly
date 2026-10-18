import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import { roomStore } from './roomStore.js';
import { ws } from './wsStore.js';
import { goHome, restoreLocation } from './navigation.js';

test('home navigation clears the room URL and reload/back/forward follow the URL', (t) => {
  const savedWindow = globalThis.window;
  let url = new URL('http://localhost/?room=ABC123');
  const history = [];
  globalThis.window = {
    get location() { return url; },
    history: {
      pushState(_state, _title, path) {
        history.push(path);
        url = new URL(path, url);
      },
    },
  };
  const connect = t.mock.method(ws, 'connect', () => {});
  const disconnect = t.mock.method(ws, 'disconnect', () => {});
  t.after(() => {
    if (savedWindow === undefined) delete globalThis.window;
    else globalThis.window = savedWindow;
    roomStore.reset();
  });

  roomStore.setRoomState({ id: 'ABC123', isHost: true, questions: [] });
  goHome();
  assert.equal(get(roomStore).currentView, 'landing');
  assert.equal(get(roomStore).roomId, '');
  assert.equal(url.search, '');
  assert.equal(disconnect.mock.callCount(), 1);
  assert.deepEqual(history, ['/']);

  // Refreshing the landing URL must not reconnect to the previous room.
  restoreLocation();
  assert.equal(connect.mock.callCount(), 0);
  assert.equal(get(roomStore).currentView, 'landing');
  goHome();
  assert.deepEqual(history, ['/'], 'home should not add duplicate history entries');

  // A room URL deliberately restores the room, including browser Back.
  url = new URL('http://localhost/?room=abc123');
  restoreLocation();
  assert.deepEqual(connect.mock.calls[0].arguments, ['ABC123']);
  roomStore.setRoomState({ id: 'ABC123', questions: [] });
  assert.equal(get(roomStore).currentView, 'room');

  // Forward to the landing URL disconnects and resets the room again.
  url = new URL('http://localhost/');
  restoreLocation();
  assert.equal(get(roomStore).currentView, 'landing');
  assert.equal(get(roomStore).roomId, '');
});
