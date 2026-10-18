<script>
  // Word-by-word reveal inspired by theshanelevine's Streaming Text reference.
  /** @type {string} */
  export let text;
  export let animate = false;
  $: words = text.match(/\S+\s*|\s+/gu) || [];
  $: interval = Math.min(45, 1800 / Math.max(words.length, 1));
</script>

{#if animate}
  <span class="sr-only">{text}</span>
  <span aria-hidden="true">{#each words as word, index}<span class="word" style:animation-delay={`${index * interval}ms`}>{word}</span>{/each}</span>
{:else}
  {text}
{/if}

<style>
  .word { animation: reveal 260ms ease-out both; }
  @keyframes reveal { from { opacity: 0; } to { opacity: 1; } }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; border: 0; }
  @media (prefers-reduced-motion: reduce) { .word { animation: none; } }
</style>
