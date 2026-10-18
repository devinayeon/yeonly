<script>
  import { onMount, onDestroy } from 'svelte';
  import { roomStore, isHostRoom } from '../stores/roomStore.js';
  import { ws } from '../stores/wsStore.js';
  import { toast } from '../stores/toastStore.js';
  import { fetchPublicRooms, createRoom, checkRoom } from '../utils/api.js';
  import { goHome } from '../stores/navigation.js';

  let activeTab = 'classes'; // 'classes' (Mahasiswa) | 'create' (Asisten/Dosen)
  let roomName = '';
  let quickCode = '';
  let isPublic = true;
  let isSubmitting = false;

  /** @type {import('../types').PublicRoom[]} */
  let publicRooms = [];
  let isLoadingRooms = true;
  let isFetchingRooms = false;
  const requests = new AbortController();
  /** @type {ReturnType<typeof setInterval> | null} */
  let pollInterval = null;

  // Search & Pagination (Multi-column responsive grid: 6 items per page)
  let searchQuery = '';
  let currentPage = 1;
  const ITEMS_PER_PAGE = 6;

  $: filteredRooms = publicRooms.filter(r => 
    r.name.toLowerCase().includes(searchQuery.trim().toLowerCase()) ||
    r.id.toLowerCase().includes(searchQuery.trim().toLowerCase())
  );
  $: totalPages = Math.max(1, Math.ceil(filteredRooms.length / ITEMS_PER_PAGE));
  $: if (currentPage > totalPages) currentPage = totalPages;
  $: paginatedRooms = filteredRooms.slice((currentPage - 1) * ITEMS_PER_PAGE, currentPage * ITEMS_PER_PAGE);

  onMount(() => {
    fetchActiveRooms();
    pollInterval = setInterval(fetchActiveRooms, 3000);

    const params = new URLSearchParams(window.location.search);
    const code = params.get('join');
    if (code) {
      handleJoin(code.toUpperCase());
    }
  });

  onDestroy(() => {
    requests.abort();
    if (pollInterval) clearInterval(pollInterval);
  });

  async function fetchActiveRooms() {
    if (isFetchingRooms || requests.signal.aborted) return;
    isFetchingRooms = true;
    try {
      publicRooms = await fetchPublicRooms(requests.signal);
    } catch (err) {
      if (!requests.signal.aborted) console.error('Failed to fetch rooms', err);
    } finally {
      isLoadingRooms = false;
      isFetchingRooms = false;
    }
  }

  async function handleCreate() {
    if (isSubmitting) return;
    const trimmed = roomName.trim();
    if (!trimmed) {
      toast.warning('Masukkan nama mata kuliah atau topik sesi kelas.');
      return;
    }

    isSubmitting = true;
    try {
      const data = await createRoom(trimmed, isPublic, requests.signal);
      if (requests.signal.aborted) return;
      roomStore.setHostKey(data.hostKey, data.roomId);
      window.history.pushState({}, '', `/?room=${data.roomId}`);
      ws.connect(data.roomId, data.hostKey);
      toast.success(`Sesi kelas "${trimmed}" aktif!`);
    } catch (err) {
      if (requests.signal.aborted) return;
      toast.error(err instanceof Error ? err.message : 'Terjadi kesalahan saat memulai sesi');
    } finally {
      isSubmitting = false;
    }
  }

  /** @param {string} [codeToJoin] @param {import('../types').RoomRole | null} [asRole] */
  async function handleJoin(codeToJoin, asRole = null) {
    if (isSubmitting) return;
    const rawCode = codeToJoin !== undefined ? codeToJoin : quickCode;
    const code = rawCode.trim().toUpperCase();
    if (!code || code.length < 3) {
      toast.warning('Masukkan kode kelas yang valid (minimal 3 karakter).');
      return;
    }

    isSubmitting = true;
    try {
      await checkRoom(code, requests.signal);
      if (requests.signal.aborted) return;
      window.history.pushState({}, '', `/?room=${code}`);
      if (asRole === 'student') {
        roomStore.newParticipantIdentity();
        ws.connect(code, '', 'student');
      } else {
        ws.connect(code, '', asRole);
      }
    } catch (err) {
      if (requests.signal.aborted) return;
      toast.error(err instanceof Error ? err.message : 'Gagal bergabung ke kelas');
    } finally {
      isSubmitting = false;
    }
  }
</script>

<div class="landing-page-root animate-fade-in">
  <!-- Subtle Ethereal Watermark Rings (Exact art-direction from Image 2 reference) -->
  <div class="bg-watermark-geometry" aria-hidden="true">
    <div class="watermark-circle circle-1"></div>
    <div class="watermark-circle circle-2"></div>
  </div>

  <div class="landing-content-flow">
    <!-- 1. HERO SPLIT SECTION (Horizontal Typography Rhythm inspired by Image 2) -->
    <section class="hero-split-section">
      <div class="hero-anchor-left">
        <span class="hero-eyebrow">Platform Tanya-Jawab Kuliah</span>
        <h1 class="hero-main-title">
          Tanya apa saja.<br />
          <span class="hero-title-accent">100% anonim.</span>
        </h1>
        <p class="hero-mini-kicker">Diskusi interaktif ruang kelas bebas rasa cemas</p>
      </div>

      <div class="hero-narrative-right">
        <div class="hero-brand-block">
          <button type="button" class="hero-brand-logo" on:click={goHome} title="Yeonly">
            Yeonly
          </button>
        </div>
        <p class="hero-statement">
          Ajukan pertanyaan di kelas secara bebas dan dapatkan jawaban langsung. Tanpa akun mahasiswa, tanpa riwayat tersimpan, dan data otomatis bersih saat kelas selesai.
        </p>
      </div>
    </section>

    <!-- 2. DIRECT ACTION BAR (The Primary Workflow for Students with a Code) -->
    <section class="direct-join-section">
      <div class="join-bar-card apple-card">
        <form on:submit|preventDefault={() => handleJoin(quickCode)} class="join-bar-form">
          <div class="join-bar-left">
            <span class="join-bar-prompt">Punya kode ruang kelas?</span>
            <div class="join-input-group">
              <span class="code-prefix" aria-hidden="true">KODE</span>
              <input 
                id="quick-code-input"
                type="text" 
                bind:value={quickCode}
                placeholder="Ketik 6 karakter (cth: B84KQ6)"
                maxlength="8"
                class="quick-code-field"
                autocomplete="off"
              />
            </div>
          </div>

          <button 
            type="submit" 
            class="btn-apple-primary btn-join-direct"
            disabled={isSubmitting || !quickCode.trim()}
          >
            <span>{isSubmitting ? 'Memproses...' : 'Gabung Kelas'}</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <line x1="5" y1="12" x2="19" y2="12"></line>
              <polyline points="12 5 19 12 12 19"></polyline>
            </svg>
          </button>
        </form>
      </div>
    </section>

    <!-- 3. SESSIONS WORKSPACE DECK (Expansive Responsive Multi-column Grid) -->
    <section class="sessions-workspace-section">
      <!-- Workspace Navigation Bar -->
      <div class="workspace-nav-bar">
        <div class="workspace-nav-segmented" role="tablist">
          <button 
            type="button" 
            role="tab"
            aria-selected={activeTab === 'classes'}
            class="tab-btn {activeTab === 'classes' ? 'active' : ''}"
            on:click={() => activeTab = 'classes'}
          >
            <span>Kelas Berlangsung</span>
            {#if publicRooms.length > 0}
              <span class="pill-count">{publicRooms.length}</span>
            {/if}
          </button>
          
          <button 
            type="button" 
            role="tab"
            aria-selected={activeTab === 'create'}
            class="tab-btn {activeTab === 'create' ? 'active' : ''}"
            on:click={() => activeTab = 'create'}
          >
            <span>Mulai Sesi Baru</span>
          </button>
        </div>

        {#if activeTab === 'classes'}
          <div class="workspace-tools">
            {#if publicRooms.length > 2}
              <div class="search-field-pill">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" class="search-icon">
                  <circle cx="11" cy="11" r="8"></circle>
                  <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                </svg>
                <input 
                  type="text" 
                  bind:value={searchQuery}
                  placeholder="Cari kelas atau topik..."
                  class="search-mini-input"
                />
                {#if searchQuery}
                  <button type="button" class="btn-clear-mini" on:click={() => searchQuery = ''}>✕</button>
                {/if}
              </div>
            {/if}

            <button type="button" class="btn-refresh-text" on:click={fetchActiveRooms} title="Segarkan daftar">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
                <polyline points="23 4 23 10 17 10"></polyline>
                <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
              </svg>
              <span>Segarkan</span>
            </button>
          </div>
        {/if}
      </div>

      <!-- TAB 1: Classes Multi-column Grid -->
      {#if activeTab === 'classes'}
        <div class="classes-container">
          {#if isLoadingRooms}
            <div class="deck-state-loading">
              <span class="loading-spinner"></span>
              <span>Memeriksa sesi kelas aktif di kampus...</span>
            </div>
          {:else if publicRooms.length === 0}
            <div class="deck-empty-state apple-card">
              <div class="empty-icon-circle" aria-hidden="true">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                  <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
                  <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
                </svg>
              </div>
              <h3 class="empty-heading">Belum ada sesi kelas aktif</h3>
              <p class="empty-subtext">
                Daftar ini akan otomatis menampilkan kelas saat dosen atau asisten membuka sesi.
              </p>
              <button type="button" class="btn-apple-secondary btn-deck-cta" on:click={() => activeTab = 'create'}>
                <span>Sebagai dosen? Mulai sesi kelas baru</span>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="9 18 15 12 9 6"></polyline>
                </svg>
              </button>
            </div>
          {:else if filteredRooms.length === 0}
            <div class="deck-empty-state apple-card">
              <h3 class="empty-heading">Kelas tidak ditemukan</h3>
              <p class="empty-subtext">Tidak ada sesi kelas yang cocok dengan kata kunci "{searchQuery}".</p>
              <button type="button" class="btn-apple-secondary btn-deck-cta" on:click={() => searchQuery = ''}>
                Tampilkan semua kelas
              </button>
            </div>
          {:else}
            <!-- Responsive Multi-Column Adaptive Class Grid -->
            <div class="classes-grid">
              {#each paginatedRooms as room (room.id)}
                <div class="class-card apple-card-interactive animate-fade-in">
                  <div class="class-card-top">
                    <div class="class-icon-box" aria-hidden="true">
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
                        <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>
                      </svg>
                    </div>

                    <div class="class-title-block">
                      <h3 class="class-title-text" title={room.name}>{room.name}</h3>
                      {#if isHostRoom(room.id)}
                        <span class="owner-pill">Kelas Anda</span>
                      {/if}
                    </div>
                  </div>

                  <div class="class-card-metrics">
                    <span class="metric-chip">
                      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
                        <circle cx="9" cy="7" r="4"></circle>
                        <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
                        <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
                      </svg>
                      <span>{room.activeUsers} Peserta</span>
                    </span>

                    <span class="metric-chip">
                      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                      </svg>
                      <span>{room.questionCount} Pertanyaan</span>
                    </span>
                  </div>

                  <div class="class-card-bottom">
                    {#if isHostRoom(room.id)}
                      <div class="host-actions-duo">
                        <button 
                          type="button" 
                          class="btn-apple-primary btn-join-card"
                          on:click={() => handleJoin(room.id, 'host')}
                          disabled={isSubmitting}
                          title="Masuk sebagai Host (Yeonly)"
                        >
                          Host
                        </button>
                        <button 
                          type="button" 
                          class="btn-apple-secondary btn-join-secondary"
                          on:click={() => handleJoin(room.id, 'student')}
                          disabled={isSubmitting}
                          title="Masuk sebagai Peserta"
                        >
                          Peserta
                        </button>
                      </div>
                    {:else}
                      <button 
                        type="button" 
                        class="btn-apple-primary btn-join-card full-width"
                        on:click={() => handleJoin(room.id)}
                        disabled={isSubmitting}
                      >
                        <span>Gabung Kelas</span>
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                          <polyline points="9 18 15 12 9 6"></polyline>
                        </svg>
                      </button>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>

            <!-- Pagination Bar -->
            {#if totalPages > 1}
              <div class="deck-pagination">
                <button 
                  type="button" 
                  class="btn-page-step" 
                  disabled={currentPage === 1}
                  on:click={() => currentPage--}
                >
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <polyline points="15 18 9 12 15 6"></polyline>
                  </svg>
                  <span>Sebelumnya</span>
                </button>

                <div class="deck-pagination-dots">
                  {#each Array(totalPages) as _, i}
                    <button 
                      type="button" 
                      class="dot {currentPage === i + 1 ? 'active' : ''}" 
                      on:click={() => currentPage = i + 1}
                      title="Halaman {i + 1}"
                    ></button>
                  {/each}
                  <span class="pagination-page-label">{currentPage} / {totalPages}</span>
                </div>

                <button 
                  type="button" 
                  class="btn-page-step" 
                  disabled={currentPage === totalPages}
                  on:click={() => currentPage++}
                >
                  <span>Berikutnya</span>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <polyline points="9 18 15 12 9 6"></polyline>
                  </svg>
                </button>
              </div>
            {/if}
          {/if}
        </div>

      <!-- TAB 2: Clean, Spacious Create Form -->
      {:else}
        <div class="create-container apple-card">
          <form on:submit|preventDefault={handleCreate} class="create-form-flow">
            <div class="create-field">
              <label for="create-room-name-input" class="create-label">Nama Mata Kuliah atau Topik Praktikum</label>
              <div class="create-input-box">
                <input 
                  id="create-room-name-input"
                  type="text" 
                  bind:value={roomName}
                  placeholder="Cth: Algoritma & Pemrograman, Basis Data Sesi 3"
                  maxlength="60"
                  class="create-text-input"
                  required
                  autocomplete="off"
                />
              </div>
            </div>

            <div class="checkbox-row">
              <label class="custom-checkbox-label">
                <input type="checkbox" bind:checked={isPublic} />
                <span>Tampilkan di daftar kelas aktif agar mahasiswa dapat langsung bergabung dengan satu klik</span>
              </label>
            </div>

            <button 
              type="submit" 
              class="btn-apple-primary btn-submit-create" 
              disabled={isSubmitting || !roomName.trim()}
            >
              <span>{isSubmitting ? 'Memulai Sesi...' : 'Mulai Sesi Kelas Sekarang'}</span>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <line x1="5" y1="12" x2="19" y2="12"></line>
                <polyline points="12 5 19 12 12 19"></polyline>
              </svg>
            </button>

            <div class="create-hint-box">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="16" x2="12" y2="12"></line>
                <line x1="12" y1="8" x2="12.01" y2="8"></line>
              </svg>
              <span>Setelah sesi dibuka, Anda dapat menampilkan QR code layar penuh di proyektor, mengelola pertanyaan langsung dari depan, dan mengunduh rekap saat kelas selesai.</span>
            </div>
          </form>
        </div>
      {/if}
    </section>
  </div>
</div>

<style>
  .landing-page-root {
    position: relative;
    width: 100%;
    min-height: calc(100vh - 80px);
    overflow: hidden;
  }

  /* Subtle Ethereal Watermark Rings (Pure CSS, exact homage to Image 2 reference) */
  .bg-watermark-geometry {
    position: absolute;
    top: -50px;
    right: -100px;
    width: 900px;
    height: 900px;
    pointer-events: none;
    z-index: 0;
    opacity: 0.045;
  }

  .watermark-circle {
    position: absolute;
    border-radius: 50%;
    border: 75px solid #000000;
  }

  .circle-1 {
    width: 680px;
    height: 680px;
    top: 40px;
    right: 20px;
    filter: blur(28px);
  }

  .circle-2 {
    width: 520px;
    height: 520px;
    top: 260px;
    right: 180px;
    filter: blur(22px);
  }

  /* Main Content Layout Flow */
  .landing-content-flow {
    position: relative;
    z-index: 1;
    max-width: 1400px;
    margin: 0 auto;
    padding: 40px clamp(24px, 5vw, 64px) 80px;
    display: flex;
    flex-direction: column;
    gap: 40px;
  }

  /* 1. HERO SPLIT SECTION */
  .hero-split-section {
    display: grid;
    grid-template-columns: 1.15fr 1fr;
    gap: 48px;
    align-items: flex-end;
    padding-bottom: 8px;
  }

  .hero-anchor-left {
    display: flex;
    flex-direction: column;
  }

  .hero-eyebrow {
    font-size: 0.78rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-tertiary);
    margin-bottom: 12px;
  }

  .hero-main-title {
    font-family: var(--font-display);
    font-size: clamp(2.6rem, 4.8vw, 4.2rem);
    font-weight: 800;
    letter-spacing: -0.04em;
    line-height: 1.06;
    color: var(--mono-black);
    margin: 0 0 14px 0;
  }

  .hero-title-accent {
    color: var(--text-secondary);
    font-weight: 700;
  }

  .hero-mini-kicker {
    font-size: 0.95rem;
    color: var(--text-tertiary);
    font-weight: 500;
  }

  .hero-narrative-right {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding-bottom: 6px;
  }

  .hero-brand-block {
    display: flex;
    align-items: center;
  }

  .hero-brand-logo {
    all: unset;
    cursor: pointer;
    font-family: var(--font-display);
    font-size: clamp(1.85rem, 2.8vw, 2.4rem);
    font-weight: 800;
    letter-spacing: -0.04em;
    color: var(--mono-black);
    line-height: 1.05;
    transition: opacity var(--transition-fast);
  }

  .hero-brand-logo:hover {
    opacity: 0.75;
  }

  .hero-brand-logo:focus-visible {
    outline: 2px solid var(--mono-black);
    outline-offset: 4px;
    border-radius: var(--radius-xs);
  }

  .hero-statement {
    font-size: clamp(0.98rem, 1.25vw, 1.12rem);
    color: var(--text-secondary);
    line-height: 1.6;
    max-width: 52ch;
    margin: 0;
  }

  /* 2. DIRECT JOIN BAR */
  .direct-join-section {
    width: 100%;
  }

  .join-bar-card {
    background: var(--bg-surface);
    border: 1px solid var(--border-medium);
    border-radius: var(--radius-xl);
    padding: 16px 22px;
    box-shadow: var(--shadow-sm);
    transition: all var(--transition-fast);
  }

  .join-bar-card:focus-within {
    border-color: var(--border-contrast);
    box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.08);
  }

  .join-bar-form {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    flex-wrap: wrap;
  }

  .join-bar-left {
    display: flex;
    align-items: center;
    gap: 18px;
    flex: 1;
    min-width: 280px;
    flex-wrap: wrap;
  }

  .join-bar-prompt {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--mono-black);
    white-space: nowrap;
  }

  .join-input-group {
    display: flex;
    align-items: center;
    gap: 10px;
    background: var(--bg-subtle);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-pill);
    padding: 6px 16px;
    flex: 1;
    min-width: 200px;
  }

  .code-prefix {
    font-family: var(--font-mono);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    color: var(--text-tertiary);
  }

  .quick-code-field {
    flex: 1;
    border: none;
    background: transparent;
    font-family: var(--font-mono);
    font-size: 1.05rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--mono-black);
    outline: none;
  }

  .quick-code-field::placeholder {
    font-family: var(--font-sans);
    font-size: 0.88rem;
    font-weight: 400;
    letter-spacing: normal;
    text-transform: none;
    color: var(--text-tertiary);
  }

  .btn-join-direct {
    padding: 10px 24px;
    font-size: 0.92rem;
    font-weight: 600;
    white-space: nowrap;
  }

  /* 3. SESSIONS WORKSPACE SECTION */
  .sessions-workspace-section {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .workspace-nav-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .workspace-nav-segmented {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: var(--bg-subtle);
    padding: 4px;
    border-radius: var(--radius-pill);
    border: 1px solid var(--border-subtle);
  }

  .tab-btn {
    padding: 8px 18px;
    border-radius: var(--radius-pill);
    font-size: 0.86rem;
    font-weight: 600;
    color: var(--text-secondary);
    transition: all var(--transition-fast);
    display: inline-flex;
    align-items: center;
    gap: 8px;
    white-space: nowrap;
  }

  .tab-btn.active {
    background: var(--bg-surface);
    color: var(--mono-black);
    box-shadow: var(--shadow-sm);
  }

  .tab-btn:hover:not(.active) {
    color: var(--mono-black);
  }

  .pill-count {
    font-size: 0.72rem;
    font-weight: 700;
    padding: 1px 7px;
    border-radius: var(--radius-pill);
    background: var(--mono-black);
    color: var(--mono-white);
  }

  .workspace-tools {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .search-field-pill {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-pill);
    padding: 6px 14px;
    transition: all var(--transition-fast);
  }

  .search-field-pill:focus-within {
    border-color: var(--border-contrast);
    box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.08);
  }

  .search-icon {
    color: var(--text-tertiary);
    flex-shrink: 0;
  }

  .search-mini-input {
    border: none;
    background: transparent;
    font-size: 0.84rem;
    color: var(--mono-black);
    outline: none;
    width: 160px;
  }

  .btn-clear-mini {
    font-size: 0.75rem;
    color: var(--text-tertiary);
    padding: 0 2px;
  }

  .btn-refresh-text {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-tertiary);
    padding: 6px 12px;
    border-radius: var(--radius-pill);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    transition: all var(--transition-fast);
  }

  .btn-refresh-text:hover {
    color: var(--mono-black);
    border-color: var(--border-medium);
  }

  /* CLASSES MULTI-COLUMN GRID */
  .classes-container {
    width: 100%;
  }

  .deck-state-loading {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 60px 20px;
    color: var(--text-tertiary);
    font-size: 0.9rem;
  }

  .loading-spinner {
    width: 18px;
    height: 18px;
    border: 2px solid var(--border-subtle);
    border-top-color: var(--mono-black);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Empty Deck State */
  .deck-empty-state {
    text-align: center;
    padding: 56px 24px;
    background: var(--bg-surface);
    border: 1px dashed var(--border-medium);
    border-radius: var(--radius-xl);
  }

  .empty-icon-circle {
    width: 52px;
    height: 52px;
    border-radius: 50%;
    background: var(--bg-subtle);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 16px;
  }

  .empty-heading {
    font-size: 1.12rem;
    font-weight: 700;
    color: var(--mono-black);
    margin-bottom: 6px;
  }

  .empty-subtext {
    font-size: 0.9rem;
    color: var(--text-secondary);
    line-height: 1.5;
    max-width: 420px;
    margin: 0 auto 20px;
  }

  .btn-deck-cta {
    padding: 8px 18px;
    font-size: 0.85rem;
  }

  /* Adaptive Multi-column Grid (fills the wide screen effortlessly) */
  .classes-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
    gap: 20px;
  }

  .class-card {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 18px;
    padding: 20px 22px;
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
    transition: all var(--transition-fast);
  }

  .class-card:hover {
    border-color: var(--border-strong);
    box-shadow: var(--shadow-md);
    transform: translateY(-2px);
  }

  .class-card-top {
    display: flex;
    align-items: flex-start;
    gap: 14px;
  }

  .class-icon-box {
    width: 42px;
    height: 42px;
    border-radius: 12px;
    background: var(--bg-subtle);
    color: var(--mono-black);
    border: 1px solid var(--border-subtle);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .class-title-block {
    flex: 1;
    min-width: 0;
  }

  .class-title-text {
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--mono-black);
    line-height: 1.35;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    margin-bottom: 4px;
  }

  .owner-pill {
    display: inline-flex;
    align-items: center;
    font-size: 0.68rem;
    font-weight: 700;
    color: var(--mono-white);
    background: var(--mono-black);
    padding: 1px 7px;
    border-radius: var(--radius-pill);
  }

  .class-card-metrics {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 8px 0;
    border-top: 1px solid var(--border-subtle);
    border-bottom: 1px solid var(--border-subtle);
  }

  .metric-chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-tertiary);
  }

  .metric-chip svg {
    color: var(--text-secondary);
  }

  .class-card-bottom {
    display: flex;
    align-items: center;
  }

  .btn-join-card {
    padding: 8px 18px;
    font-size: 0.86rem;
    font-weight: 600;
  }

  .btn-join-card.full-width {
    width: 100%;
    justify-content: space-between;
  }

  .host-actions-duo {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
  }

  .host-actions-duo button {
    flex: 1;
  }

  .btn-join-secondary {
    padding: 8px 14px;
    font-size: 0.84rem;
    font-weight: 500;
  }

  /* Deck Pagination */
  .deck-pagination {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 24px;
    padding-top: 16px;
    border-top: 1px solid var(--border-subtle);
  }

  .btn-page-step {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 0.82rem;
    font-weight: 600;
    color: var(--text-secondary);
    padding: 6px 14px;
    border-radius: var(--radius-pill);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    transition: all var(--transition-fast);
  }

  .btn-page-step:hover:not(:disabled) {
    color: var(--mono-black);
    background: var(--bg-hover);
  }

  .btn-page-step:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  .deck-pagination-dots {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--border-strong);
    transition: all var(--transition-fast);
    padding: 0;
  }

  .dot.active {
    background: var(--mono-black);
    transform: scale(1.3);
  }

  .pagination-page-label {
    font-family: var(--font-mono);
    font-size: 0.76rem;
    font-weight: 600;
    color: var(--text-tertiary);
    margin-left: 6px;
  }

  /* TAB 2: CREATE SESSION FORM */
  .create-container {
    max-width: 820px;
    margin: 0 auto;
    width: 100%;
    padding: 32px 36px;
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-sm);
  }

  .create-form-flow {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .create-field {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .create-label {
    font-size: 0.88rem;
    font-weight: 600;
    color: var(--mono-black);
  }

  .create-input-box {
    background: var(--bg-subtle);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-pill);
    padding: 12px 20px;
    transition: all var(--transition-fast);
  }

  .create-input-box:focus-within {
    background: var(--bg-surface);
    border-color: var(--border-contrast);
    box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.08);
  }

  .create-text-input {
    width: 100%;
    border: none;
    background: transparent;
    font-size: 0.98rem;
    color: var(--mono-black);
    outline: none;
  }

  .checkbox-row {
    padding: 2px 0;
  }

  .custom-checkbox-label {
    display: inline-flex;
    align-items: flex-start;
    gap: 10px;
    font-size: 0.85rem;
    color: var(--text-secondary);
    cursor: pointer;
    line-height: 1.5;
  }

  .custom-checkbox-label input {
    margin-top: 3px;
    cursor: pointer;
  }

  .btn-submit-create {
    padding: 12px 28px;
    font-size: 0.95rem;
    font-weight: 600;
    width: 100%;
  }

  .create-hint-box {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 14px 16px;
    border-radius: var(--radius-md);
    background: var(--bg-subtle);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    font-size: 0.82rem;
    line-height: 1.5;
  }

  .create-hint-box svg {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--text-tertiary);
  }

  /* RESPONSIVE LAYOUT BREAKPOINTS */
  @media (max-width: 960px) {
    .hero-split-section {
      grid-template-columns: 1fr;
      gap: 24px;
      align-items: flex-start;
    }
    .landing-content-flow {
      padding: 24px 20px 60px;
      gap: 32px;
    }
    .hero-main-title {
      font-size: 2.5rem;
    }
    .classes-grid {
      grid-template-columns: 1fr;
    }
    .create-container {
      padding: 24px 18px;
    }
  }

  @media (max-width: 640px) {
    .join-bar-card {
      padding: 14px 16px;
    }
    .join-bar-form {
      flex-direction: column;
      align-items: stretch;
      gap: 14px;
    }
    .join-bar-left {
      flex-direction: column;
      align-items: stretch;
      gap: 8px;
      width: 100%;
      min-width: unset;
    }
    .btn-join-direct {
      width: 100%;
      justify-content: center;
    }
    .workspace-nav-bar {
      flex-direction: column;
      align-items: stretch;
    }
    .workspace-nav-segmented {
      width: 100%;
    }
    .tab-btn {
      flex: 1;
      justify-content: center;
    }
    .workspace-tools {
      width: 100%;
      justify-content: space-between;
    }
    .search-field-pill {
      flex: 1;
    }
    .search-mini-input {
      width: 100%;
    }
    .deck-pagination {
      flex-direction: column;
      gap: 12px;
    }
  }
</style>
