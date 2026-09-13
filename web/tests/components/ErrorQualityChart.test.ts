import { mount } from '@vue/test-utils'
import ErrorQualityChart from '../../src/components/ErrorQualityChart.vue'
import type { ErrorSummary } from '../../src/types'

const summaries: ErrorSummary[] = [
  { routeId: '51', horizonSeconds: 300, sampleCount: 4, meanErrorSeconds: 60, onTimeRate: 0.75 },
  { routeId: '80', horizonSeconds: 300, sampleCount: 2, meanErrorSeconds: -30, onTimeRate: 1 },
  { routeId: null, horizonSeconds: 600, sampleCount: 2, meanErrorSeconds: -15, onTimeRate: 1 },
]

describe('ErrorQualityChart', () => {
  it('shows loading, error and empty states', () => {
    expect(mount(ErrorQualityChart, { props: { summaries: [], isLoading: true, hasError: false, stale: false } }).text()).toContain('Chargement de la qualité')
    expect(mount(ErrorQualityChart, { props: { summaries: [], isLoading: false, hasError: true, stale: true } }).text()).toContain('Qualité indisponible')
    expect(mount(ErrorQualityChart, { props: { summaries: [], isLoading: false, hasError: false, stale: false } }).text()).toContain('Pas encore assez d’observations')
  })

  it('aggregates routes with sample-count weighting and exposes accessible details', () => {
    const wrapper = mount(ErrorQualityChart, { props: { summaries, isLoading: false, hasError: false, stale: true } })

    expect(wrapper.text()).toContain('+30 s')
    expect(wrapper.text()).toContain('83 %')
    expect(wrapper.text()).toContain('−15 s')
    expect(wrapper.text()).toContain('Données vieillissantes')
    expect(wrapper.find('svg[role="img"] title').text()).toContain('Erreur moyenne par horizon')
    expect(wrapper.findAll('table tbody tr')).toHaveLength(2)
    expect(wrapper.find('svg').attributes('aria-describedby')).toBe('quality-chart-description')
  })
})
