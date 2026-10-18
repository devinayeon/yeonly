/** @param {unknown} value @returns {value is Record<string, unknown>} */
export const isRecord = value => value !== null && typeof value === 'object' && !Array.isArray(value);
/** @param {unknown} value @returns {value is string} */
const isString = value => typeof value === 'string';
/** @param {unknown} value @returns {value is number} */
const isCount = value => typeof value === 'number' && Number.isFinite(value) && value >= 0;
/** @param {unknown} value */
const isBoolean = value => typeof value === 'boolean';
/** @param {unknown} value @param {(item: unknown) => boolean} check */
const optional = (value, check) => value === undefined || check(value);
/** @param {unknown} value @param {(item: unknown) => boolean} check */
const arrayOf = (value, check) => Array.isArray(value) && value.every(check);

/** @param {unknown} value */
function isQuestion(value) {
  return isRecord(value)
    && ['id', 'roomId', 'authorAlias', 'authorId', 'authorRole', 'text', 'createdAt'].every(key => isString(value[key]))
    && isCount(value.upvotes) && isBoolean(value.hasVoted)
    && ['unanswered', 'pinned', 'answered'].includes(String(value.status))
    && Number.isFinite(Date.parse(String(value.createdAt)));
}

/** @param {unknown} value */
function isTyping(value) {
  return isRecord(value) && isString(value.id) && optional(value.alias, isString) && isBoolean(value.isTyping);
}

/** @param {unknown} value */
function isPulse(value) {
  return isRecord(value) && ['paham', 'bingung', 'kecepatan', 'total', 'pahamPct', 'bingungPct', 'kecepatanPct']
    .every(key => isCount(value[key]));
}

/** @param {unknown} value */
function isRoom(value) {
  return isRecord(value) && isString(value.id)
    && optional(value.participantId, isString) && optional(value.participantToken, isString)
    && optional(value.name, isString) && optional(value.userAlias, isString)
    && optional(value.isHost, isBoolean) && optional(value.isFrozen, isBoolean)
    && optional(value.activeUsers, isCount) && optional(value.pulseStats, isPulse)
    && optional(value.questions, items => arrayOf(items, isQuestion))
    && optional(value.votedQuestionIds, items => arrayOf(items, isString))
    && optional(value.typingUsers, items => arrayOf(items, isTyping));
}

/**
 * Validate at the network boundary before any store receives server data.
 * @param {string} raw
 * @returns {import('../types').ServerEvent | null}
 */
export function parseServerEvent(raw) {
  /** @type {unknown} */
  const value = JSON.parse(raw);
  if (!isRecord(value) || !isString(value.type)) return null;
  const { type, payload, requestId } = value;
  if (!optional(requestId, isString)) return null;
  let valid = false;
  switch (type) {
    case 'ROOM_STATE': valid = isRoom(payload); break;
    case 'QUESTION_ADDED':
    case 'QUESTION_UPDATED': valid = isQuestion(payload); break;
    case 'TYPING_UPDATED': valid = isTyping(payload); break;
    case 'PULSE_UPDATED': valid = isPulse(payload); break;
    case 'QUESTION_ACCEPTED': valid = isString(requestId); break;
    case 'VOTE_UPDATED':
      valid = isRecord(payload) && isString(payload.questionId) && isBoolean(payload.hasVoted) && isCount(payload.upvotes);
      break;
    case 'QUESTION_DELETED': valid = isRecord(payload) && isString(payload.questionId); break;
    case 'ROOM_FROZEN': valid = isRecord(payload) && isBoolean(payload.isFrozen); break;
    case 'USER_COUNT': valid = isRecord(payload) && isCount(payload.count); break;
    case 'ERROR':
    case 'ROOM_CLOSED':
    case 'IDENTITY_EXPIRED':
    case 'TOAST': valid = isString(payload); break;
  }
  // Each discriminant and its corresponding payload have been checked above.
  return valid ? /** @type {import('../types').ServerEvent} */ (value) : null;
}
