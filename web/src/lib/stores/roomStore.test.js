import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import { roomStore, filteredQuestions } from './roomStore.js';

test('only live arrivals animate once; history, votes and filters do not replay', () => {
  roomStore.reset();
  roomStore.setRoomState({ id: 'ROOM', questions: [{ id: 'old', text: 'History' }] });
  assert.equal(roomStore.takeQuestionAnimation('old'), false);
  roomStore.addQuestion({ id: 'new', text: 'New message' });
  assert.equal(roomStore.takeQuestionAnimation('new'), true);
  roomStore.setFilterTab('latest');
  roomStore.updateQuestion({ id: 'new', text: 'New message', upvotes: 1 });
  roomStore.addQuestion({ id: 'new', text: 'Duplicate event' });
  assert.equal(roomStore.takeQuestionAnimation('new'), false);
  assert.equal(get(roomStore).questions.length, 2);
  roomStore.reset();
});

test('multiple typists are independent and freeze/reset clears presence', () => {
  roomStore.reset();
  roomStore.setTyping({ id: 'one', alias: 'Panda', isTyping: true });
  roomStore.setTyping({ id: 'two', alias: 'Koala', isTyping: true });
  roomStore.setTyping({ id: 'one', alias: 'Panda', isTyping: true });
  assert.equal(get(roomStore).typingUsers.length, 2);
  roomStore.setTyping({ id: 'one', isTyping: false });
  assert.deepEqual(get(roomStore).typingUsers.map(x => x.id), ['two']);
  roomStore.setFrozen(true);
  roomStore.setTyping({ id: 'one', alias: 'Panda', isTyping: true });
  assert.deepEqual(get(roomStore).typingUsers, []);
  roomStore.reset();
});

test('room creator identity is preserved as Yeonly across leave and rejoin', async () => {
  const { getStoredHostKey, storeHostKey, removeStoredHostKey, isHostRoom } = await import('./roomStore.js');
  let mockStorage = {};
  const savedStorage = globalThis.localStorage;
  globalThis.localStorage = {
    getItem: (k) => mockStorage[k] || null,
    setItem: (k, v) => { mockStorage[k] = String(v); },
    removeItem: (k) => { delete mockStorage[k]; },
    clear: () => { mockStorage = {}; }
  };

  try {
    const roomId = 'ALGO01';
    const hostKey = 'sec_host_key_123';

    // 1. Creator creates room
    storeHostKey(roomId, hostKey);
    roomStore.setHostKey(hostKey, roomId);

    let state = get(roomStore);
    assert.equal(state.isHost, true);
    assert.equal(state.userAlias, 'Yeonly');
    assert.equal(state.hostKey, hostKey);
    assert.equal(isHostRoom(roomId), true);
    assert.equal(getStoredHostKey(roomId), hostKey);

    // 2. Creator leaves the room (resets in-memory store)
    roomStore.reset();
    state = get(roomStore);
    assert.equal(state.isHost, false);
    assert.equal(state.roomId, '');
    assert.equal(state.hostKey, '');

    // The key is still remembered in storage
    assert.equal(getStoredHostKey(roomId), hostKey);
    assert.equal(isHostRoom(roomId), true);

    // 3. Creator rejoins room (server sends back room state with isHost: true)
    roomStore.setRoomState({
      id: roomId,
      name: 'Algorithm Class',
      isHost: true,
      userAlias: 'Yeonly',
      questions: []
    });

    state = get(roomStore);
    assert.equal(state.roomId, roomId);
    assert.equal(state.isHost, true);
    assert.equal(state.userAlias, 'Yeonly');
    assert.equal(state.hostKey, hostKey);

    // 4. Cleaning up key when room ends
    removeStoredHostKey(roomId);
    assert.equal(isHostRoom(roomId), false);
    assert.equal(getStoredHostKey(roomId), '');
  } finally {
    globalThis.localStorage = savedStorage;
    roomStore.reset();
  }
});

test('votedQuestionIds tracks personal votes accurately and resets on new identity', () => {
  roomStore.reset();
  roomStore.setRoomState({ id: 'ROOM1', questions: [], votedQuestionIds: ['q1'] });
  assert.deepEqual(get(roomStore).votedQuestionIds, ['q1']);

  // Optimistic vote toggle
  roomStore.toggleVoteOptimistic('q2');
  assert.deepEqual(get(roomStore).votedQuestionIds, ['q1', 'q2']);

  // Toggle off q1
  roomStore.toggleVoteOptimistic('q1');
  assert.deepEqual(get(roomStore).votedQuestionIds, ['q2']);

  // Set explicit vote confirmation from server
  roomStore.setVotedQuestion('q3', true);
  assert.deepEqual(get(roomStore).votedQuestionIds, ['q2', 'q3']);
  roomStore.setVotedQuestion('q2', false);
  assert.deepEqual(get(roomStore).votedQuestionIds, ['q3']);

  // Rolling new participant identity resets personal votes
  roomStore.newParticipantIdentity();
  assert.deepEqual(get(roomStore).votedQuestionIds, []);
  roomStore.reset();
});

test('session identity survives unavailable storage and only changes when requested', (t) => {
  const saved = globalThis.sessionStorage;
  globalThis.sessionStorage = {
    getItem() { throw new Error('Storage blocked'); },
    setItem() { throw new Error('Storage blocked'); },
  };
  roomStore.reset();
  t.after(() => {
    if (saved === undefined) delete globalThis.sessionStorage;
    else globalThis.sessionStorage = saved;
    roomStore.reset();
  });
  const original = roomStore.initSession();
  assert.equal(roomStore.initSession(), original);
  const next = roomStore.newParticipantIdentity();
  assert.notEqual(next, original);
  assert.equal(roomStore.initSession(), next);
  assert.notEqual(roomStore.initSession(true), next);
});

test('room snapshots never inherit votes from a previous viewer or room', () => {
  roomStore.reset();
  roomStore.setRoomState({ id: 'ONE', votedQuestionIds: ['q'] });
  roomStore.setRoomState({ id: 'TWO' });
  assert.deepEqual(get(roomStore).votedQuestionIds, []);
  roomStore.reset();
});

test('filters keep source order intact and question deletion clears its personal vote', () => {
  roomStore.reset();
  const questions = [
    { id: 'a', status: 'unanswered', upvotes: 2, createdAt: '2026-10-18T00:00:00Z' },
    { id: 'b', status: 'pinned', upvotes: 0, createdAt: '2026-10-18T01:00:00Z' },
    { id: 'c', status: 'answered', upvotes: 9, createdAt: '2026-10-18T02:00:00Z' },
  ];
  roomStore.setRoomState({ id: 'ROOM', questions, activeUsers: 0, votedQuestionIds: ['b'] });
  assert.equal(get(roomStore).activeUsers, 0);
  assert.deepEqual(get(filteredQuestions).map(q => q.id), ['b', 'a']);
  roomStore.setFilterTab('answered');
  assert.deepEqual(get(filteredQuestions).map(q => q.id), ['c']);
  assert.deepEqual(get(roomStore).questions, questions);
  roomStore.deleteQuestion('b');
  assert.deepEqual(get(roomStore).votedQuestionIds, []);
  roomStore.reset();
});


