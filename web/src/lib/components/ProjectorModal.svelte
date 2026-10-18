<script>
  import QRCode from 'qrcode';
  import { copyText } from '../utils/clipboard.js';
  import { roomStore } from '../stores/roomStore.js';
  import { toast } from '../stores/toastStore.js';

  /** @type {HTMLCanvasElement | undefined} */
  let canvasEl;

  $: joinUrl = typeof window !== 'undefined' 
    ? `${window.location.origin}/?join=${encodeURIComponent($roomStore.roomId)}`
    : '';

  $: if (canvasEl && joinUrl) {
    renderQR();
  }

  function renderQR() {
    if (!canvasEl || !joinUrl) return;
    QRCode.toCanvas(canvasEl, joinUrl, {
      width: 240,
      margin: 2,
      color: {
        dark: '#1d1d1f',
        light: '#ffffff',
      },
    }, (error) => {
      if (error) console.error('Failed to render QR', error);
    });
  }

  function close() {
    roomStore.toggleProjector(false);
  }

  async function copyJoinUrl() {
    if (!await copyText(joinUrl)) {
      toast.error('Link kelas gagal disalin. Silakan salin secara manual.');
      return;
    }
    toast.success('Link langsung kelas berhasil disalin!');
  }
</script>

<div class="projector-overlay" role="dialog" aria-modal="true" tabindex="-1" on:click|self={close} on:keydown|self={(e) => { if (e.key === 'Escape') close(); }}>
  <div class="projector-card apple-card">
    <button type="button" class="btn-close" on:click={close} title="Tutup mode proyektor (Esc)">✕</button>

    <div class="projector-top">
      <span class="status-pill dark">Mode Layar Kelas</span>
      <h2 class="projector-room-title">{$roomStore.roomName || 'Ruang Diskusi Kuliah'}</h2>
      <p class="projector-room-desc">Pindai kode QR dengan kamera ponsel atau masukkan kode di bawah:</p>
    </div>

    <div class="projector-center">
      <div class="qr-box">
        <canvas bind:this={canvasEl} class="qr-canvas-element"></canvas>
        <span class="qr-scan-guide">Pindai untuk langsung bergabung</span>
      </div>

      <div class="code-box-display">
        <div class="code-badge-block">
          <span class="code-kicker">Kode Ruang Kelas</span>
          <span class="code-characters">{$roomStore.roomId}</span>
        </div>

        <button type="button" class="btn-apple-secondary btn-copy-link" on:click={copyJoinUrl}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
          </svg>
          <span>Salin Link Langsung</span>
        </button>

        <div class="connected-counter-badge">
          <span class="live-indicator"></span>
          <span><strong>{$roomStore.activeUsers}</strong> Mahasiswa Terhubung</span>
        </div>
      </div>
    </div>

    <div class="projector-bottom">
      <span>Semua pertanyaan bersifat 100% anonim. Mahasiswa bebas bertanya tanpa rasa ragu atau cemas.</span>
    </div>
  </div>
</div>

<style>
  .projector-overlay {
    position: fixed;
    inset: 0;
    z-index: 90;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: saturate(180%) blur(20px);
    -webkit-backdrop-filter: saturate(180%) blur(20px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
  }

  .projector-card {
    position: relative;
    max-width: 620px;
    width: 100%;
    padding: 36px 32px 28px;
    border-radius: var(--radius-xl);
    text-align: center;
    box-shadow: var(--shadow-lg);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
  }

  .btn-close {
    position: absolute;
    top: 18px;
    right: 18px;
    width: 32px;
    height: 32px;
    border-radius: var(--radius-pill);
    background: var(--bg-subtle);
    color: var(--text-secondary);
    font-size: 0.95rem;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .btn-close:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .projector-top {
    margin-bottom: 24px;
  }

  .projector-room-title {
    font-family: var(--font-display);
    font-size: 1.75rem;
    font-weight: 800;
    letter-spacing: -0.02em;
    color: var(--text-primary);
    margin: 12px 0 6px;
  }

  .projector-room-desc {
    font-size: 0.92rem;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .projector-center {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 32px;
    margin-bottom: 24px;
  }

  .qr-box {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .qr-canvas-element {
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-sm);
    background: #ffffff;
    border: 1px solid var(--border-subtle);
    padding: 6px;
  }

  .qr-scan-guide {
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--text-tertiary);
  }

  .code-box-display {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
  }

  .code-badge-block {
    background: var(--bg-subtle);
    border: 1px solid var(--border-medium);
    padding: 16px 28px;
    border-radius: var(--radius-md);
  }

  .code-kicker {
    display: block;
    font-size: 0.72rem;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 4px;
  }

  .code-characters {
    font-family: var(--font-mono);
    font-size: 2.8rem;
    font-weight: 800;
    letter-spacing: 0.12em;
    color: var(--mono-black);
    line-height: 1;
  }

  .btn-copy-link {
    width: 100%;
    font-size: 0.85rem;
    padding: 8px 16px;
  }

  .connected-counter-badge {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: 0.82rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .live-indicator {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--semantic-green);
  }

  .projector-bottom {
    padding-top: 18px;
    border-top: 1px solid var(--border-subtle);
    font-size: 0.82rem;
    color: var(--text-tertiary);
  }

  @media (max-width: 600px) {
    .projector-center {
      flex-direction: column;
      gap: 20px;
    }
    .projector-card {
      padding: 28px 18px;
    }
  }
</style>
