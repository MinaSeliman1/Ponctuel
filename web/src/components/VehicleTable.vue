<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Vehicle } from '../types'
import { downloadCsv, vehiclesToCsv } from '../utils/csv'
import { copyVehicleFiltersLink, readVehicleFilters, syncVehicleFiltersToUrl, type DelayFilter, type VehicleSort } from '../utils/vehicleFilters'

const props = defineProps<{
  vehicles: Vehicle[]
  isLoading: boolean
  error: string | null
}>()

const initialFilters = readVehicleFilters()
const search = ref(initialFilters.search)
const filtersOpen = ref(false)
const filtersTrigger = ref<HTMLButtonElement | null>(null)
const filtersPanel = ref<HTMLDivElement | null>(null)
const selectedRoute = ref(initialFilters.route)
const selectedDelay = ref<DelayFilter>(initialFilters.delay)
const currentPage = ref(1)
const pageSize = 10
const copyStatus = ref<'idle' | 'success' | 'error'>('idle')
const sortBy = ref<VehicleSort>(initialFilters.sort ?? 'arrival')

const routeOptions = computed(() => Array.from(new Set(
  props.vehicles
    .map((vehicle) => vehicle.routeId)
    .filter((route): route is string => Boolean(route)),
)).sort((left, right) => left.localeCompare(right, 'fr', { numeric: true })))

const filteredVehicles = computed(() => {
  const term = search.value.trim().toLowerCase()
  const filtered = props.vehicles.filter((vehicle) => {
    const matchesSearch = !term || [vehicle.routeId, vehicle.vehicleId, vehicle.tripId]
      .some((value) => value?.toLowerCase().includes(term))
    const matchesRoute = !selectedRoute.value || vehicle.routeId === selectedRoute.value
    const matchesDelay = selectedDelay.value === 'all'
      || (selectedDelay.value === 'late' && vehicle.delaySeconds !== null && vehicle.delaySeconds > 120)
      || (selectedDelay.value === 'on-time' && (vehicle.delaySeconds === null || vehicle.delaySeconds <= 120))
    return matchesSearch && matchesRoute && matchesDelay
  })

  if (sortBy.value === 'arrival') return filtered

  return [...filtered].sort((left, right) => {
    if (sortBy.value === 'delay') {
      if (left.delaySeconds === null && right.delaySeconds === null) return 0
      if (left.delaySeconds === null) return 1
      if (right.delaySeconds === null) return -1
      return right.delaySeconds - left.delaySeconds
    }

    if (sortBy.value === 'route') return compareNullableText(left.routeId, right.routeId)
    return left.vehicleId.localeCompare(right.vehicleId, 'fr', { numeric: true })
  })
})

const pageCount = computed(() => Math.max(1, Math.ceil(filteredVehicles.value.length / pageSize)))
const paginatedVehicles = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return filteredVehicles.value.slice(start, start + pageSize)
})
const firstDisplayed = computed(() => filteredVehicles.value.length === 0 ? 0 : (currentPage.value - 1) * pageSize + 1)
const lastDisplayed = computed(() => Math.min(currentPage.value * pageSize, filteredVehicles.value.length))
const hasActiveCriteria = computed(() => Boolean(search.value.trim() || selectedRoute.value || selectedDelay.value !== 'all'))
const activeFilterCount = computed(() => Number(Boolean(selectedRoute.value)) + Number(selectedDelay.value !== 'all'))
const activeFilterChips = computed(() => {
  const chips: Array<{ key: 'search' | 'route' | 'delay'; label: string; ariaLabel: string }> = []
  const trimmedSearch = search.value.trim()

  if (trimmedSearch) chips.push({
    key: 'search',
    label: `Recherche : « ${trimmedSearch} »`,
    ariaLabel: `Supprimer la recherche « ${trimmedSearch} »`,
  })
  if (selectedRoute.value) chips.push({
    key: 'route',
    label: `Ligne : ${selectedRoute.value}`,
    ariaLabel: `Supprimer le filtre de ligne ${selectedRoute.value}`,
  })
  if (selectedDelay.value !== 'all') chips.push({
    key: 'delay',
    label: selectedDelay.value === 'late' ? 'Retard important' : 'À l’heure ou léger retard',
    ariaLabel: `Supprimer le filtre « ${selectedDelay.value === 'late' ? 'retard important' : 'à l’heure ou léger retard'} »`,
  })

  return chips
})

watch([search, selectedRoute, selectedDelay, sortBy], () => {
  currentPage.value = 1
  copyStatus.value = 'idle'
  syncVehicleFiltersToUrl({ search: search.value, route: selectedRoute.value, delay: selectedDelay.value, sort: sortBy.value })
})

function restoreFiltersFromUrl(): void {
  const filters = readVehicleFilters()
  search.value = filters.search
  selectedRoute.value = filters.route
  selectedDelay.value = filters.delay
  sortBy.value = filters.sort ?? 'arrival'
  currentPage.value = 1
  copyStatus.value = 'idle'
}

onMounted(() => window.addEventListener('popstate', restoreFiltersFromUrl))
onBeforeUnmount(() => window.removeEventListener('popstate', restoreFiltersFromUrl))
watch(pageCount, (count) => {
  if (currentPage.value > count) currentPage.value = count
})

function goToPage(page: number): void {
  currentPage.value = Math.max(1, Math.min(page, pageCount.value))
}

function resetFilters(): void {
  search.value = ''
  selectedRoute.value = ''
  selectedDelay.value = 'all'
  currentPage.value = 1
}

function clearFilter(filter: 'search' | 'route' | 'delay'): void {
  if (filter === 'search') search.value = ''
  if (filter === 'route') selectedRoute.value = ''
  if (filter === 'delay') selectedDelay.value = 'all'
}

function compareNullableText(left: string | null | undefined, right: string | null | undefined): number {
  if (!left && !right) return 0
  if (!left) return 1
  if (!right) return -1
  return left.localeCompare(right, 'fr', { numeric: true })
}

function focusFiltersContent(): void {
  void nextTick(() => filtersPanel.value?.querySelector<HTMLSelectElement | HTMLButtonElement>('select, button')?.focus())
}

function openFilters(): void {
  filtersOpen.value = true
  focusFiltersContent()
}

function closeFilters(): void {
  filtersOpen.value = false
  void nextTick(() => filtersTrigger.value?.focus())
}

function toggleFilters(): void {
  if (filtersOpen.value) {
    closeFilters()
  } else {
    openFilters()
  }
}

function exportVehicles(): void {
  if (filteredVehicles.value.length === 0) return
  const timestamp = new Date().toISOString().replaceAll(/[-:]/g, '').replace(/\.\d{3}Z$/, 'Z')
  downloadCsv(`ponctuel-vehicules-${timestamp}.csv`, vehiclesToCsv(filteredVehicles.value))
}

async function copyShareableLink(): Promise<void> {
  try {
    await copyVehicleFiltersLink({ search: search.value, route: selectedRoute.value, delay: selectedDelay.value, sort: sortBy.value })
    copyStatus.value = 'success'
  } catch {
    copyStatus.value = 'error'
  }
}

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
      <label class="sort-field" for="vehicle-sort">
        <span>Trier</span>
        <select id="vehicle-sort" v-model="sortBy">
          <option value="arrival">Ordre d’arrivée</option>
          <option value="route">Ligne</option>
          <option value="vehicle">Véhicule</option>
          <option value="delay">Retard décroissant</option>
        </select>
      </label>
      <button
        class="export-button"
        type="button"
        aria-label="Exporter les véhicules en CSV"
        :disabled="props.isLoading || Boolean(props.error) || filteredVehicles.length === 0"
        @click="exportVehicles"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3v12m0 0 4-4m-4 4-4-4M5 18v2h14v-2" /></svg>
        Exporter CSV
      </button>
      <button
        class="share-button"
        type="button"
        aria-label="Copier le lien des filtres"
        @click="copyShareableLink"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.1.1l2-2a5 5 0 0 0-7.1-7.1l-1.2 1.2M14 11a5 5 0 0 0-7.1-.1l-2 2A5 5 0 0 0 7.9 20l1.2-1.2" /></svg>
        {{ copyStatus === 'success' ? 'Lien copié' : 'Copier le lien' }}
      </button>
      <span v-if="copyStatus !== 'idle'" class="sr-only" role="status" aria-live="polite">
        {{ copyStatus === 'success' ? 'Le lien des filtres a été copié.' : 'Impossible de copier le lien des filtres.' }}
      </span>
      <div class="filter-menu">
        <button
          ref="filtersTrigger"
          class="filter-button"
          type="button"
          aria-controls="vehicle-filters"
          :aria-expanded="filtersOpen"
          :aria-label="`Filtres actifs : ${activeFilterCount}`"
          @click="toggleFilters"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M7 12h10m-7 6h4" /></svg>
          Filtres <span class="filter-count">{{ activeFilterCount }}</span>
        </button>
        <div v-if="filtersOpen" ref="filtersPanel" id="vehicle-filters" class="filter-panel" role="region" aria-label="Filtres des véhicules" tabindex="-1" @keydown.esc.stop="closeFilters">
          <label class="filter-field" for="route-filter">
            <span>Ligne</span>
            <select id="route-filter" v-model="selectedRoute">
              <option value="">Toutes les lignes</option>
              <option v-for="route in routeOptions" :key="route" :value="route">{{ route }}</option>
            </select>
          </label>
          <label class="filter-field" for="delay-filter">
            <span>Retard</span>
            <select id="delay-filter" v-model="selectedDelay">
              <option value="all">Tous les états</option>
              <option value="on-time">À l’heure ou léger retard</option>
              <option value="late">Retard important</option>
            </select>
          </label>
          <div class="filter-actions">
            <button class="text-button" type="button" @click="resetFilters">Réinitialiser</button>
            <button class="text-button" type="button" @click="closeFilters">Fermer</button>
          </div>
        </div>
      </div>
    </div>
  </div>

  <div v-if="activeFilterChips.length > 0" class="active-filters" role="group" aria-label="Filtres actifs">
    <span class="active-filters-label">Filtres actifs</span>
    <ul class="filter-chips">
      <li v-for="chip in activeFilterChips" :key="chip.key">
        <button class="filter-chip" type="button" :aria-label="chip.ariaLabel" @click="clearFilter(chip.key)">
          <span>{{ chip.label }}</span>
          <span class="chip-remove" aria-hidden="true">×</span>
        </button>
      </li>
    </ul>
    <button class="text-button reset-all-filters" type="button" aria-label="Réinitialiser tous les filtres" @click="resetFilters">
      Réinitialiser tout
    </button>
  </div>

  <div v-if="props.isLoading" class="table-state" role="status">Chargement des véhicules…</div>
  <div v-else-if="props.error" class="table-state table-state-error" role="alert">Les positions ne sont pas disponibles pour le moment.</div>
  <div v-else-if="filteredVehicles.length === 0" class="table-state">
    {{ hasActiveCriteria ? 'Aucun autobus ne correspond à cette recherche ou à ces filtres.' : 'Aucun autobus en service.' }}
  </div>
  <div v-else class="table-scroll">
    <table>
      <thead><tr><th>Ligne</th><th>Véhicule</th><th>Trajet</th><th>Retard</th><th>Dernière position</th><th>Heure</th><th>Statut</th></tr></thead>
      <tbody>
        <tr v-for="vehicle in paginatedVehicles" :key="vehicle.vehicleId">
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
    <span>Affichage de {{ firstDisplayed }} à {{ lastDisplayed }} sur {{ filteredVehicles.length }} véhicules</span>
    <nav class="pagination" aria-label="Pagination des véhicules">
      <button type="button" :disabled="currentPage === 1" aria-label="Page précédente" @click="goToPage(currentPage - 1)">‹</button>
      <span class="page-label" aria-live="polite">Page {{ currentPage }} sur {{ pageCount }}</span>
      <button type="button" :disabled="currentPage === pageCount" aria-label="Page suivante" @click="goToPage(currentPage + 1)">›</button>
    </nav>
  </div>
</section>
</template>
