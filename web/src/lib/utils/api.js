import { isRecord } from './protocol.js';

const API_ENV = import.meta.env ?? {};
const API_BASE = (API_ENV.VITE_API_BASE ?? '').replace(/\/+$/, '');

/**
 * Join a configured API origin with an API path.
 * An empty base keeps same-origin requests, which preserves the Vite dev proxy.
 * @param {string} base
 * @param {string} path
 * @returns {string}
 */
export function joinApiUrl(base, path) {
  return `${(base ?? '').replace(/\/+$/, '')}${path}`;
}

/** @param {string} path @param {RequestInit} [options] @returns {Promise<unknown>} */
async function requestJSON(path, options) {
  const response = await fetch(joinApiUrl(API_BASE, path), options);
  /** @type {unknown} */
  const data = await response.json();
  if (!response.ok) {
    throw new Error(isRecord(data) && typeof data.error === 'string' ? data.error : `Permintaan gagal (${response.status})`);
  }
  return data;
}

/** @param {AbortSignal} [signal] @returns {Promise<import('../types').PublicRoom[]>} */
export async function fetchPublicRooms(signal) {
  const data = await requestJSON('/api/rooms/public', { signal });
  if (!isRecord(data)) throw new Error('Daftar kelas tidak valid');
  // The Go server may encode an empty slice as null.
  const rooms = data.rooms ?? [];
  if (!Array.isArray(rooms) || !rooms.every(room => isRecord(room)
    && typeof room.id === 'string' && typeof room.name === 'string'
    && typeof room.createdAt === 'string' && Number.isFinite(Date.parse(room.createdAt))
    && typeof room.questionCount === 'number' && Number.isFinite(room.questionCount) && room.questionCount >= 0
    && typeof room.activeUsers === 'number' && Number.isFinite(room.activeUsers) && room.activeUsers >= 0)) {
    throw new Error('Daftar kelas tidak valid');
  }
  return rooms;
}

/** @param {string} name @param {boolean} isPublic @param {AbortSignal} [signal] */
export async function createRoom(name, isPublic, signal) {
  const data = await requestJSON('/api/rooms', {
    method: 'POST', signal,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, isPublic }),
  });
  if (!isRecord(data) || typeof data.roomId !== 'string' || !data.roomId || typeof data.hostKey !== 'string' || !data.hostKey) {
    throw new Error('Respons pembuatan kelas tidak valid');
  }
  return { roomId: data.roomId, hostKey: data.hostKey };
}

/** @param {string} code @param {AbortSignal} [signal] */
export async function checkRoom(code, signal) {
  await requestJSON(`/api/rooms/${encodeURIComponent(code)}`, { signal });
}
