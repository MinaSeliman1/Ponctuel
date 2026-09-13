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

function mockApi(options: { qualityStatus?: number; qualityData?: typeof summaries; refreshStatus?: number } = {}) {
  const requestCounts = { dashboard: 0, vehicles: 0, quality: 0 }
  return vi.spyOn(globalThis, 'fetch').mockImplementation(async (_input, init) => {
    const body = JSON.parse(String((init as RequestInit).body)) as { query: string }
    if (body.query.includes('ErrorSummary')) {
      requestCounts.quality += 1
      if (options.refreshStatus && requestCounts.quality > 1) {
        return new Response('unavailable', { status: options.refreshStatus })
      }
      if (options.qualityStatus && options.qualityStatus !== 200) {
        return new Response('unavailable', { status: options.qualityStatus })
      }
      return new Response(JSON.stringify({ data: { errorSummary: options.qualityData ?? summaries } }), { status: 200 })
    }
    if (body.query.includes('Vehicles')) {
      requestCounts.vehicles += 1
      if (options.refreshStatus && requestCounts.vehicles > 1) {
        return new Response('unavailable', { status: options.refreshStatus })
      }
      return new Response(JSON.stringify({ data: { vehicles } }), { status: 200 })
    }
    requestCounts.dashboard += 1
    if (options.refreshStatus && requestCounts.dashboard > 1) {
      return new Response('unavailable', { status: options.refreshStatus })
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

  it('refreshes the three dashboard queries without losing the network view', async () => {
    const fetchSpy = mockApi()
    const wrapper = mount(App)

    await flushPromises()
    expect(fetchSpy).toHaveBeenCalledTimes(3)

    await wrapper.get('button[aria-label="Actualiser les données"]').trigger('click')
    await flushPromises()

    expect(fetchSpy).toHaveBeenCalledTimes(6)
    expect(wrapper.text()).toContain('1234')
    wrapper.unmount()
  })

  it('keeps the last valid view when a refresh fails', async () => {
    const fetchSpy = mockApi({ refreshStatus: 503 })
    const wrapper = mount(App)

    await flushPromises()
    await wrapper.get('button[aria-label="Actualiser les données"]').trigger('click')
    await flushPromises()

    expect(fetchSpy).toHaveBeenCalledTimes(6)
    expect(wrapper.text()).toContain('1234')
    expect(wrapper.find('table').exists()).toBe(true)
    expect(wrapper.get('[role="alert"]').text()).toContain('dernières données valides sont conservées')
    expect(wrapper.get('button[aria-label="Actualiser les données"]').text()).toContain('Réessayer')
    wrapper.unmount()
  })

  it('allows disabling automatic refresh while keeping manual refresh available', async () => {
    const fetchSpy = mockApi()
    const wrapper = mount(App)

    await flushPromises()
    await wrapper.get('button[aria-label="Préférences"]').trigger('click')

    expect(wrapper.get('[role="dialog"] h2').text()).toBe('Préférences')
    const autoRefresh = wrapper.get('input[type="checkbox"]')
    expect((autoRefresh.element as HTMLInputElement).checked).toBe(true)

    await autoRefresh.setValue(false)
    expect(wrapper.text()).toContain('Actualisation automatique suspendue')

    await wrapper.get('button[aria-label="Actualiser les données"]').trigger('click')
    await flushPromises()
    expect(fetchSpy).toHaveBeenCalledTimes(6)
    wrapper.unmount()
  })

  it('focuses and closes the preferences dialog from the keyboard', async () => {
    mockApi()
    const wrapper = mount(App, { attachTo: document.body })

    await flushPromises()
    const preferencesButton = wrapper.get('button[aria-label="Préférences"]')
    await preferencesButton.trigger('click')
    await flushPromises()

    expect(document.activeElement).toBe(wrapper.get('input[type="checkbox"]').element)

    await wrapper.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    await flushPromises()

    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement).toBe(preferencesButton.element)
    wrapper.unmount()
  })
})
