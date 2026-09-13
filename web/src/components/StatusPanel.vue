<script setup lang="ts">
import type { Dashboard } from '../types'

const props = withDefaults(defineProps<{
  dashboard: Dashboard | null
  isLoading: boolean
  error: string | null
  isRefreshing?: boolean
  refreshError?: boolean
}>(), { isRefreshing: false, refreshError: false })

defineEmits<{ refresh: [] }>()

function formatDate(value: string | null): string {
  if (!value) return 'Aucune collecte enregistrée'
  return new Intl.DateTimeFormat('fr-CA', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

function relativeLabel(value: string | null): string {
  if (!value) return 'En attente d’une première collecte.'
  const minutes = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 60000))
  if (minutes === 0) return 'À l’instant — données à jour.'
  if (minutes === 1) return 'Il y a 1 minute.'
  return `Il y a ${minutes} minutes.`
}

function modeLabel(mode: string): string {
  return mode.toLowerCase() === 'stm' ? 'Temps réel (bus)' : 'Fixture local'
}
</script>

<template>
<section class="status-panel" aria-live="polite">
  <div class="collection-status">
    <div class="status-heading-row">
      <span class="eyebrow">Dernière collecte</span>
      <span v-if="props.dashboard?.stale" class="stale-label" role="status">
        <span class="warning-icon" aria-hidden="true">!</span>
        Données vieillissantes
      </span>
    </div>
    <p v-if="props.isLoading" class="status-value status-muted">Chargement des données…</p>
    <p v-else-if="props.error" class="status-value status-error">Impossible de charger les données</p>
    <template v-else>
      <p class="status-value">{{ formatDate(props.dashboard?.lastCollectedAt ?? null) }}</p>
      <p class="status-detail">{{ relativeLabel(props.dashboard?.lastCollectedAt ?? null) }}</p>
    </template>
    <p v-if="props.refreshError" class="status-detail status-refresh-error" role="alert">
      Actualisation impossible. Les dernières données valides sont conservées. Vous pouvez réessayer.
    </p>
    <div v-if="!props.isLoading" class="refresh-actions">
      <button
        class="text-button"
        type="button"
        aria-label="Actualiser les données"
        :disabled="props.isRefreshing"
        @click="$emit('refresh')"
      >
        {{ props.error || props.refreshError ? 'Réessayer' : 'Actualiser' }}
      </button>
      <span v-if="props.isRefreshing" class="refresh-status" role="status">Actualisation des données…</span>
    </div>
  </div>

  <div class="status-metric">
    <span class="eyebrow">Événements</span>
    <strong v-if="props.dashboard && !props.isLoading">{{ props.dashboard.eventCount.toLocaleString('fr-CA') }}</strong>
    <span v-else class="metric-placeholder" aria-label="Chargement">—</span>
  </div>

  <div class="status-metric status-metric-wide">
    <span class="eyebrow">Mode</span>
    <span class="mode-select">
      <span class="mode-dot" aria-hidden="true"></span>
      {{ modeLabel(props.dashboard?.mode ?? 'fixture') }}
      <span class="chevron" aria-hidden="true">⌄</span>
    </span>
  </div>
</section>
</template>
