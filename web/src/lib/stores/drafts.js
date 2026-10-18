// Drafts stay in memory, scoped to the room and profile; never sent as typing data.
/** @type {Map<string, string>} */
const drafts = new Map();
/** @param {string} roomId @param {boolean} isHost */
export const draftKey = (roomId, isHost) => `${roomId}:${isHost ? 'host' : 'student'}`;
/** @param {string} key */
export const readDraft = key => drafts.get(key) || '';
/** @param {string} key @param {string} text */
export function saveDraft(key, text) { if (text) drafts.set(key, text); else drafts.delete(key); }
/** @param {string} roomId */
export function clearRoomDrafts(roomId) {
  for (const key of drafts.keys()) if (key.startsWith(`${roomId}:`)) drafts.delete(key);
}
