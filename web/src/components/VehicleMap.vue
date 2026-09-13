<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Vehicle } from '../types'

const props = defineProps<{
  vehicles: Vehicle[]
  isLoading: boolean
}>()

const visibleVehicles = computed(() => props.vehicles.slice(0, 150))
const selectedVehicleId = ref<string | null>(null)
const selectedVehicle = computed(() => props.vehicles.find((vehicle) => vehicle.vehicleId === selectedVehicleId.value) ?? null)
const showStreets = ref(true)
const showVehicles = ref(true)

watch(selectedVehicle, (vehicle) => {
  if (!vehicle) selectedVehicleId.value = null
})
watch(showVehicles, (visible) => {
  if (!visible) selectedVehicleId.value = null
})

function selectVehicle(vehicleId: string): void {
  if (props.vehicles.some((vehicle) => vehicle.vehicleId === vehicleId)) selectedVehicleId.value = vehicleId
}

function clearSelection(): void {
  selectedVehicleId.value = null
}

function xFor(longitude: number): number {
  return Math.min(1130, Math.max(70, ((longitude + 73.72) / 0.29) * 1060 + 70))
}

function yFor(latitude: number): number {
  return Math.min(300, Math.max(55, 320 - ((latitude - 45.43) / 0.19) * 260))
}

function markerTone(delay: number | null): string {
  if (delay === null || delay <= 120) return '#079b76'
  if (delay <= 600) return '#d39111'
  return '#c83232'
}

function formatDelay(seconds: number | null): string {
  if (seconds === null) return 'Non disponible'
  const minutes = Math.round(seconds / 60)
  return `${minutes > 0 ? '+' : ''}${minutes} min`
}

function formatTime(value: string): string {
  return new Intl.DateTimeFormat('fr-CA', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}
</script>

<template>
<section class="map-panel" aria-labelledby="map-title">
  <div class="map-title" id="map-title">
    <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 17.5h12M7 17.5v-7h10v7M9 10.5V7.8A2.8 2.8 0 0 1 11.8 5h.4A2.8 2.8 0 0 1 15 7.8v2.7M8 20h2m4 0h2M8 13h8" /></svg>
    <span>Positions des autobus</span>
  </div>
  <div class="map-controls" aria-label="Couches de carte">
    <label><input v-model="showStreets" type="checkbox" aria-label="Réseau routier" /> <span>Réseau routier</span></label>
    <label><input v-model="showVehicles" type="checkbox" aria-label="Positions des autobus" /> <span>Positions des autobus</span></label>
  </div>
  <div class="map-legend" role="group" aria-label="Légende des retards">
    <span class="map-legend-title">État du service</span>
    <span><i class="legend-dot legend-dot-on-time" aria-hidden="true"></i>À l’heure</span>
    <span><i class="legend-dot legend-dot-warn" aria-hidden="true"></i>Léger retard</span>
    <span><i class="legend-dot legend-dot-late" aria-hidden="true"></i>Retard important</span>
  </div>
  <svg class="transit-map" viewBox="0 0 1200 360" role="img" aria-label="Carte schématique des positions d’autobus à Montréal">
    <rect width="1200" height="360" fill="#eef2f3" />
    <path class="map-water" d="M910 0h290v360H1010c-18-29-5-61-31-90-24-27-50-34-47-76 2-32 38-48 34-82-4-35-45-55-56-112Z" />
    <path class="map-island" d="M0 0h894c19 29 38 47 39 73 1 38-42 58-42 96 0 36 40 51 39 88-1 36-50 56-82 103H0Z" />
    <g v-if="showStreets" class="map-streets">
      <path d="M-80 56 880 286M-45 130 815 334M-50 215 880 42M20 310 890 138M116 0 42 360M265 0 190 360M418 0 350 360M580 0 512 360M745 0 676 360M0 95 890 94M0 176 888 176M0 258 890 258" />
      <path d="M72 20 862 340M188 0 872 310M340 0 884 268M515 0 893 212M688 0 900 164" class="major-street" />
    </g>
    <g class="map-labels">
      <text x="90" y="72">Mont-Royal</text>
      <text x="185" y="150">Côte-des-Neiges</text>
      <text x="382" y="78">Le Plateau-Mont-Royal</text>
      <text x="310" y="198">Westmount</text>
      <text x="606" y="178" class="city-label">Montréal</text>
      <text x="1030" y="72">Longueuil</text>
      <text x="723" y="326">Verdun</text>
      <text x="822" y="130">Fleuve Saint-Laurent</text>
    </g>
    <g v-if="isLoading" class="map-loading"><text x="600" y="190" text-anchor="middle">Chargement des positions…</text></g>
    <g v-else-if="showVehicles" class="vehicle-markers">
      <g
        v-for="vehicle in visibleVehicles"
        :key="vehicle.vehicleId"
        class="vehicle-marker"
        :class="{ 'vehicle-marker-selected': selectedVehicleId === vehicle.vehicleId }"
        :transform="`translate(${xFor(vehicle.longitude)} ${yFor(vehicle.latitude)})`"
        role="button"
        tabindex="0"
        :aria-label="`Autobus ${vehicle.vehicleId}`"
        :aria-pressed="selectedVehicleId === vehicle.vehicleId"
        @click="selectVehicle(vehicle.vehicleId)"
        @keydown.enter.prevent="selectVehicle(vehicle.vehicleId)"
        @keydown.space.prevent="selectVehicle(vehicle.vehicleId)"
      >
        <circle r="14" :fill="markerTone(vehicle.delaySeconds)" stroke="#ffffff" stroke-width="3" />
        <path d="M-6 5v-8h12v8M-4-3v-3h8v3M-5 7h2m6 0h2" stroke="#fff" stroke-width="1.8" fill="none" stroke-linecap="round" />
      </g>
    </g>
    <g class="north-control"><rect x="1132" y="16" width="42" height="58" rx="6" /><path d="m1153 28 7 18-7-4-7 4 7-18Z" /><text x="1153" y="66" text-anchor="middle">N</text></g>
    <g class="zoom-control"><rect x="1132" y="98" width="42" height="92" rx="6" /><path d="M1143 126h20M1153 116v20M1143 163h20" /></g>
    <g class="scale-control"><path d="M1025 326h120" /><path d="M1025 320v12m60-12v12m60-12v12" /><text x="1025" y="314">0</text><text x="1085" y="314">2</text><text x="1145" y="314" text-anchor="end">4 km</text></g>
  </svg>
  <aside v-if="selectedVehicle" class="vehicle-detail" role="dialog" :aria-labelledby="`vehicle-detail-title-${selectedVehicle.vehicleId}`">
    <div class="vehicle-detail-heading">
      <span class="eyebrow">Véhicule sélectionné</span>
      <button class="vehicle-detail-close" type="button" :aria-label="`Fermer le détail de l’autobus ${selectedVehicle.vehicleId}`" @click="clearSelection">×</button>
    </div>
    <h3 :id="`vehicle-detail-title-${selectedVehicle.vehicleId}`">Autobus {{ selectedVehicle.vehicleId }}</h3>
    <dl class="vehicle-detail-list">
      <div><dt>Ligne</dt><dd>{{ selectedVehicle.routeId ?? 'Non identifiée' }}</dd></div>
      <div><dt>Trajet</dt><dd>{{ selectedVehicle.tripId ?? 'Non identifié' }}</dd></div>
      <div><dt>Retard</dt><dd>{{ formatDelay(selectedVehicle.delaySeconds) }}</dd></div>
      <div><dt>Position</dt><dd>{{ selectedVehicle.latitude.toFixed(4) }}, {{ selectedVehicle.longitude.toFixed(4) }}</dd></div>
      <div><dt>Collecté</dt><dd>{{ formatTime(selectedVehicle.recordedAt) }}</dd></div>
    </dl>
  </aside>
</section>
</template>
