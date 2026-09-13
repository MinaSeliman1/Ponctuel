<script setup lang="ts">
import { computed } from 'vue'
import type { ErrorSummary } from '../types'

interface Props {
  summaries: ErrorSummary[]
  isLoading: boolean
  hasError: boolean
  stale: boolean
}

interface HorizonPoint {
  horizonSeconds: number
  sampleCount: number
  meanErrorSeconds: number
  onTimeRate: number
}

const props = defineProps<Props>()

const width = 760
const height = 300
const plotLeft = 62
const plotRight = 22
const plotTop = 28
const plotBottom = 52
const plotWidth = width - plotLeft - plotRight
const plotHeight = height - plotTop - plotBottom

const points = computed<HorizonPoint[]>(() => {
  const byHorizon = new Map<number, { sampleCount: number; weightedError: number; weightedOnTime: number }>()
  for (const summary of props.summaries) {
    if (!Number.isFinite(summary.horizonSeconds) || !Number.isFinite(summary.sampleCount) || summary.sampleCount <= 0) continue
    if (!Number.isFinite(summary.meanErrorSeconds) || !Number.isFinite(summary.onTimeRate)) continue
    const current = byHorizon.get(summary.horizonSeconds) ?? { sampleCount: 0, weightedError: 0, weightedOnTime: 0 }
    current.sampleCount += summary.sampleCount
    current.weightedError += summary.meanErrorSeconds * summary.sampleCount
    current.weightedOnTime += summary.onTimeRate * summary.sampleCount
    byHorizon.set(summary.horizonSeconds, current)
  }
  return [...byHorizon.entries()]
    .map(([horizonSeconds, values]) => ({
      horizonSeconds,
      sampleCount: values.sampleCount,
      meanErrorSeconds: values.weightedError / values.sampleCount,
      onTimeRate: values.weightedOnTime / values.sampleCount,
    }))
    .sort((left, right) => left.horizonSeconds - right.horizonSeconds)
})

const scale = computed(() => {
  const maxHorizon = Math.max(300, Math.ceil((points.value.at(-1)?.horizonSeconds ?? 300) / 300) * 300)
  const maximumAbsoluteError = Math.max(
    60,
    ...points.value.map((point) => Math.abs(point.meanErrorSeconds)),
  )
  const maxY = Math.ceil(maximumAbsoluteError / 60) * 60
  return { maxHorizon, maxY }
})

const chartPoints = computed(() => points.value.map((point) => ({
  ...point,
  x: plotLeft + (point.horizonSeconds / scale.value.maxHorizon) * plotWidth,
  y: plotTop + ((scale.value.maxY - point.meanErrorSeconds) / (scale.value.maxY * 2)) * plotHeight,
})))

const polyline = computed(() => chartPoints.value.map((point) => `${point.x},${point.y}`).join(' '))
const zeroY = computed(() => plotTop + plotHeight / 2)
const yTicks = computed(() => [scale.value.maxY, 0, -scale.value.maxY])

function formatError(value: number): string {
  const rounded = Math.round(value * 10) / 10
  const number = Math.abs(rounded).toLocaleString('fr-CA', { maximumFractionDigits: 1 })
  if (rounded > 0) return `+${number} s`
  if (rounded < 0) return `−${number} s`
  return '0 s'
}

function formatHorizon(seconds: number): string {
  return `${Math.round(seconds / 60)} min`
}

function formatRate(value: number): string {
  return `${Math.round(value * 100)} %`
}

function toneClass(value: number): string {
  const absolute = Math.abs(value)
  if (absolute <= 60) return 'quality-point-good'
  if (absolute <= 300) return 'quality-point-warn'
  return 'quality-point-late'
}
</script>

<template>
  <section class="quality-panel" aria-labelledby="quality-title">
    <div class="quality-heading">
      <div>
        <span class="eyebrow">Mesure observée</span>
        <h2 id="quality-title">Qualité des prédictions</h2>
        <p class="quality-subtitle">Erreur moyenne par horizon, pondérée par le nombre d’observations.</p>
      </div>
      <span v-if="props.stale" class="quality-stale" role="status">Données vieillissantes</span>
    </div>

    <div v-if="props.isLoading" class="quality-state" role="status">Chargement de la qualité des prédictions…</div>
    <div v-else-if="props.hasError" class="quality-state quality-state-error" role="alert">Qualité indisponible pour le moment.</div>
    <div v-else-if="!points.length" class="quality-state" role="status">Pas encore assez d’observations correspondantes pour calculer une erreur.</div>
    <div v-else class="quality-content">
      <svg
        class="quality-chart"
        :viewBox="`0 0 ${width} ${height}`"
        role="img"
        aria-labelledby="quality-chart-title"
        aria-describedby="quality-chart-description"
      >
        <title id="quality-chart-title">Erreur moyenne par horizon</title>
        <desc id="quality-chart-description">Chaque point agrège les arrivées observées pour un horizon de prédiction. Une erreur positive indique une arrivée après la prédiction.</desc>
        <line class="quality-axis" :x1="plotLeft" :x2="plotLeft" :y1="plotTop" :y2="plotTop + plotHeight" />
        <line class="quality-axis" :x1="plotLeft" :x2="plotLeft + plotWidth" :y1="plotTop + plotHeight" :y2="plotTop + plotHeight" />
        <line class="quality-zero" :x1="plotLeft" :x2="plotLeft + plotWidth" :y1="zeroY" :y2="zeroY" />
        <g v-for="tick in yTicks" :key="tick" class="quality-tick">
          <text :x="plotLeft - 10" :y="plotTop + ((scale.maxY - tick) / (scale.maxY * 2)) * plotHeight + 4" text-anchor="end">{{ formatError(tick) }}</text>
        </g>
        <text class="quality-axis-label" :x="plotLeft" :y="height - 14">0 min</text>
        <text class="quality-axis-label" :x="plotLeft + plotWidth" :y="height - 14" text-anchor="end">{{ formatHorizon(scale.maxHorizon) }}</text>
        <polyline class="quality-line" :points="polyline" />
        <g v-for="point in chartPoints" :key="point.horizonSeconds">
          <circle :class="['quality-point', toneClass(point.meanErrorSeconds)]" :cx="point.x" :cy="point.y" r="6" :aria-label="`${formatHorizon(point.horizonSeconds)} : ${formatError(point.meanErrorSeconds)}`" />
        </g>
      </svg>
      <p class="sr-only">Le graphique couvre {{ points.length }} horizons et {{ points.reduce((total, point) => total + point.sampleCount, 0).toLocaleString('fr-CA') }} observations.</p>
      <div class="quality-table-scroll">
        <table class="quality-table">
          <caption class="sr-only">Résumé textuel de la qualité des prédictions par horizon</caption>
          <thead>
            <tr><th scope="col">Horizon</th><th scope="col">Erreur moyenne</th><th scope="col">À l’heure</th><th scope="col">Observations</th></tr>
          </thead>
          <tbody>
            <tr v-for="point in points" :key="`row-${point.horizonSeconds}`">
              <td>{{ formatHorizon(point.horizonSeconds) }}</td>
              <td :class="toneClass(point.meanErrorSeconds)">{{ formatError(point.meanErrorSeconds) }}</td>
              <td>{{ formatRate(point.onTimeRate) }}</td>
              <td>{{ point.sampleCount.toLocaleString('fr-CA') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>
