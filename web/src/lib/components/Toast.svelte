<script>
  import { reveal } from '../utils/motion.js';
  import { toast } from '../stores/toastStore.js';
</script>

<div class="toast-container" aria-live="polite">
  {#each $toast as item (item.id)}
    <div class="toast-card {item.type}" transition:reveal={{ y: 8 }}>
      <div class="toast-icon">
        {#if item.type === 'success'}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
        {:else if item.type === 'error'}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="8" x2="12" y2="12"></line>
            <line x1="12" y1="16" x2="12.01" y2="16"></line>
          </svg>
        {:else if item.type === 'warning'}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
            <line x1="12" y1="9" x2="12" y2="13"></line>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
        {:else}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="16" x2="12" y2="12"></line>
            <line x1="12" y1="8" x2="12.01" y2="8"></line>
          </svg>
        {/if}
      </div>
      <div class="toast-content">{item.message}</div>
      <button class="toast-close" on:click={() => toast.remove(item.id)} title="Tutup pemberitahuan">×</button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: 24px;
    right: 24px;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: 380px;
    width: calc(100vw - 48px);
    pointer-events: none;
  }

  .toast-card {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: 12px;
    background: rgba(29, 29, 31, 0.94);
    backdrop-filter: saturate(180%) blur(20px);
    -webkit-backdrop-filter: saturate(180%) blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.16);
    border-radius: var(--radius-pill);
    padding: 10px 18px;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.25);
    color: #ffffff;

  }


  .toast-icon {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    color: #ffffff;
  }







  .toast-content {
    flex: 1;
    font-size: 0.85rem;
    line-height: 1.4;
    color: #ffffff;
    font-weight: 500;
  }

  .toast-close {
    color: rgba(255, 255, 255, 0.6);
    font-size: 1.1rem;
    line-height: 1;
    padding: 0 4px;
    transition: color var(--transition-fast);
  }

  .toast-close:hover {
    color: #ffffff;
  }
</style>
