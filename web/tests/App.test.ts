import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from '../src/App.vue'

const dashboard = {
  eventCount: 24,
  lastCollectedAt: '2026-09-12T14:00:00Z',
  mode: 'STM',
  stale: false,
}

const vehicles = [
  {
    vehicleId: '1234',
    routeId: '51',
    tripId: 'trip-51',
    latitude: 45.52,
    longitude: -73.58,
    recordedAt: '2026-09-12T14:00:00Z',
    delaySeconds: 30,
  },
]

const summaries = [
  { routeId: '51', horizonSeconds: 300, sampleCount: 4, meanErrorSeconds: 60, onTimeRate: 0.75 },
  { routeId: '80', horizonSeconds: 300, sampleCount: 2, meanErrorSeconds: -30, onTimeRate: 1 },
]

function mockApi(options: { qualityStatus?: number; qualityData?: typeof summaries } = {}) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation(async (_input, init) => {
    const body = JSON.parse(String((init as RequestInit).body)) as { query: string }
    if (body.query.includes('ErrorSummary')) {
      if (options.qualityStatus && options.qualityStatus !== 200) {
        return new Response('unavailable', { status: options.qualityStatus })
      }
      return new Response(JSON.stringify({ data: { errorSummary: options.qualityData ?? summaries } }), { status: 200 })
    }
    if (body.query.includes('Vehicles')) {
      return new Response(JSON.stringify({ data: { vehicles } }), { status: 200 })
    }
    return new Response(JSON.stringify({ data: { dashboard } }), { status: 200 })
  })
}

describe('App prediction quality integration', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('loads the quality chart alongside the live network data', async () => {
    mockApi()
    const wrapper = mount(App)

    await flushPromises()

    expect(wrapper.text()).toContain('Qualité des prédictions')
    expect(wrapper.text()).toContain('+30 s')
    expect(wrapper.text()).toContain('Véhicules en service')
  })

  it('keeps the network dashboard available when quality data fails', async () => {
    mockApi({ qualityStatus: 503 })
    const wrapper = mount(App)

    await flushPromises()

    expect(wrapper.text()).toContain('Qualité indisponible pour le moment.')
    expect(wrapper.text()).toContain('Véhicules en service')
    expect(wrapper.text()).toContain('1234')
  })

  it('explains when the API has no observations for quality yet', async () => {
    mockApi({ qualityData: [] })
    const wrapper = mount(App)

    await flushPromises()

    expect(wrapper.text()).toContain('Pas encore assez d’observations correspondantes')
  })
})
