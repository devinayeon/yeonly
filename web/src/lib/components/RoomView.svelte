<script>
  import { onDestroy } from 'svelte';
  import { roomStore, filteredQuestions, isHostRoom } from '../stores/roomStore.js';
  import { ws } from '../stores/wsStore.js';
  import { copyText } from '../utils/clipboard.js';
  import { goHome } from '../stores/navigation.js';
  import { toast } from '../stores/toastStore.js';
  import QuestionCard from './QuestionCard.svelte';
  import QuestionInput from './QuestionInput.svelte';
  import TypingIndicator from './TypingIndicator.svelte';
  import HostToolbar from './HostToolbar.svelte';

  let copied = false;
  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let copyTimer;
  onDestroy(() => clearTimeout(copyTimer));

  /** @param {import('../types').FilterTab} tab */
  function setTab(tab) {
    roomStore.setFilterTab(tab);
  }

  async function copyRoomCode() {
    if (!$roomStore.roomId) return;
    if (!await copyText($roomStore.roomId)) {
      toast.error('Kode kelas gagal disalin. Silakan salin secara manual.');
      return;
    }
    copied = true;
    toast.success(`Kode ruang ${$roomStore.roomId} disalin`);
    clearTimeout(copyTimer);
    copyTimer = setTimeout(() => { copied = false; }, 2000);
  }

  function switchRoom() {
    goHome();
  }

  function leaveRoom() {
    if (confirm('Keluar dari ruang kuliah ini?')) {
      goHome();
    }
  }

  $: totalQuestions = $roomStore.questions.length;
  $: pinnedCount = $roomStore.questions.filter(q => q.status === 'pinned').length;
  $: answeredCount = $roomStore.questions.filter(q => q.status === 'answered').length;
</script>

<div class="chat-room-layout animate-fade-in">
  <!-- Top Native Header (Apple HIG Frosted Glass) -->
  <header class="chat-top-header">
    <div class="header-left">
      <!-- Switch Room Button -->
      <button type="button" class="btn-switch-room" on:click={switchRoom} title="Pilih atau ganti ruang kuliah">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
          <line x1="19" y1="12" x2="5" y2="12"></line>
          <polyline points="12 19 5 12 12 5"></polyline>
        </svg>
        <span>Ganti Ruang</span>
      </button>

      <div class="room-identity-group">
        <h2 class="room-title">{$roomStore.roomName || 'Ruang Kuliah'}</h2>
        
        {#if $roomStore.isHost}
          <div class="user-role-pill host">
            <span class="role-text">Yeonly · Asisten Praktikum</span>
            <button type="button" class="btn-switch-role on-host" on:click={() => ws.switchRole('student')} title="Coba peran Peserta (Mahasiswa)">
              Beralih ke Peserta
            </button>
          </div>
        {:else}
          <div class="user-role-pill student">
            <span class="role-text">{$roomStore.userAlias || 'Peserta'}</span>
            {#if isHostRoom($roomStore.roomId)}
              <button type="button" class="btn-switch-role host-return" on:click={() => ws.switchRole('host')} title="Kembali ke peran Host (Yeonly)">
                Ke Host
              </button>
            {/if}
          </div>
        {/if}

        <button type="button" class="room-code-tag" on:click={copyRoomCode} title="Klik untuk salin kode ruang">
          <span class="code-label">KODE:</span>
          <span class="code-value">{$roomStore.roomId}</span>
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            {#if copied}
              <path d="M20 6L9 17l-5-5"></path>
            {:else}
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            {/if}
          </svg>
        </button>
      </div>
    </div>

    <div class="header-right">
      <!-- Online Students Badge with Semantic Live Dot -->
      <div class="students-count-badge">
        <span class="online-dot"></span>
        <span>{$roomStore.activeUsers} Peserta</span>
      </div>

      <!-- Proyektor Button -->
      <button type="button" class="btn-header-tool" on:click={() => roomStore.toggleProjector(true)} title="Tampilkan Proyektor QR Code">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
          <line x1="8" y1="21" x2="16" y2="21"></line>
          <line x1="12" y1="17" x2="12" y2="21"></line>
        </svg>
        <span class="hide-mobile">Proyektor</span>
      </button>

      <!-- Leave Room Button -->
      <button type="button" class="btn-header-tool exit" on:click={leaveRoom} title="Keluar dari sesi kuliah">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
          <polyline points="16 17 21 12 16 7"></polyline>
          <line x1="21" y1="12" x2="9" y2="12"></line>
        </svg>
      </button>
    </div>
  </header>

  {#if !$ws.connected}
    <div class="connection-status" role="status"><span class="motion-spinner" aria-hidden="true"></span> Menghubungkan kembali… Pesan Anda tetap tersimpan.</div>
  {/if}
  <!-- Host Toolbar (if lecturer) -->
  {#if $roomStore.isHost}
    <div class="host-toolbar-wrapper">
      <HostToolbar />
    </div>
  {/if}

  <!-- Monochrome Filter Pills Bar -->
  <div class="chat-sub-nav">
    <div class="filter-pills" role="tablist">
      <button 
        type="button" 
        role="tab"
        aria-selected={$roomStore.filterTab === 'popular'}
        class="filter-pill {$roomStore.filterTab === 'popular' ? 'active' : ''}" 
        on:click={() => setTab('popular')}
      >
        Terpopuler
      </button>
      <button 
        type="button" 
        role="tab"
        aria-selected={$roomStore.filterTab === 'latest'}
        class="filter-pill {$roomStore.filterTab === 'latest' ? 'active' : ''}" 
        on:click={() => setTab('latest')}
      >
        Terbaru
      </button>
      <button 
        type="button" 
        role="tab"
        aria-selected={$roomStore.filterTab === 'pinned'}
        class="filter-pill {$roomStore.filterTab === 'pinned' ? 'active' : ''}" 
        on:click={() => setTab('pinned')}
      >
        Dibahas ({pinnedCount})
      </button>
      <button 
        type="button" 
        role="tab"
        aria-selected={$roomStore.filterTab === 'answered'}
        class="filter-pill {$roomStore.filterTab === 'answered' ? 'active' : ''}" 
        on:click={() => setTab('answered')}
      >
        Selesai ({answeredCount})
      </button>
    </div>

    <span class="total-questions-text">{totalQuestions} pertanyaan</span>
  </div>

  <!-- Chat Message Stream (Timeline) -->
  <div class="chat-stream-container">
    <!-- System Security & Privacy Notice -->
    <div class="system-timeline-card">
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" class="system-icon">
        <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
      </svg>
      <span class="system-text">Ruang kelas aktif. Semua pertanyaan 100% anonim dan data otomatis terhapus saat sesi selesai.</span>
    </div>

    {#if $filteredQuestions.length === 0}
      <div class="empty-chat-state">
        <div class="empty-bubble-icon" aria-hidden="true">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
          </svg>
        </div>
        <h3>Belum Ada Pertanyaan</h3>
        <p>
          {#if $roomStore.filterTab === 'pinned'}
            Belum ada pertanyaan yang sedang dibahas oleh asisten.
          {:else if $roomStore.filterTab === 'answered'}
            Belum ada pertanyaan yang ditandai selesai.
          {:else}
            Jadilah yang pertama bertanya. Tulis pertanyaanmu pada kotak di bawah.
          {/if}
        </p>
      </div>
    {:else}
      <div class="messages-list">
        {#each $filteredQuestions as q (q.id)}
          <QuestionCard question={q} />
        {/each}
      </div>
    {/if}

    <TypingIndicator />
  </div>

  <!-- Pinned Bottom Chat Input Bar -->
  {#key `${$roomStore.roomId}:${$roomStore.isHost}`}<QuestionInput />{/key}
</div>

<style>
  .chat-room-layout {
    display: flex;
    flex-direction: column;
    min-height: 100vh;
    max-width: 860px;
    margin: 0 auto;
    background: var(--bg-app);
  }

  /* Top Chat Header */
  .chat-top-header {
    position: sticky;
    top: 0;
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 20px;
    background: rgba(255, 255, 255, 0.86);
    backdrop-filter: saturate(180%) blur(20px);
    -webkit-backdrop-filter: saturate(180%) blur(20px);
    border-bottom: 1px solid var(--border-subtle);
    gap: 12px;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .btn-switch-room {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
    padding: 6px 12px;
    border-radius: var(--radius-pill);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    flex-shrink: 0;
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .btn-switch-room:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
    border-color: var(--border-medium);
  }

  .room-identity-group {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .room-title {
    font-size: 1rem;
    font-weight: 700;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Role Badges */
  .user-role-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 2px 8px;
    border-radius: var(--radius-pill);
    font-size: 0.72rem;
    font-weight: 600;
    flex-shrink: 0;
  }

  .user-role-pill.host {
    background: var(--mono-black);
    color: var(--mono-white);
  }

  .user-role-pill.student {
    background: var(--bg-subtle);
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
  }

  .role-text {
    font-weight: 700;
  }

  .btn-switch-role {
    font-size: 0.68rem;
    font-weight: 500;
    padding: 1px 7px;
    border-radius: var(--radius-pill);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    cursor: pointer;
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .btn-switch-role:hover {
    color: var(--text-primary);
    border-color: var(--border-strong);
  }

  .btn-switch-role.on-host {
    background: rgba(255, 255, 255, 0.2);
    color: #ffffff;
    border-color: rgba(255, 255, 255, 0.35);
  }

  .btn-switch-role.on-host:hover {
    background: rgba(255, 255, 255, 0.3);
  }

  .btn-switch-role.host-return {
    color: var(--text-primary);
    font-weight: 600;
  }

  .room-code-tag {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-family: var(--font-mono);
    font-size: 0.72rem;
    font-weight: 600;
    color: var(--text-primary);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    padding: 3px 8px;
    border-radius: var(--radius-pill);
    flex-shrink: 0;
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .room-code-tag:hover {
    border-color: var(--border-strong);
    background: var(--bg-hover);
  }

  .code-label {
    opacity: 0.65;
    font-size: 0.65rem;
  }

  /* Header Right */
  .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .students-count-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--bg-surface);
    padding: 5px 11px;
    border-radius: var(--radius-pill);
    border: 1px solid var(--border-subtle);
  }

  .online-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--semantic-green);
  }

  .btn-header-tool {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
    padding: 6px 12px;
    border-radius: var(--radius-pill);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .btn-header-tool:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
    border-color: var(--border-medium);
  }

  .btn-header-tool.exit:hover {
    color: var(--semantic-red);
    background: var(--semantic-red-subtle);
    border-color: rgba(215, 0, 21, 0.2);
  }

  /* Host Toolbar */
  .host-toolbar-wrapper {
    padding: 12px 18px 0;
  }

  /* Sub Nav / Filter Pills */
  .chat-sub-nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 20px;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--bg-app);
    gap: 12px;
  }

  .filter-pills {
    display: flex;
    align-items: center;
    gap: 3px;
    background: var(--bg-subtle);
    padding: 3px;
    border-radius: var(--radius-pill);
    border: 1px solid var(--border-subtle);
    overflow-x: auto;
  }

  .filter-pill {
    padding: 5px 13px;
    border-radius: var(--radius-pill);
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--text-secondary);
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
    white-space: nowrap;
  }

  .filter-pill:hover:not(.active) {
    color: var(--text-primary);
  }

  .filter-pill.active {
    background: var(--mono-black);
    color: var(--mono-white);
    box-shadow: var(--shadow-sm);
  }

  .total-questions-text {
    font-size: 0.76rem;
    color: var(--text-tertiary);
    font-weight: 500;
    white-space: nowrap;
  }

  /* Chat Stream */
  .chat-stream-container {
    flex: 1;
    padding: 18px 18px 10px;
    display: flex;
    flex-direction: column;
  }

  .system-timeline-card {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    align-self: center;
    padding: 6px 16px;
    border-radius: var(--radius-pill);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    font-size: 0.76rem;
    color: var(--text-secondary);
    margin-bottom: 22px;
    text-align: center;
    max-width: 90%;
    box-shadow: var(--shadow-sm);
  }

  .system-icon {
    color: var(--text-tertiary);
    flex-shrink: 0;
  }

  .empty-chat-state {
    margin: 40px auto;
    text-align: center;
    max-width: 400px;
    padding: 32px 20px;
  }

  .empty-bubble-icon {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    color: var(--text-tertiary);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 12px;
  }

  .empty-chat-state h3 {
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 6px;
  }

  .empty-chat-state p {
    font-size: 0.88rem;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .messages-list {
    display: flex;
    flex-direction: column;
    margin-bottom: 16px;
  }

  @media (max-width: 640px) {
    .chat-top-header {
      padding: 10px 14px;
    }
    .room-title {
      max-width: 120px;
      font-size: 0.92rem;
    }
    .hide-mobile {
      display: none;
    }
    .total-questions-text {
      display: none;
    }
  }
</style>
