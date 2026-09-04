<template>
  <div class="app-root">
    <div v-if="standaloneViteWithoutWails" class="vite-only-banner" role="status">
      Сейчас открыт только dev-сервер Vite — вызовы Go недоступны. Для терминала с бэкендом запустите
      <code>wails dev</code> и откройте в браузере адрес из его лога (часто
      <a href="http://localhost:34115/" target="_blank" rel="noopener noreferrer">http://localhost:34115</a>).
    </div>
    <Dashboard />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Dashboard from './views/Dashboard.vue'

/** true в `npm run dev` без Wails: нет window.go, биндинги к Go не работают */
const standaloneViteWithoutWails = computed(() => {
  if (!import.meta.env.DEV) {
    return false
  }
  if (typeof window === 'undefined') {
    return false
  }
  return !(window as unknown as { go?: unknown }).go
})
</script>

<style scoped>
.app-root {
  min-height: 100vh;
}

.vite-only-banner {
  position: sticky;
  top: 0;
  z-index: 10000;
  padding: 10px 14px;
  text-align: center;
  font-size: 14px;
  line-height: 1.45;
  color: #1a1a1a;
  background: #f5c542;
  border-bottom: 1px solid #c9a008;
}

.vite-only-banner code {
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.08);
  font-size: 0.92em;
}

.vite-only-banner a {
  color: #0b3d6d;
  font-weight: 600;
}
</style>