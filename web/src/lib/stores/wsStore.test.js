import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import { ws, resolveWSOrigin } from './wsStore.js';
import { roomStore } from './roomStore.js';

const question = (overrides = {}) => ({
  id: 'q', roomId: 'ROOM', text: 'A valid question', authorId: 'author',
  authorAlias: 'Panda', authorRole: 'student', hasVoted: false,
  upvotes: 0, status: 'unanswered', createdAt: '2026-10-18T00:00:00Z', ...overrides,
});

function setupIdentitySockets(t) {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  const connections = [];
  class FakeSocket {
    static OPEN = 1;
    constructor(url) { this.url = new URL(url); this.readyState = 0; this.sent = []; connections.push(this); }
    send(raw) { this.sent.push(JSON.parse(raw)); }
    close() { this.readyState = 3; this.onclose?.(); }
    open() { this.readyState = 1; this.onopen(); }
    receive(type, payload) { this.onmessage({ data: JSON.stringify({ type, payload }) }); }
  }
  const saved = Object.fromEntries(['WebSocket', 'window', 'localStorage', 'sessionStorage'].map(key => [key, globalThis[key]]));
  const sessions = new Map();
  globalThis.WebSocket = FakeSocket;
  globalThis.window = { location: { protocol: 'http:', host: 'localhost:5173' } };
  globalThis.localStorage = { getItem: () => JSON.stringify({ ROOM: 'host-key' }), setItem() {} };
  globalThis.sessionStorage = {
    getItem: key => sessions.get(key),
    setItem: (key, value) => sessions.set(key, value),
    removeItem: key => sessions.delete(key),
  };
  roomStore.reset();
  t.after(() => {
    ws.disconnect();
    roomStore.reset();
    for (const [key, value] of Object.entries(saved)) {
      if (value === undefined) delete globalThis[key]; else globalThis[key] = value;
    }
  });
  return connections;
}

test('changing participant identity replaces the socket before the next vote', (t) => {
  const connections = setupIdentitySockets(t);
  ws.connect('ROOM', '', 'student');
  const first = connections[0];
  first.open();
  first.receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'participant-1', participantToken: 'token-1',
    questions: [question()], votedQuestionIds: [],
  });
  ws.upvoteQuestion('q');
  first.receive('QUESTION_UPDATED', question({ upvotes: 1 }));
  first.receive('VOTE_UPDATED', { questionId: 'q', hasVoted: true, upvotes: 1 });

  ws.rollNewIdentity();
  assert.equal(connections.length, 2, 'new identity must establish a new connection');
  const second = connections[1];
  assert.equal(first.readyState, 3);
  assert.equal(first.url.searchParams.has('session'), false);
  assert.equal(second.url.searchParams.has('session'), false, 'a new identity must not reuse the old token');
  assert.equal(second.url.searchParams.has('hostKey'), false);
  assert.equal(get(roomStore).participantToken, '');
  assert.equal(get(roomStore).questions[0].upvotes, 1);
  assert.deepEqual(get(roomStore).votedQuestionIds, []);
  assert.equal(ws.upvoteQuestion('q'), false, 'do not vote while the replacement socket is connecting');
  assert.deepEqual(get(roomStore).votedQuestionIds, []);

  first.receive('VOTE_UPDATED', { questionId: 'q', hasVoted: true, upvotes: 1 });
  assert.deepEqual(get(roomStore).votedQuestionIds, [], 'ignore events from the previous identity');
  second.open();
  second.receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'participant-2', participantToken: 'token-2',
    questions: [question({ upvotes: 1 })], votedQuestionIds: [],
  });
  assert.equal(ws.upvoteQuestion('q'), true);
  assert.equal(first.sent.length, 1, 'new vote must not use the previous identity');
  assert.deepEqual(second.sent, [{ action: 'UPVOTE_QUESTION', questionId: 'q' }]);
  second.receive('QUESTION_UPDATED', question({ upvotes: 2 }));
  second.receive('VOTE_UPDATED', { questionId: 'q', hasVoted: true, upvotes: 1 });
  assert.equal(get(roomStore).questions[0].upvotes, 2);
  assert.deepEqual(get(roomStore).votedQuestionIds, ['q']);
  ws.connect('ROOM', '', 'student');
  assert.equal(connections.length, 2, 'same identity should reuse its connection');
});

test('expired participant identity is discarded and reconnects without the stale token', (t) => {
  const connections = setupIdentitySockets(t);
  ws.connect('ROOM', '', 'student');
  const first = connections[0];
  first.open();
  first.receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'participant-1', participantToken: 'stale-token',
    questions: [], votedQuestionIds: [],
  });
  first.receive('IDENTITY_EXPIRED', 'identitas tersimpan tidak valid');

  assert.equal(get(roomStore).participantToken, '');
  first.close();
  t.mock.timers.tick(3000);

  assert.equal(connections.length, 2, 'expired identity must reconnect with a fresh identity');
  assert.equal(connections[1].url.searchParams.has('session'), false);
  connections[1].receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'participant-2', participantToken: 'fresh-token',
    questions: [], votedQuestionIds: [],
  });
  assert.equal(get(roomStore).participantToken, 'fresh-token');
});

test('a closed room stops the reconnect loop', (t) => {
  const connections = setupIdentitySockets(t);
  ws.connect('ROOM', '', 'student');
  const first = connections[0];
  first.open();
  first.receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'participant-1', participantToken: 'token-1',
    questions: [], votedQuestionIds: [],
  });
  first.receive('ROOM_CLOSED', 'ruang sudah ditutup');
  first.close();
  t.mock.timers.tick(10000);

  assert.equal(connections.length, 1);
  assert.equal(get(ws).reconnecting, false);
});

test('participant reconnect retains identity and role even with a saved host key', (t) => {
  const connections = setupIdentitySockets(t);
  ws.connect('ROOM', 'host-key');
  connections[0].open();
  connections[0].receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'host-id', participantToken: 'host-token',
    isHost: true, questions: [], votedQuestionIds: ['q'],
  });
  ws.switchRole('student');
  const student = connections[1];
  student.open();
  assert.equal(student.url.searchParams.has('session'), false);
  student.receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'student-id', participantToken: 'student-token',
    questions: [], votedQuestionIds: [],
  });
  student.close();
  t.mock.timers.tick(3000);
  const reconnected = connections[2];
  assert.equal(reconnected.url.searchParams.has('hostKey'), false);
  assert.equal(reconnected.url.searchParams.get('session'), 'student-token');
  reconnected.open();
  reconnected.receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'student-id', participantToken: 'student-token',
    questions: [], votedQuestionIds: [],
  });
  ws.switchRole('host');
  assert.equal(connections[3].url.searchParams.get('hostKey'), 'host-key');
  assert.deepEqual(get(roomStore).votedQuestionIds, []);
});

test('page reload restores the selected participant role instead of a stored host key', (t) => {
  const connections = setupIdentitySockets(t);
  ws.connect('ROOM', '', 'student');
  const student = connections[0];
  student.open();
  assert.equal(student.url.searchParams.has('session'), false);
  student.receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'student-id', participantToken: 'student-token',
    isHost: false, questions: [],
  });

  // Reload discards the stores and socket, but preserves browser storage.
  ws.disconnect();
  roomStore.reset();
  ws.connect('ROOM');
  const restored = connections[1];
  assert.equal(restored.url.searchParams.has('hostKey'), false);
  assert.equal(restored.url.searchParams.get('session'), 'student-token');
  restored.open();
  restored.receive('ROOM_STATE', {
    id: 'ROOM', participantId: 'student-id', participantToken: 'student-token',
    isHost: false, questions: [],
  });

  // An explicit host choice must still override the remembered participant role.
  ws.switchRole('host');
  const host = connections[2];
  assert.equal(host.url.searchParams.get('hostKey'), 'host-key');
  host.open();
  host.receive('ROOM_STATE', { id: 'ROOM', isHost: true, questions: [] });
  ws.disconnect();
  roomStore.reset();
  ws.connect('ROOM');
  assert.equal(connections[3].url.searchParams.get('hostKey'), 'host-key');
});

test('malformed events do not discard valid events in the same frame', (t) => {
  const connections = setupIdentitySockets(t);
  t.mock.method(console, 'error', () => {});
  ws.connect('ROOM', '', 'student');
  const socket = connections[0];
  socket.open();
  socket.onmessage({ data: '{broken\n' + JSON.stringify({ type: 'USER_COUNT', payload: { count: 2 } }) });
  assert.equal(get(roomStore).activeUsers, 2);
});

test('WebSocket presence, acknowledgements and reconnection lifecycle', async (t) => {
  t.mock.timers.enable({ apis: ['setTimeout', 'Date'], now: 10000 });
  const connections = [];
  class FakeSocket {
    static OPEN = 1;
    constructor() { this.readyState = 0; this.sent = []; connections.push(this); }
    send(raw) { this.sent.push(JSON.parse(raw)); }
    close() { this.readyState = 3; this.onclose?.(); }
    open() { this.readyState = 1; this.onopen(); }
    receive(type, payload, requestId) { this.onmessage({ data: JSON.stringify({ type, payload, requestId }) }); }
  }
  const saved = { WebSocket: globalThis.WebSocket, window: globalThis.window, localStorage: globalThis.localStorage };
  globalThis.WebSocket = FakeSocket;
  globalThis.window = { location: { protocol: 'http:', host: 'localhost:5173' } };
  globalThis.localStorage = { getItem: () => 'private-session', setItem: () => {} };
  t.after(() => {
    ws.disconnect();
    roomStore.reset();
    for (const [key, value] of Object.entries(saved)) {
      if (value === undefined) delete globalThis[key]; else globalThis[key] = value;
    }
  });

  ws.connect('ROOM');
  const first = connections[0];
  first.open();
  first.receive('ROOM_STATE', { id: 'ROOM', questions: [] });
  ws.setTyping(true);
  ws.setTyping(true);
  assert.deepEqual(first.sent, [{ action: 'TYPING', isTyping: true }]);
  t.mock.timers.tick(1500);
  ws.setTyping(true);
  assert.equal(first.sent.length, 2);
  ws.setTyping(false);
  assert.equal(first.sent[2].isTyping, false);

  first.receive('TYPING_UPDATED', { id: 'peer', alias: 'Panda', isTyping: true });
  assert.equal(get(roomStore).typingUsers.length, 1);
  t.mock.timers.tick(6001);
  assert.equal(get(roomStore).typingUsers.length, 0);

  const accepted = ws.submitQuestion('A valid question');
  const requestId = first.sent.at(-1).requestId;
  first.onmessage({ data: JSON.stringify({ type: 'QUESTION_ADDED', payload: question() }) + '\n' + JSON.stringify({ type: 'QUESTION_ACCEPTED', requestId }) });
  assert.equal(await accepted, true);
  assert.equal(get(roomStore).questions.length, 1);
  const rejected = ws.submitQuestion('Rejected draft');
  first.receive('ERROR', 'Room is frozen', first.sent.at(-1).requestId);
  assert.equal(await rejected, false);

  first.receive('TYPING_UPDATED', { id: 'peer', alias: 'Panda', isTyping: true });
  first.receive('ROOM_FROZEN', { isFrozen: true });
  assert.equal(get(roomStore).typingUsers.length, 0);
  const sentBeforeFrozenTyping = first.sent.length;
  ws.setTyping(true);
  assert.equal(first.sent.length, sentBeforeFrozenTyping);

  first.receive('ROOM_FROZEN', { isFrozen: false });
  const interrupted = ws.submitQuestion('Keep this draft');
  first.close();
  assert.equal(await interrupted, false);
  assert.equal(get(ws).reconnecting, true);
  t.mock.timers.tick(3000);
  const second = connections[1];
  second.open();
  first.receive('TYPING_UPDATED', { id: 'stale', alias: 'Old socket', isTyping: true });
  assert.equal(get(roomStore).typingUsers.length, 0);
  second.receive('ROOM_STATE', { id: 'ROOM', questions: [], typingUsers: [{ id: 'new', alias: 'Koala', isTyping: true }] });
  assert.equal(get(roomStore).typingUsers.length, 1);
  ws.disconnect();
  assert.equal(get(roomStore).typingUsers.length, 0);
  t.mock.timers.tick(10000);
  assert.equal(connections.length, 2);
});

test('resolveWSOrigin prefers the configured backend and mirrors the page origin otherwise', () => {
  assert.equal(resolveWSOrigin('wss://api.nayeonworks.tech', { protocol: 'https:', host: 'nayeonworks.tech' }), 'wss://api.nayeonworks.tech');
  assert.equal(resolveWSOrigin('wss://api.nayeonworks.tech/', { protocol: 'https:', host: 'nayeonworks.tech' }), 'wss://api.nayeonworks.tech');
  assert.equal(resolveWSOrigin('', { protocol: 'https:', host: 'nayeonworks.tech' }), 'wss://nayeonworks.tech');
  assert.equal(resolveWSOrigin('', { protocol: 'http:', host: 'localhost:5173' }), 'ws://localhost:5173');
});
