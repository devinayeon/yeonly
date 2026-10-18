<script>
  import { onDestroy } from 'svelte';
  import { draftKey, readDraft, saveDraft } from '../stores/drafts.js';
  import { roomStore } from '../stores/roomStore.js';
  import { ws } from '../stores/wsStore.js';
  import { toast } from '../stores/toastStore.js';

  const key = draftKey($roomStore.roomId, $roomStore.isHost);
  let text = readDraft(key);
  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let typingTimer;
  function stopTyping() { clearTimeout(typingTimer); ws.setTyping(false); }
  function handleInput() {
    saveDraft(key, text);
    clearTimeout(typingTimer);
    ws.setTyping(Boolean(text.trim()));
    typingTimer = setTimeout(stopTyping, 2500);
  }
  onDestroy(() => { saveDraft(key, text); stopTyping(); });
  let isSubmitting = false;

  const MAX_LEN = 300;
  const MIN_LEN = 5;

  async function handleSubmit() {
    if (isSubmitting) return;
    const trimmed = text.trim();
    if (trimmed.length < MIN_LEN) {
      toast.warning(`Pertanyaan minimal ${MIN_LEN} karakter.`);
      return;
    }
    if (trimmed.length > MAX_LEN) {
      toast.warning(`Pertanyaan maksimal ${MAX_LEN} karakter.`);
      return;
    }
    if ($roomStore.isFrozen) {
      toast.warning('Sesi pertanyaan saat ini sedang dijeda oleh asisten.');
      return;
    }

    stopTyping();
    isSubmitting = true;
    try {
      const accepted = await ws.submitQuestion(trimmed);
      if (accepted) {
        text = '';
        saveDraft(key, '');
        toast.success('Pertanyaan terkirim');
      }
    } finally {
      isSubmitting = false;
    }
  }

  /** @param {KeyboardEvent} e */
  function handleKeyDown(e) {
    if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
      e.preventDefault();
      handleSubmit();
    }
  }

  $: charCount = text.length;
</script>

<!-- Bottom Pinned Chat Dock (Apple HIG Style) -->
<div class="chat-input-dock">
  <div class="dock-container apple-card {$roomStore.isFrozen ? 'frozen' : ''}">
    {#if $roomStore.isFrozen}
      <div class="frozen-overlay">
        <span>Sesi pertanyaan sedang dijeda oleh Asisten Praktikum</span>
      </div>
    {/if}

    <!-- Top Metadata Strip -->
    <div class="dock-meta-row">
      <div class="sender-identity-pill {$roomStore.isHost ? 'host' : ''}">
        {#if $roomStore.isHost}
          <span class="sender-name"><strong>Yeonly</strong> · Asisten Praktikum</span>
        {:else}
          <span class="sender-name">Anonim: <strong>{$roomStore.userAlias || 'Mahasiswa'}</strong></span>
        {/if}
      </div>
    </div>

    <!-- Main Message Input Line -->
    <div class="dock-input-row">
      <textarea
        bind:value={text}
        on:keydown={handleKeyDown}
        on:input={handleInput}
        on:blur={stopTyping}
        aria-label={$roomStore.isHost ? "Pesan Yeonly" : "Pertanyaan anonim"}
        maxlength={MAX_LEN}
        disabled={$roomStore.isFrozen || isSubmitting}
        placeholder={$roomStore.isHost ? "Kirim pengumuman atau instruksi sebagai Yeonly... Tekan Enter untuk mengirim." : "Tulis pertanyaanmu secara anonim... Tekan Enter untuk mengirim."}
        rows="2"
        class="chat-textarea"
      ></textarea>

      <button 
        type="button" 
        class="btn-send-message"
        on:click={handleSubmit}
        aria-busy={isSubmitting}
        aria-label={isSubmitting ? "Mengirim pesan" : "Kirim pesan"}
        disabled={!$ws.connected || $roomStore.isFrozen || isSubmitting || text.trim().length < MIN_LEN}
        title="Kirim Pertanyaan (Enter)"
      >
        {#if isSubmitting}<span class="motion-spinner" aria-hidden="true"></span>{:else}
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3">
          <line x1="22" y1="2" x2="11" y2="13"></line>
          <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
        </svg>
        {/if}
      </button>
    </div>

    <!-- Character limit caption -->
    <div class="dock-bottom-meta">
      <span class="caption-text">{$roomStore.isHost ? "Nama Yeonly terlihat oleh peserta" : "Nama anonim • Tanpa akun"}</span>
      <span class="char-count {charCount >= MAX_LEN ? 'limit' : ''}">
        {charCount} / {MAX_LEN}
      </span>
    </div>
  </div>
</div>

<style>
  .chat-input-dock {
    position: sticky;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 40;
    padding: 12px 18px 18px;
    background: linear-gradient(to top, var(--bg-app) 75%, transparent);
  }

  .dock-container {
    position: relative;
    max-width: 820px;
    margin: 0 auto;
    padding: 12px 16px 10px;
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-lg);
    background: rgba(255, 255, 255, 0.94);
    backdrop-filter: saturate(180%) blur(20px);
    -webkit-backdrop-filter: saturate(180%) blur(20px);
    border: 1px solid var(--border-subtle);
  }

  .dock-container.frozen {
    opacity: 0.75;
  }

  .frozen-overlay {
    position: absolute;
    inset: 0;
    background: rgba(255, 255, 255, 0.92);
    backdrop-filter: blur(4px);
    border-radius: var(--radius-xl);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.88rem;
    font-weight: 600;
    color: var(--text-primary);
    z-index: 10;
  }

  /* Meta Row */
  .dock-meta-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    margin-bottom: 8px;
    flex-wrap: wrap;
  }

  .sender-identity-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 0.76rem;
    color: var(--text-secondary);
    background: var(--bg-subtle);
    padding: 3px 10px;
    border-radius: var(--radius-pill);
    border: 1px solid var(--border-subtle);
  }

  .sender-identity-pill.host {
    background: var(--mono-black);
    color: var(--mono-white);
    border-color: var(--mono-black);
  }

  .sender-identity-pill strong {
    color: inherit;
    font-weight: 700;
  }

  /* Input Row */
  .dock-input-row {
    display: flex;
    align-items: flex-end;
    gap: 10px;
  }

  .chat-textarea {
    flex: 1;
    border: 1px solid var(--border-subtle);
    background: var(--bg-subtle);
    border-radius: var(--radius-md);
    padding: 10px 14px;
    font-size: 0.92rem;
    line-height: 1.45;
    color: var(--text-primary);
    outline: none;
    resize: none;
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .chat-textarea:focus {
    background: var(--bg-surface);
    border-color: var(--border-contrast);
    box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.08);
  }

  .btn-send-message {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    background: var(--mono-black);
    color: var(--mono-white);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  }

  .btn-send-message:hover:not(:disabled) {
    background: var(--mono-black-hover);
    transform: scale(1.04);
  }

  .btn-send-message:disabled {
    opacity: 0.35;
    cursor: not-allowed;
    background: var(--bg-hover);
    color: var(--text-tertiary);
    box-shadow: none;
  }

  /* Bottom Meta */
  .dock-bottom-meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 6px;
    font-size: 0.72rem;
    color: var(--text-tertiary);
  }

  .char-count {
    font-family: var(--font-mono);
  }

  .char-count.limit {
    color: var(--semantic-red);
    font-weight: 700;
  }

  @media (max-width: 600px) {
    .chat-input-dock {
      padding: 8px 12px 12px;
    }
    .caption-text {
      display: none;
    }
  }
</style>
