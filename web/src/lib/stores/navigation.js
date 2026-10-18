import { roomStore } from './roomStore.js';
import { ws } from './wsStore.js';

// The URL is the reload/back/forward source of truth for the visible page.
export function restoreLocation() {
  const params = new URLSearchParams(window.location.search);
  const roomId = params.get('room')?.trim();
  if (roomId) {
    ws.connect(roomId.toUpperCase());
  } else {
    ws.disconnect();
    roomStore.reset();
  }
}

export function goHome() {
  // Invalidate the old socket before it can restore a room over the landing page.
  ws.disconnect();
  roomStore.reset();
  if (window.location.pathname !== '/' || window.location.search || window.location.hash) {
    window.history.pushState({}, '', '/');
  }
}
