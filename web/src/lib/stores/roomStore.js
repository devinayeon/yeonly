import { writable, derived, get } from 'svelte/store';
import { readStorage, writeStorage, readHostKeys, removeStorage } from '../utils/storage.js';
import { createId } from '../utils/ids.js';

const PARTICIPANT_TOKEN_PREFIX = 'yeonly_participant_token:';

/** @param {string} roomId */
export function getStoredHostKey(roomId) {
  return roomId ? readHostKeys()[roomId.toUpperCase()] ?? '' : '';
}

/** @param {string} roomId @param {string} key */
export function storeHostKey(roomId, key) {
  if (!roomId || !key) return;
  const keys = { ...readHostKeys(), [roomId.toUpperCase()]: key };
  writeStorage('localStorage', 'yeonly_host_keys', JSON.stringify(keys));
}

/** @param {string} roomId */
export function removeStoredHostKey(roomId) {
  const keys = readHostKeys();
  delete keys[roomId.toUpperCase()];
  writeStorage('localStorage', 'yeonly_host_keys', JSON.stringify(keys));
}

/** @param {string} roomId */
export function getStoredParticipantToken(roomId) {
  if (!roomId) return '';
  return readStorage(
    'sessionStorage',
    `${PARTICIPANT_TOKEN_PREFIX}${roomId.toUpperCase()}`,
  ) ?? '';
}

/** @param {string} roomId @param {string} token */
export function storeParticipantToken(roomId, token) {
  if (!roomId || !token) return;
  writeStorage(
    'sessionStorage',
    `${PARTICIPANT_TOKEN_PREFIX}${roomId.toUpperCase()}`,
    token,
  );
}

/** @param {string} roomId */
export function removeStoredParticipantToken(roomId) {
  if (!roomId) return;
  removeStorage(
    'sessionStorage',
    `${PARTICIPANT_TOKEN_PREFIX}${roomId.toUpperCase()}`,
  );
}

/** @param {string} roomId */
export function isHostRoom(roomId) {
  return Boolean(getStoredHostKey(roomId));
}

/** @param {string} roomId */
export function getStoredRoomRole(roomId) {
  const role = readStorage('sessionStorage', `yeonly_room_role:${roomId.toUpperCase()}`);
  return role === 'host' || role === 'student' ? role : null;
}

/** @param {string} roomId @param {boolean} isHost */
export function storeRoomRole(roomId, isHost) {
  writeStorage('sessionStorage', `yeonly_room_role:${roomId.toUpperCase()}`, isHost ? 'host' : 'student');
}

/** @returns {import('../types').RoomState} */
function createInitialState() {
  return {
    roomId: '',
    roomName: '',
    isHost: false,
    hostKey: '',
    userAlias: '',
    isFrozen: false,
    questions: [],
    votedQuestionIds: [],
    typingUsers: [],
    pulseStats: {
      paham: 0,
      bingung: 0,
      kecepatan: 0,
      total: 0,
      pahamPct: 0,
      bingungPct: 0,
      kecepatanPct: 0,
    },
    activeUsers: 1,
    currentView: 'landing',
    filterTab: 'popular',
    isProjectorOpen: false,
    sessionId: '',
    participantToken: '',
  };
}

function createRoomStore() {
  const { subscribe, set, update } = writable(createInitialState());
  /** @type {Map<string, number>} */
  const pendingAnimations = new Map();

  return {
    subscribe,
    set,
    update,
    initSession: (forceNew = false) => {
      const sid = (!forceNew && (get({ subscribe }).sessionId || readStorage('sessionStorage', 'yeonly_session_id'))) || createId('sess_');
      writeStorage('sessionStorage', 'yeonly_session_id', sid);
      update(s => ({ ...s, sessionId: sid }));
      return sid;
    },
    newParticipantIdentity: () => {
      const { roomId } = get({ subscribe });
      if (roomId) removeStoredParticipantToken(roomId);

      const sid = createId('sess_');
      writeStorage('sessionStorage', 'yeonly_session_id', sid);
      update(s => ({
        ...s,
        sessionId: sid,
        participantToken: '',
        userAlias: '',
        isHost: false,
        hostKey: '',
        votedQuestionIds: [],
      }));
      return sid;
    },
    /** @param {import('../types').RoomSnapshot} payload */
    setRoomState: (payload) => {
      pendingAnimations.clear();
      const storedKey = getStoredHostKey(payload.id);
      const token = typeof payload.participantToken === 'string' ? payload.participantToken : '';
      if (token) storeParticipantToken(payload.id, token);

      update(s => ({
        ...s,
        roomId: payload.id,
        sessionId: payload.participantId || s.sessionId,
        participantToken: token,
        roomName: payload.name ?? '',
        isFrozen: payload.isFrozen ?? false,
        questions: payload.questions ?? [],
        votedQuestionIds: payload.votedQuestionIds ?? (payload.questions || []).filter(q => q.hasVoted).map(q => q.id),
        typingUsers: (payload.typingUsers ?? []).map(({ id, alias }) => ({ id, alias: alias ?? '' })),
        pulseStats: payload.pulseStats ?? s.pulseStats,
        activeUsers: payload.activeUsers ?? 1,
        userAlias: payload.userAlias || (payload.isHost ? 'Yeonly' : s.userAlias),
        isHost: Boolean(payload.isHost),
        hostKey: payload.isHost ? (s.hostKey || storedKey) : '',
        currentView: 'room',
      }));
    },
    /** @param {string} questionId @param {boolean} hasVoted */
    setVotedQuestion: (questionId, hasVoted) => {
      update(s => {
        const currentSet = new Set(s.votedQuestionIds);
        if (hasVoted) {
          currentSet.add(questionId);
        } else {
          currentSet.delete(questionId);
        }
        return { ...s, votedQuestionIds: Array.from(currentSet) };
      });
    },
    /** @param {string} questionId */
    toggleVoteOptimistic: (questionId) => {
      let willHaveVoted = false;
      update(s => {
        const currentSet = new Set(s.votedQuestionIds);
        if (currentSet.has(questionId)) {
          currentSet.delete(questionId);
          willHaveVoted = false;
        } else {
          currentSet.add(questionId);
          willHaveVoted = true;
        }
        return { ...s, votedQuestionIds: Array.from(currentSet) };
      });
      return willHaveVoted;
    },
    /** @param {import('../types').View} view */
    setCurrentView: (view) => {
      update(s => ({ ...s, currentView: view }));
    },
    /** @param {import('../types').Question} q */
    addQuestion: (q) => {
      update(s => {
        // Prevent duplicates
        if (s.questions.some(item => item.id === q.id)) return s;
        const now = Date.now();
        for (const [id, receivedAt] of pendingAnimations) {
          if (now - receivedAt > 3000) pendingAnimations.delete(id);
        }
        pendingAnimations.set(q.id, now);
        return {
          ...s,
          questions: [q, ...s.questions],
        };
      });
    },
    /** @param {import('../types').Question} q */
    updateQuestion: (q) => {
      update(s => ({
        ...s,
        questions: s.questions.map(item => (item.id === q.id ? q : item)),
        votedQuestionIds: [...s.votedQuestionIds.filter(id => id !== q.id), ...(q.hasVoted ? [q.id] : [])],
      }));
    },
    /** @param {string} questionId */
    deleteQuestion: (questionId) => {
      pendingAnimations.delete(questionId);
      update(s => ({
        ...s,
        questions: s.questions.filter(item => item.id !== questionId),
        votedQuestionIds: s.votedQuestionIds.filter(id => id !== questionId),
      }));
    },
    /** @param {import('../types').PulseStats} stats */
    setPulseStats: (stats) => {
      update(s => ({ ...s, pulseStats: stats }));
    },
    /** @param {boolean} isFrozen */
    setFrozen: (isFrozen) => {
      update(s => ({ ...s, isFrozen, typingUsers: isFrozen ? [] : s.typingUsers }));
    },
    /** @param {import('../types').TypingPayload} payload */
    setTyping: ({ id, alias = '', isTyping }) => {
      if (!id) return;
      update(s => {
        const existing = s.typingUsers.find(user => user.id === id);
        if (isTyping && !s.isFrozen && existing) {
          if (existing.alias === alias) return s;
          return { ...s, typingUsers: s.typingUsers.map(user => user.id === id ? { id, alias } : user) };
        }
        if (!isTyping && !existing) return s;
        const others = s.typingUsers.filter(user => user.id !== id);
        return { ...s, typingUsers: isTyping && !s.isFrozen ? [...others, { id, alias }] : others };
      });
    },
    clearTyping: () => update(s => ({ ...s, typingUsers: [] })),
    /** @param {string} id */
    takeQuestionAnimation: (id) => {
      const receivedAt = pendingAnimations.get(id);
      pendingAnimations.delete(id);
      return receivedAt !== undefined && Date.now() - receivedAt < 3000;
    },
    /** @param {number} count */
    setUserCount: (count) => {
      update(s => ({ ...s, activeUsers: count }));
    },
    /** @param {import('../types').FilterTab} tab */
    setFilterTab: (tab) => {
      update(s => ({ ...s, filterTab: tab }));
    },
    /** @param {boolean} [show] */
    toggleProjector: (show) => {
      update(s => ({ ...s, isProjectorOpen: typeof show === 'boolean' ? show : !s.isProjectorOpen }));
    },
    /** @param {string} key @param {string} [roomId] */
    setHostKey: (key, roomId = '') => {
      if (roomId && key) {
        storeHostKey(roomId, key);
      }
      update(s => ({ ...s, hostKey: key, isHost: true, userAlias: 'Yeonly' }));
    },
    reset: () => {
      pendingAnimations.clear();
      set(createInitialState());
    },
  };
}

export const roomStore = createRoomStore();

// Derived filtered questions
export const filteredQuestions = derived(roomStore, ($room) => {
  const { questions, filterTab } = $room;

  switch (filterTab) {
    case 'pinned':
      return questions.filter(q => q.status === 'pinned');
    case 'answered':
      return questions.filter(q => q.status === 'answered');
    case 'latest':
      // Show unanswered, sorted newest first
      return questions
        .filter(q => q.status !== 'answered')
        .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
    case 'popular':
    default:
      // Show unanswered & pinned, sorted by upvotes descending, then pinned at top
      return questions
        .filter(q => q.status !== 'answered')
        .sort((a, b) => {
          if (a.status === 'pinned' && b.status !== 'pinned') return -1;
          if (b.status === 'pinned' && a.status !== 'pinned') return 1;
          if (b.upvotes === a.upvotes) {
            return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
          }
          return b.upvotes - a.upvotes;
        });
  }
});
