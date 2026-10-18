import test from 'node:test';
import assert from 'node:assert/strict';
import { parseServerEvent } from './protocol.js';

test('valid server snapshots and personal votes pass the protocol boundary', () => {
  const snapshot = { type: 'ROOM_STATE', payload: { id: 'ROOM', activeUsers: 0, questions: [], votedQuestionIds: [] } };
  assert.deepEqual(parseServerEvent(JSON.stringify(snapshot)), snapshot);
  const vote = { type: 'VOTE_UPDATED', payload: { questionId: 'q', hasVoted: false, upvotes: 0 } };
  assert.deepEqual(parseServerEvent(JSON.stringify(vote)), vote);
});

test('invalid payloads cannot corrupt typed stores', () => {
  for (const event of [
    null, [], { type: 'ROOM_STATE', payload: { id: 'ROOM', questions: {} } },
    { type: 'ROOM_STATE', payload: { id: 'ROOM', typingUsers: [null] } },
    { type: 'USER_COUNT', payload: { count: '2' } },
    { type: 'VOTE_UPDATED', payload: { questionId: 'q', hasVoted: 'false', upvotes: 1 } },
    { type: 'QUESTION_UPDATED', payload: { id: 'q', upvotes: -1 } },
    { type: 'QUESTION_ACCEPTED', requestId: 42 },
    { type: 'ERROR', payload: {} }, { type: 'UNSUPPORTED', payload: {} },
  ]) assert.equal(parseServerEvent(JSON.stringify(event)), null);
  assert.throws(() => parseServerEvent('{'), SyntaxError);
});
