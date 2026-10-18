<script>
  import { onMount } from 'svelte';

  onMount(() => {
    // Ensure light monochrome mode is consistently applied
    document.documentElement.classList.remove('dark');
    try { localStorage.removeItem('yeonly_theme'); } catch {}
  });
</script>

