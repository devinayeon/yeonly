import { writable, get } from 'svelte/store';
import { roomStore, getStoredHostKey, storeHostKey, getStoredRoomRole, storeRoomRole } from './roomStore.js';
import { toast } from './toastStore.js';
import { parseServerEvent } from '../utils/protocol.js';

function createWSStore() {
  const { subscribe, set } = writable({
    connected: false,
    reconnecting: false,
  });

  /** @type {WebSocket | null} */
  let socket = null;
  let activeRoomId = '';
  let activeHostKey = '';
  let activeSessionId = '';
  let shouldReconnect = true;
  /** @type {ReturnType<typeof setTimeout> | null} */
  let reconnectTimer = null;
  let localTyping = false;
  let lastTypingSentAt = 0;
  /** @type {Map<string, ReturnType<typeof setTimeout>>} */
  const typingTimers = new Map();
  /** @type {Map<string, { timer: ReturnType<typeof setTimeout>, resolve: (accepted: boolean) => void }>} */
  const pendingSubmissions = new Map();
  let requestSequence = 0;

  function clearTyping() {
    for (const timer of typingTimers.values()) clearTimeout(timer);
    typingTimers.clear();
    localTyping = false;
    lastTypingSentAt = 0;
    roomStore.clearTyping();
  }

  /** @param {import('../types').TypingPayload} payload */
  function receiveTyping(payload) {
    clearTimeout(typingTimers.get(payload.id));
    typingTimers.delete(payload.id);
    roomStore.setTyping(payload);
    if (payload.isTyping) {
      typingTimers.set(payload.id, setTimeout(() => {
        roomStore.setTyping({ ...payload, isTyping: false });
        typingTimers.delete(payload.id);
      }, 6000));
    }
  }

  /** @param {string | undefined} requestId @param {boolean} accepted */
  function finishSubmission(requestId, accepted) {
    if (!requestId) return;
    const pending = pendingSubmissions.get(requestId);
    if (!pending) return;
    clearTimeout(pending.timer);
    pendingSubmissions.delete(requestId);
    pending.resolve(accepted);
  }

  function cancelSubmissions() {
    for (const requestId of pendingSubmissions.keys()) finishSubmission(requestId, false);
  }

  /** @param {boolean} active */
  function setTyping(active) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    if (active && get(roomStore).isFrozen) return;
    const now = Date.now();
    if (active === localTyping && (!active || now - lastTypingSentAt < 1500)) return;
    socket.send(JSON.stringify({ action: 'TYPING', isTyping: active }));
    localTyping = active;
    lastTypingSentAt = now;
  }

  /** @param {string} roomId @param {string} [hostKey] @param {import('../types').RoomRole | null} [forceRole] */
  function connect(roomId, hostKey = '', forceRole = null) {
    if (!forceRole && !hostKey) forceRole = getStoredRoomRole(roomId);
    if (forceRole === 'student') {
      hostKey = '';
    } else if (!hostKey) {
      hostKey = getStoredHostKey(roomId);
    } else {
      storeHostKey(roomId, hostKey);
    }

    const sessionId = roomStore.initSession();
    if (socket && socket.readyState <= WebSocket.OPEN && activeRoomId === roomId && activeHostKey === hostKey && activeSessionId === sessionId) return;
    disconnect();

    activeRoomId = roomId;
    activeHostKey = hostKey;
    activeSessionId = sessionId;
    shouldReconnect = true;
    roomStore.update(s => ({ ...s, votedQuestionIds: [] }));

    if (hostKey) {
      roomStore.setHostKey(hostKey, roomId);
    } else {
      roomStore.update(s => ({ ...s, isHost: false, hostKey: '' }));
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;

    let wsUrl = `${protocol}//${host}/ws?room=${encodeURIComponent(roomId)}&session=${encodeURIComponent(sessionId)}`;
    if (hostKey) {
      wsUrl += `&hostKey=${encodeURIComponent(hostKey)}`;
    }

    const connection = new WebSocket(wsUrl);
    socket = connection;

    socket.onopen = () => {
      if (socket !== connection) return;
      set({ connected: true, reconnecting: false });
      if (reconnectTimer) clearTimeout(reconnectTimer);
    };

    socket.onmessage = (event) => {
      if (socket !== connection) return;
      if (typeof event.data !== 'string') return;
      // Legacy batching remains supported; one bad message cannot discard the rest.
      for (const message of event.data.split('\n').filter(Boolean)) {
        try {
          const inbound = parseServerEvent(message);
          if (inbound) handleInbound(inbound);
          else console.warn('Ignored invalid or unsupported WebSocket event');
        } catch (err) {
          console.error('Failed to parse WebSocket message', err);
        }
      }
    };

    socket.onclose = () => {
      if (socket !== connection) return;
      clearTyping();
      cancelSubmissions();
      set({ connected: false, reconnecting: shouldReconnect });
      if (shouldReconnect) {
        reconnectTimer = setTimeout(() => {
          connect(activeRoomId, activeHostKey, activeHostKey ? 'host' : 'student');
        }, 3000);
      }
    };

    socket.onerror = (err) => {
      console.warn('WebSocket encountered error:', err);
    };
  }

  /** @param {import('../types').ServerEvent} event */
  function handleInbound(event) {
    switch (event.type) {
      case 'ROOM_STATE':
        clearTyping();
        storeRoomRole(event.payload.id, Boolean(event.payload.isHost));
        roomStore.setRoomState(event.payload);
        for (const user of event.payload.typingUsers || []) receiveTyping(user);
        break;

      case 'TYPING_UPDATED':
        receiveTyping(event.payload);
        break;

      case 'QUESTION_ACCEPTED':
        finishSubmission(event.requestId, true);
        break;

      case 'QUESTION_ADDED':
        roomStore.addQuestion(event.payload);
        toast.info(`Pertanyaan baru dari ${event.payload.authorAlias}`);
        break;

      case 'QUESTION_UPDATED':
        roomStore.updateQuestion(event.payload);
        break;

      case 'VOTE_UPDATED':
        roomStore.setVotedQuestion(event.payload.questionId, event.payload.hasVoted);
        break;

      case 'QUESTION_DELETED':
        roomStore.deleteQuestion(event.payload.questionId);
        break;

      case 'PULSE_UPDATED':
        roomStore.setPulseStats(event.payload);
        break;

      case 'ROOM_FROZEN':
        if (event.payload.isFrozen) clearTyping();
        roomStore.setFrozen(event.payload.isFrozen);
        if (event.payload.isFrozen) {
          toast.warning('Dosen menjeda pengiriman pertanyaan sementara.');
        } else {
          toast.success('Sesi pertanyaan dibuka kembali!');
        }
        break;

      case 'USER_COUNT':
        roomStore.setUserCount(event.payload.count);
        break;

      case 'ERROR':
        finishSubmission(event.requestId, false);
        toast.error(event.payload);
        break;

      case 'TOAST':
        toast.info(event.payload);
        break;
    }
  }

  /** @param {import('../types').ClientAction} action @param {import('../types').ActionPayload} [payload] */
  function send(action, payload = {}) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      toast.error('Koneksi terputus. Mencoba menghubungkan kembali...');
      return false;
    }
    const message = JSON.stringify({ action, ...payload });
    socket.send(message);
    return true;
  }

  function disconnect() {
    shouldReconnect = false;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = null;
    clearTyping();
    cancelSubmissions();
    if (socket) {
      const previous = socket;
      socket = null;
      previous.close();
    }
    set({ connected: false, reconnecting: false });
  }

  return {
    subscribe,
    connect,
    disconnect,
    setTyping,
    /** @param {string} text @returns {Promise<boolean>} */
    submitQuestion: (text) => new Promise(resolve => {
      const requestId = `question-${++requestSequence}`;
      const timer = setTimeout(() => {
        finishSubmission(requestId, false);
        toast.warning('Konfirmasi belum diterima. Periksa daftar pertanyaan sebelum mengirim ulang.');
      }, 10000);
      pendingSubmissions.set(requestId, { resolve, timer });
      if (!send('SUBMIT_QUESTION', { text, requestId })) finishSubmission(requestId, false);
    }),
    /** @param {string} questionId */
    upvoteQuestion: (questionId) => {
      if (!send('UPVOTE_QUESTION', { questionId })) return false;
      roomStore.toggleVoteOptimistic(questionId);
      return true;
    },
    /** @param {import('../types').RoomRole} role */
    switchRole: (role) => {
      const { roomId } = get(roomStore);
      if (!roomId) return;
      if (role === 'student') {
        roomStore.newParticipantIdentity();
        connect(roomId, '', 'student');
        toast.info('Beralih ke mode Peserta (Mahasiswa)');
      } else if (role === 'host') {
        const key = getStoredHostKey(roomId);
        if (key) {
          connect(roomId, key, 'host');
          toast.success('Beralih ke mode Host (Yeonly)');
        } else {
          toast.warning('Kunci Host tidak ditemukan untuk ruang ini.');
        }
      }
    },
    rollNewIdentity: () => {
      const { roomId } = get(roomStore);
      if (!roomId) return;
      roomStore.newParticipantIdentity();
      connect(roomId, '', 'student');
      toast.success('Identitas peserta baru aktif! Anda kini pengguna yang berbeda.');
    },
    /** @param {'paham' | 'bingung' | 'kecepatan'} vote */
    updatePulse: (vote) => send('UPDATE_PULSE', { vote }),
    /** @param {string} questionId */
    hostPin: (questionId) => {
      const { hostKey } = get(roomStore);
      send('HOST_PIN', { questionId, hostKey });
    },
    /** @param {string} questionId */
    hostResolve: (questionId) => {
      const { hostKey } = get(roomStore);
      send('HOST_RESOLVE', { questionId, hostKey });
    },
    /** @param {string} questionId */
    hostDelete: (questionId) => {
      const { hostKey } = get(roomStore);
      send('HOST_DELETE', { questionId, hostKey });
    },
    hostToggleFreeze: () => {
      const { hostKey } = get(roomStore);
      send('HOST_FREEZE', { hostKey });
    },
    hostClearPulse: () => {
      const { hostKey } = get(roomStore);
      send('HOST_CLEAR_PULSE', { hostKey });
    },
  };
}

export const ws = createWSStore();
