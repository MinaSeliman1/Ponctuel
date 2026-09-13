<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchDashboard, fetchErrorSummary, fetchVehicles } from './api/client'
import ErrorQualityChart from './components/ErrorQualityChart.vue'
import StatusPanel from './components/StatusPanel.vue'
import VehicleMap from './components/VehicleMap.vue'
import VehicleTable from './components/VehicleTable.vue'
import type { Dashboard, ErrorSummary, Vehicle } from './types'

const dashboard = ref<Dashboard | null>(null)
const vehicles = ref<Vehicle[]>([])
const errorSummaries = ref<ErrorSummary[]>([])
const isLoading = ref(true)
const error = ref<string | null>(null)
const qualityError = ref(false)
const isRefreshing = ref(false)
const preferencesOpen = ref(false)
const autoRefreshEnabled = ref(true)
let controller: AbortController | null = null
let refreshTimer: number | undefined
const refreshIntervalMs = 30_000

const modeLabel = computed(() => dashboard.value?.mode.toLowerCase() === 'stm' ? 'Temps réel' : 'Fixture local')

async function loadData() {
  if (controller !== null && (isLoading.value || isRefreshing.value)) return

  const hasExistingData = dashboard.value !== null || vehicles.value.length > 0 || errorSummaries.value.length > 0
  controller?.abort()
  const currentController = new AbortController()
  controller = currentController
  if (hasExistingData) {
    isRefreshing.value = true
  } else {
    isLoading.value = true
  }
  error.value = null
  qualityError.value = false
  errorSummaries.value = []

  const [dashboardResult, vehiclesResult, qualityResult] = await Promise.allSettled([
    fetchDashboard(currentController.signal),
    fetchVehicles(500, currentController.signal),
    fetchErrorSummary(100, currentController.signal),
  ])

  if (dashboardResult.status === 'fulfilled') {
    dashboard.value = dashboardResult.value
  } else if (!(dashboardResult.reason instanceof DOMException && dashboardResult.reason.name === 'AbortError')) {
    error.value = 'unavailable'
  }

  if (vehiclesResult.status === 'fulfilled') {
    vehicles.value = vehiclesResult.value
  } else if (!(vehiclesResult.reason instanceof DOMException && vehiclesResult.reason.name === 'AbortError')) {
    error.value = 'unavailable'
  }

  if (qualityResult.status === 'fulfilled') {
    errorSummaries.value = qualityResult.value
  } else if (!(qualityResult.reason instanceof DOMException && qualityResult.reason.name === 'AbortError')) {
    qualityError.value = true
  }

  if (controller === currentController) {
    isLoading.value = false
    isRefreshing.value = false
  }
}

function stopRefreshTimer(): void {
  if (refreshTimer !== undefined) {
    window.clearInterval(refreshTimer)
    refreshTimer = undefined
  }
}

function scheduleRefresh(): void {
  stopRefreshTimer()
  if (autoRefreshEnabled.value) {
    refreshTimer = window.setInterval(() => void loadData(), refreshIntervalMs)
  }
}

watch(autoRefreshEnabled, scheduleRefresh)

onMounted(() => {
  void loadData()
  scheduleRefresh()
})
onUnmounted(() => {
  controller?.abort()
  stopRefreshTimer()
})
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="brand-lockup"><span class="brand-name">Ponctuel</span><span class="brand-divider" aria-hidden="true"></span><span class="brand-context">Réseau STM</span></div>
      <div class="topbar-actions">
        <span class="live-clock">Suivi des positions</span>
        <div class="preferences-wrap">
          <button class="icon-button" type="button" aria-label="Préférences" aria-controls="preferences-panel" :aria-expanded="preferencesOpen" aria-haspopup="dialog" @click="preferencesOpen = !preferencesOpen">
            <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3" /><path d="M12 2v3m0 14v3M2 12h3m14 0h3m-4.9-7.1-2.1 2.1m-10 10-2.1 2.1m0-14.2 2.1 2.1m10 10 2.1 2.1" /></svg>
          </button>
          <div v-if="preferencesOpen" id="preferences-panel" class="preferences-panel" role="dialog" aria-labelledby="preferences-title">
            <h2 id="preferences-title">Préférences</h2>
            <label class="preference-toggle">
              <input v-model="autoRefreshEnabled" type="checkbox" />
              <span>Actualisation automatique</span>
            </label>
            <p v-if="autoRefreshEnabled">Les données sont rafraîchies toutes les 30 secondes.</p>
            <p v-else>Actualisation automatique suspendue. Le bouton Actualiser reste disponible.</p>
            <button class="text-button" type="button" @click="preferencesOpen = false">Fermer</button>
          </div>
        </div>
      </div>
    </header>
    <main class="main-content">
      <StatusPanel :dashboard="dashboard" :is-loading="isLoading" :is-refreshing="isRefreshing" :error="error" @refresh="loadData" />
      <div class="content-heading"><div><span class="eyebrow">Réseau en direct</span><h1>Les autobus, au bon moment.</h1></div><span class="mode-summary"><span class="mode-dot"></span>{{ modeLabel }}</span></div>
      <VehicleMap :vehicles="vehicles" :is-loading="isLoading" />
      <ErrorQualityChart :summaries="errorSummaries" :is-loading="isLoading" :has-error="qualityError" :stale="dashboard?.stale ?? false" />
      <VehicleTable :vehicles="vehicles" :is-loading="isLoading" :error="error" />
    </main>
    <footer class="app-footer"><span>Données STM · Visualisation Ponctuel</span><a href="https://www.stm.info/fr/a-propos/developpeurs" target="_blank" rel="noreferrer">Source et conditions d’utilisation</a></footer>
  </div>
</template>
