<script>
  import { roomStore, removeStoredHostKey } from '../stores/roomStore.js';
  import { ws } from '../stores/wsStore.js';
  import { toast } from '../stores/toastStore.js';

  function toggleFreeze() {
    ws.hostToggleFreeze();
  }

  function downloadRecap() {
    if (!$roomStore.roomId) return;
    window.open(`/api/rooms/${encodeURIComponent($roomStore.roomId)}/recap`, '_blank');
    toast.success('Mengunduh rekap Q&A sesi kuliah (.md)...');
  }

  function handleEndRoom() {
    if (confirm('Apakah Anda yakin ingin menutup ruang kuliah ini? Semua data di RAM akan terhapus permanen.')) {
      if ($roomStore.roomId) {
        removeStoredHostKey($roomStore.roomId);
      }
      ws.disconnect();
      roomStore.reset();
      window.history.pushState({}, '', '/');
      toast.info('Ruang kuliah telah ditutup.');
    }
  }
</script>

<div class="host-toolbar apple-card">
  <div class="toolbar-info">
    <div class="host-badge-row">
      <span class="status-pill dark">Host</span>
      <span class="host-caption">Kendali moderasi sesi kelas Anda</span>
    </div>
  </div>

  <div class="toolbar-btns">
    <button 
      type="button"
      class="btn-apple-secondary btn-tool {$roomStore.isFrozen ? 'frozen' : ''}" 
      on:click={toggleFreeze}
      title={$roomStore.isFrozen ? 'Buka sesi pertanyaan' : 'Jeda sementara pengiriman pertanyaan'}
    >
      {#if $roomStore.isFrozen}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
          <polygon points="5 3 19 12 5 21 5 3"></polygon>
        </svg>
        <span>Buka Pertanyaan</span>
      {:else}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
          <rect x="6" y="4" width="4" height="16"></rect>
          <rect x="14" y="4" width="4" height="16"></rect>
        </svg>
        <span>Jeda Pertanyaan</span>
      {/if}
    </button>

    <button type="button" class="btn-apple-secondary btn-tool" on:click={downloadRecap} title="Unduh daftar pertanyaan (.md)">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
        <polyline points="7 10 12 15 17 10"></polyline>
        <line x1="12" y1="15" x2="12" y2="3"></line>
      </svg>
      <span>Unduh Rekap</span>
    </button>

    <button type="button" class="btn-apple-secondary btn-tool end" on:click={handleEndRoom} title="Akhiri sesi dan hapus data">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="15" y1="9" x2="9" y2="15"></line>
        <line x1="9" y1="9" x2="15" y2="15"></line>
      </svg>
      <span>Tutup Ruang</span>
    </button>
  </div>
</div>

<style>
  .host-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 18px;
    margin-bottom: 16px;
    border-radius: var(--radius-lg);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
  }

  .toolbar-info {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .host-badge-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .host-caption {
    font-size: 0.82rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .toolbar-btns {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .btn-tool {
    padding: 7px 14px;
    font-size: 0.82rem;
    font-weight: 600;
  }

  .btn-tool.frozen {
    background: var(--mono-black);
    color: var(--mono-white);
    border-color: var(--mono-black);
  }

  .btn-tool.end:hover {
    color: var(--semantic-red);
    background: var(--semantic-red-subtle);
    border-color: rgba(215, 0, 21, 0.2);
  }

  @media (max-width: 680px) {
    .host-toolbar {
      flex-direction: column;
      gap: 12px;
      align-items: flex-start;
    }
    .toolbar-btns {
      width: 100%;
      flex-wrap: wrap;
    }
    .host-badge-row { flex-wrap: wrap; }
  }
</style>
