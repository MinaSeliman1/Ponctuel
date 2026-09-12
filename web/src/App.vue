<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { fetchDashboard, fetchVehicles } from './api/client'
import StatusPanel from './components/StatusPanel.vue'
import VehicleMap from './components/VehicleMap.vue'
import VehicleTable from './components/VehicleTable.vue'
import type { Dashboard, Vehicle } from './types'

const dashboard = ref<Dashboard | null>(null)
const vehicles = ref<Vehicle[]>([])
const isLoading = ref(true)
const error = ref<string | null>(null)
let controller: AbortController | null = null

const modeLabel = computed(() => dashboard.value?.mode.toLowerCase() === 'stm' ? 'Temps réel' : 'Fixture local')

async function loadData() {
  controller?.abort()
  controller = new AbortController()
  isLoading.value = true
  error.value = null
  try {
    const [nextDashboard, nextVehicles] = await Promise.all([
      fetchDashboard(controller.signal),
      fetchVehicles(500, controller.signal),
    ])
    dashboard.value = nextDashboard
    vehicles.value = nextVehicles
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') return
    error.value = 'unavailable'
  } finally {
    isLoading.value = false
  }
}

onMounted(loadData)
onUnmounted(() => controller?.abort())
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="brand-lockup"><span class="brand-name">Ponctuel</span><span class="brand-divider" aria-hidden="true"></span><span class="brand-context">Réseau STM</span></div>
      <div class="topbar-actions"><span class="live-clock">Suivi des positions</span><button class="icon-button" type="button" aria-label="Préférences"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-1.8 1.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5v.1h-2.6v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.9.3l-.1.1-1.8-1.8.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H6.4v-2.6h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.9l-.1-.1 1.8-1.8.1.1a1.7 1.7 0 0 0 1.9.3 1.7 1.7 0 0 0 1-1.5V4.4H15v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.9-.3l.1-.1 1.8 1.8-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.5 1h.1V13h-.1a1.7 1.7 0 0 0-1.5 1Z" /></svg></button></div>
    </header>
    <main class="main-content">
      <StatusPanel :dashboard="dashboard" :is-loading="isLoading" :error="error" @refresh="loadData" />
      <div class="content-heading"><div><span class="eyebrow">Réseau en direct</span><h1>Les autobus, au bon moment.</h1></div><span class="mode-summary"><span class="mode-dot"></span>{{ modeLabel }}</span></div>
      <VehicleMap :vehicles="vehicles" :is-loading="isLoading" />
      <VehicleTable :vehicles="vehicles" :is-loading="isLoading" :error="error" />
    </main>
    <footer class="app-footer"><span>Données STM · Visualisation Ponctuel</span><a href="https://www.stm.info/fr/a-propos/developpeurs" target="_blank" rel="noreferrer">Source et conditions d’utilisation</a></footer>
  </div>
</template>
