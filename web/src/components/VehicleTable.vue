<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Vehicle } from '../types'

const props = defineProps<{
  vehicles: Vehicle[]
  isLoading: boolean
  error: string | null
}>()

const search = ref('')
const filteredVehicles = computed(() => {
  const term = search.value.trim().toLowerCase()
  if (!term) return props.vehicles
  return props.vehicles.filter((vehicle) => [vehicle.routeId, vehicle.vehicleId, vehicle.tripId].some((value) => value?.toLowerCase().includes(term)))
})

function formatDelay(seconds: number | null): string {
  if (seconds === null) return '—'
  const minutes = Math.round(seconds / 60)
  return `${minutes > 0 ? '+' : ''}${minutes} min`
}

function delayClass(seconds: number | null): string {
  if (seconds === null || seconds <= 120) return 'delay-on-time'
  if (seconds <= 600) return 'delay-warn'
  return 'delay-late'
}

function formatTime(value: string): string {
  return new Intl.DateTimeFormat('fr-CA', { hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}
</script>

<template>
<section class="vehicles-panel" aria-labelledby="vehicles-title">
  <div class="vehicles-toolbar">
    <h2 id="vehicles-title">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 17.5h12M7 17.5v-7h10v7M9 10.5V7.8A2.8 2.8 0 0 1 11.8 5h.4A2.8 2.8 0 0 1 15 7.8v2.7M8 20h2m4 0h2M8 13h8" /></svg>
      Véhicules en service
    </h2>
    <div class="table-actions">
      <label class="search-field">
        <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="10.8" cy="10.8" r="6.8" /><path d="m16 16 5 5" /></svg>
        <span class="sr-only">Rechercher un véhicule</span>
        <input v-model="search" type="search" placeholder="Rechercher une ligne, un véhicule ou un trajet…" />
      </label>
      <button class="filter-button" type="button" aria-label="Filtres actifs">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M7 12h10m-7 6h4" /></svg>
        Filtres <span class="filter-count">1</span>
      </button>
    </div>
  </div>

  <div v-if="props.isLoading" class="table-state" role="status">Chargement des véhicules…</div>
  <div v-else-if="props.error" class="table-state table-state-error" role="alert">Les positions ne sont pas disponibles pour le moment.</div>
  <div v-else-if="filteredVehicles.length === 0" class="table-state">Aucun autobus ne correspond à cette recherche.</div>
  <div v-else class="table-scroll">
    <table>
      <thead><tr><th>Ligne</th><th>Véhicule</th><th>Trajet</th><th>Retard</th><th>Dernière position</th><th>Heure</th><th>Statut</th></tr></thead>
      <tbody>
        <tr v-for="vehicle in filteredVehicles" :key="vehicle.vehicleId">
          <td><span class="route-number">{{ vehicle.routeId ?? '—' }}</span></td>
          <td class="strong-cell">{{ vehicle.vehicleId }}</td>
          <td class="trip-cell">{{ vehicle.tripId ?? 'Trajet non identifié' }}</td>
          <td><span :class="['delay-value', delayClass(vehicle.delaySeconds)]">{{ formatDelay(vehicle.delaySeconds) }}</span></td>
          <td class="position-cell">{{ vehicle.latitude.toFixed(4) }}, {{ vehicle.longitude.toFixed(4) }}</td>
          <td>{{ formatTime(vehicle.recordedAt) }}</td>
          <td><span class="service-status"><span class="service-dot"></span>En service</span></td>
        </tr>
      </tbody>
    </table>
  </div>
  <div class="table-footer">
    <span>Affichage de {{ filteredVehicles.length }} sur {{ props.vehicles.length }} véhicules en service</span>
    <nav class="pagination" aria-label="Pagination des véhicules"><button type="button" disabled aria-label="Page précédente">‹</button><button type="button" class="current-page">1</button><button type="button">2</button><button type="button">3</button><button type="button" aria-label="Page suivante">›</button></nav>
  </div>
</section>
</template>
