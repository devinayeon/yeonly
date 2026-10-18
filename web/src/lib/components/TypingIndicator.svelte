<script>
  import { reveal } from '../utils/motion.js';
  import { roomStore } from '../stores/roomStore.js';
  $: users = $roomStore.typingUsers;
  $: names = users.slice(0, 2).map(user => user.alias || 'Seseorang').join(' dan ');
</script>

<div class="typing-status" role="status" aria-live="polite" aria-atomic="true">
  {#if users.length > 0}
    <div class="typing-bubble" transition:reveal={{ duration: 140, y: 3 }}>
      <span class="dots" aria-hidden="true"><span></span><span></span><span></span></span>
      <span class="typing-label">{names}{users.length > 2 ? ` dan ${users.length - 2} lainnya` : ''} sedang mengetik<span aria-hidden="true">…</span></span>
    </div>
  {/if}
</div>

<style>
  .typing-status { 
    min-height: 42px; 
    margin: -6px 0 14px; 
  }
  .typing-bubble { 
    display: inline-flex; 
    align-items: center; 
    gap: 10px; 
    max-width: 100%; 
    padding: 8px 14px; 
    border-radius: 16px 16px 16px 4px; 
    background: var(--bg-surface); 
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary); 
    box-shadow: var(--shadow-sm);
  }
  .typing-label { 
    font-size: 0.76rem; 
    line-height: 1.5; 
    overflow-wrap: anywhere; 
  }
  .dots { 
    display: flex; 
    align-items: center; 
    gap: 4px; 
    height: 16px; 
    flex-shrink: 0; 
    color: var(--text-primary); 
  }
  .dots span { 
    display: block; 
    width: 5px; 
    height: 5px; 
    border-radius: 50%; 
    background: currentColor; 
    animation: pulse 1s ease-in-out infinite; 
  }
  .dots span:nth-child(2) { animation-delay: 150ms; }
  .dots span:nth-child(3) { animation-delay: 300ms; }
  @keyframes pulse { 0%, 80%, 100% { opacity: 0.3; transform: scale(0.85); } 40% { opacity: 1; transform: scale(1.15); } }
  @media (prefers-reduced-motion: reduce) { .dots span { animation: none; opacity: 0.7; } }
</style>
