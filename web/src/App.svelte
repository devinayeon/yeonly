<script>
  import { onMount } from 'svelte';
  import { roomStore } from './lib/stores/roomStore.js';
  import { ws } from './lib/stores/wsStore.js';
  import { restoreLocation } from './lib/stores/navigation.js';
  import Navbar from './lib/components/Navbar.svelte';
  import LandingView from './lib/components/LandingView.svelte';
  import RoomView from './lib/components/RoomView.svelte';
  import Toast from './lib/components/Toast.svelte';
  import ProjectorModal from './lib/components/ProjectorModal.svelte';

  onMount(() => {
    // Clear any previous dark mode settings
    document.documentElement.classList.remove('dark');
    try { localStorage.removeItem('yeonly_theme'); } catch {}

    roomStore.initSession();

    restoreLocation();
    window.addEventListener('popstate', restoreLocation);
    return () => {
      window.removeEventListener('popstate', restoreLocation);
      ws.disconnect();
    };
  });
</script>

<div class="app-layout">
  <Navbar />

  <main class="main-content">
    {#if $roomStore.currentView === 'room'}
      <RoomView />
    {:else}
      <LandingView />
    {/if}
  </main>

  <!-- Projector Modal overlay -->
  {#if $roomStore.isProjectorOpen}
    <ProjectorModal />
  {/if}

  <!-- Floating Toast Notifications -->
  <Toast />
</div>

<style>
  .app-layout {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    background-color: var(--bg-app);
  }

  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
</style>
