<script>
  import { roomStore } from '../stores/roomStore.js';
  import { ws } from '../stores/wsStore.js';
  import { toast } from '../stores/toastStore.js';
  import StreamingText from './StreamingText.svelte';

  /** @type {import('../types').Question} */
  export let question;
  const animateText = roomStore.takeQuestionAnimation(question.id);

  function handleUpvote() {
    ws.upvoteQuestion(question.id);
  }

  function handlePin() {
    ws.hostPin(question.id);
    toast.info('Status: Sedang Dibahas');
  }

  function handleResolve() {
    ws.hostResolve(question.id);
    toast.success('Pertanyaan selesai dijawab');
  }

  function handleDelete() {
    if (confirm('Hapus pertanyaan ini?')) {
      ws.hostDelete(question.id);
      toast.warning('Pertanyaan dihapus');
    }
  }

  /** @param {string} isoString */
  function formatTime(isoString) {
    if (!isoString) return '';
    const date = new Date(isoString);
    return date.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
  }

  /** @param {string} name */
  function getAvatarLetter(name) {
    if (!name) return 'A';
    const words = name.trim().split(/\s+/u);
    if (words.length > 1) {
      return (words[0][0] + words[1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  }

  $: hasVoted = $roomStore.votedQuestionIds.includes(question.id);
  $: isMyMessage = question.authorId === $roomStore.sessionId;
</script>

<div class="chat-bubble-row {isMyMessage ? 'is-self' : ''} {question.status}">
  <!-- Monochrome Avatar Circle -->
  <div class="chat-avatar-wrap">
    <div class="avatar-circle">
      <span>{getAvatarLetter(question.authorAlias)}</span>
    </div>
  </div>

  <!-- Bubble Content -->
  <div class="chat-bubble-main">
    <!-- Sender & Timestamp Row -->
    <div class="bubble-header">
      <span class="bubble-author">
        {question.authorAlias || 'Mahasiswa Anonim'}
        {#if question.authorRole === 'host'}
          <span class="yeonly-role-badge">Asisten Praktikum · Host</span>
        {/if}
      </span>
      <span class="bubble-meta">
        <span class="bubble-time">{formatTime(question.createdAt)}</span>
        <span class="bubble-via">• {question.authorRole === 'host' ? 'Asisten Praktikum' : 'Anonim'}</span>
      </span>

      {#if isMyMessage}
        <span class="self-badge">Anda</span>
      {/if}
    </div>

    <!-- Pinned / Answered Status Banner -->
    {#if question.status === 'pinned'}
      <div class="system-status-chip pinned">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="17" x2="12" y2="22"></line>
          <path d="M5 17h14v-1.76a2 2 0 0 0-1.11-1.79l-1.78-.89A2 2 0 0 1 15 10.76V6a3 3 0 0 0-6 0v4.76a2 2 0 0 1-1.11 1.79l-1.78.89A2 2 0 0 0 5 15.24z"></path>
        </svg>
        <span>Sedang dibahas di depan kelas</span>
      </div>
    {:else if question.status === 'answered'}
      <div class="system-status-chip answered">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
        <span>Selesai dijawab</span>
      </div>
    {/if}

    <!-- Question Message Text Container -->
    <div class="bubble-content-box apple-card">
      <p class="bubble-text">
        <StreamingText text={question.text} animate={animateText} />
      </p>

      <div class="bubble-footer">
        <!-- Upvote Action Chip (Monochrome Inversion) -->
        <button 
          type="button" 
          class="upvote-chip {hasVoted ? 'voted' : ''}" 
          on:click={handleUpvote}
          aria-pressed={hasVoted}
          title={hasVoted ? 'Batalkan Upvote' : 'Upvote pertanyaan ini'}
        >
          <span class="arrow" aria-hidden="true">▲</span>
          <span class="vote-count">{question.upvotes || 0}</span>
          <span class="vote-label">Upvote</span>
        </button>

        <!-- Host Moderation Controls -->
        {#if $roomStore.isHost}
          <div class="host-inline-controls">
            {#if question.status !== 'pinned'}
              <button type="button" class="btn-host-action" on:click={handlePin} title="Tandai sedang dibahas">
                Bahas
              </button>
            {/if}
            {#if question.status !== 'answered'}
              <button type="button" class="btn-host-action" on:click={handleResolve} title="Tandai selesai dijawab">
                Selesai
              </button>
            {/if}
            <button type="button" class="btn-host-action danger" on:click={handleDelete} title="Hapus pertanyaan">
              Hapus
            </button>
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .chat-bubble-row {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px;
    max-width: 85%;
  }

  .chat-bubble-row.is-self {
    margin-left: auto;
    flex-direction: row-reverse;
  }

  /* Avatar Circle */
  .chat-avatar-wrap {
    flex-shrink: 0;
    margin-top: 2px;
  }

  .avatar-circle {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    color: var(--text-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.8rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    box-shadow: var(--shadow-sm);
  }

  .is-self .avatar-circle {
    background: var(--mono-black);
    color: var(--mono-white);
    border-color: var(--mono-black);
  }

  /* Bubble Main */
  .chat-bubble-main {
    flex: 1;
    min-width: 0;
  }

  .bubble-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 5px;
    flex-wrap: wrap;
  }

  .is-self .bubble-header {
    justify-content: flex-end;
  }

  .bubble-author {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .yeonly-role-badge {
    font-size: 0.68rem;
    font-weight: 700;
    padding: 1px 7px;
    border-radius: var(--radius-pill);
    background: var(--mono-black);
    color: var(--mono-white);
    letter-spacing: 0.01em;
  }

  .bubble-meta {
    font-size: 0.74rem;
    color: var(--text-tertiary);
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .self-badge {
    font-size: 0.66rem;
    font-weight: 600;
    padding: 1px 7px;
    border-radius: var(--radius-pill);
    background: var(--bg-subtle);
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
  }

  /* Status Chips */
  .system-status-chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 3px 10px;
    border-radius: var(--radius-pill);
    font-size: 0.72rem;
    font-weight: 600;
    margin-bottom: 6px;
  }

  .system-status-chip.pinned {
    background: var(--mono-black);
    color: var(--mono-white);
  }

  .system-status-chip.answered {
    background: var(--bg-subtle);
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
  }

  /* Message Content Box */
  .bubble-content-box {
    padding: 14px 16px;
    border-radius: var(--radius-lg);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    box-shadow: var(--shadow-sm);
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .pinned .bubble-content-box {
    border-color: var(--border-contrast);
    box-shadow: var(--shadow-md);
  }

  .answered .bubble-content-box {
    opacity: 0.75;
  }

  .is-self .bubble-content-box {
    background: var(--bg-surface);
    border-color: var(--border-medium);
  }

  .bubble-text {
    font-size: 0.94rem;
    line-height: 1.55;
    color: var(--text-primary);
    white-space: pre-wrap;
    word-break: break-word;
    margin-bottom: 10px;
  }

  /* Bubble Footer */
  .bubble-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    flex-wrap: wrap;
    padding-top: 6px;
    border-top: 1px solid var(--border-subtle);
  }

  /* Upvote Chip (Monochrome) */
  .upvote-chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 11px;
    border-radius: var(--radius-pill);
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--text-secondary);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .upvote-chip:hover {
    color: var(--text-primary);
    border-color: var(--border-strong);
    background: var(--bg-hover);
  }

  .upvote-chip.voted {
    background: var(--mono-black);
    border-color: var(--mono-black);
    color: var(--mono-white);
  }

  .upvote-chip .arrow {
    font-size: 0.68rem;
    line-height: 1;
  }

  .upvote-chip .vote-count {
    font-family: var(--font-mono);
    font-weight: 700;
  }

  .upvote-chip .vote-label {
    font-size: 0.72rem;
    font-weight: 500;
  }

  /* Host Actions */
  .host-inline-controls {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .btn-host-action {
    font-size: 0.72rem;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: var(--radius-pill);
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    transition: color var(--transition-fast), background-color var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  }

  .btn-host-action:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
    border-color: var(--border-strong);
  }

  .btn-host-action.danger {
    color: var(--text-tertiary);
  }

  .btn-host-action.danger:hover {
    color: var(--semantic-red);
    background: var(--semantic-red-subtle);
    border-color: rgba(215, 0, 21, 0.2);
  }

  @media (max-width: 600px) {
    .chat-bubble-row {
      max-width: 95%;
    }
  }
</style>
