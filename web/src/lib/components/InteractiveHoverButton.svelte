<script>
  // Svelte adaptation of the dot-fill interaction by Magic UI:
  // https://21st.dev/@dillionverma/components/interactive-hover-button
  /** @type {import('svelte/elements').HTMLButtonAttributes & { label: string, variant?: 'default' | 'hero', size?: 'default' | 'compact', busy?: boolean }} */
  let {
    label,
    type = 'button',
    variant = 'default',
    size = 'default',
    disabled = false,
    busy = false,
    onclick,
    ...rest
  } = $props();
</script>

<button
  {...rest}
  {type}
  {onclick}
  disabled={disabled || busy}
  aria-busy={busy || undefined}
  class="interactive-hover-button"
  class:hero={variant === 'hero'}
  class:compact={size === 'compact'}
>
  <span class="rest-label">{#if busy}<span class="motion-spinner" aria-hidden="true"></span>{/if}{label}</span>
  <span class="hover-label" aria-hidden="true">
    <span>{label}</span>
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M5 12h14m-6-6 6 6-6 6" />
    </svg>
  </span>
</button>

<style>
  .interactive-hover-button {
    --button-fill: var(--text-primary);
    --button-ink: var(--text-inverted);
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    min-height: 44px;
    max-width: 100%;
    overflow: hidden;
    padding: 11px 24px;
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-pill);
    background: var(--bg-surface);
    color: var(--text-primary);
    font-size: 0.9rem;
    font-weight: 600;
    line-height: 1.5;
    white-space: nowrap;
    vertical-align: middle;
    transition: box-shadow var(--transition-normal), transform var(--transition-fast);
  }

  .interactive-hover-button.hero {
    --button-fill: var(--text-primary);
    --button-ink: var(--text-inverted);
    background: #ffffff;
    color: var(--text-primary);
    border-color: rgba(255, 255, 255, 0.65);
    font-weight: 700;
    font-size: 0.92rem;
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.12);
  }

  .interactive-hover-button.compact {
    padding-block: 8px;
    font-size: 0.85rem;
  }

  .interactive-hover-button::before {
    content: '';
    position: absolute;
    top: 50%;
    left: 24px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    z-index: -1;
    background: var(--button-fill);
    transform: translate(-50%, -50%) scale(1);
    transition: transform 300ms var(--ease-apple);
  }

  .rest-label {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding-left: 16px;
    transition: transform 300ms ease, opacity 300ms ease;
  }

  .hover-label {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    color: var(--button-ink);
    opacity: 0;
    transform: translateX(48px);
    transition: transform 300ms ease, opacity 300ms ease;
    pointer-events: none;
  }

  .hover-label svg {
    flex-shrink: 0;
  }

  .hero .rest-label {
    white-space: normal;
  }

  .hero .hover-label {
    padding-inline: 16px;
    white-space: normal;
  }

  .interactive-hover-button:focus-visible {
    outline: 3px solid var(--button-fill);
    outline-offset: 3px;
  }

  .interactive-hover-button.hero:focus-visible {
    outline-color: #ffffff;
  }

  .interactive-hover-button:enabled:focus-visible::before {
    transform: translate(-50%, -50%) scale(180);
  }

  .interactive-hover-button:enabled:focus-visible .rest-label {
    opacity: 0;
    transform: translateX(48px);
  }

  .interactive-hover-button:enabled:focus-visible .hover-label {
    opacity: 1;
    transform: translateX(0);
  }

  @media (hover: hover) and (pointer: fine) {
    .interactive-hover-button:enabled:hover::before {
      transform: translate(-50%, -50%) scale(180);
    }

    .interactive-hover-button:enabled:hover .rest-label {
      opacity: 0;
      transform: translateX(48px);
    }

    .interactive-hover-button:enabled:hover .hover-label {
      opacity: 1;
      transform: translateX(0);
    }
  }

  .interactive-hover-button:disabled {
    cursor: not-allowed;
    opacity: 0.45;
    transform: none;
  }

  @media (prefers-reduced-motion: reduce) {
    .interactive-hover-button,
    .interactive-hover-button::before,
    .rest-label,
    .hover-label {
      transition: none;
    }

    .interactive-hover-button:active {
      transform: none;
    }
  }
</style>
