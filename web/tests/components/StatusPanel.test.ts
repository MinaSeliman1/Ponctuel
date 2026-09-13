import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import StatusPanel from '../../src/components/StatusPanel.vue'

describe('StatusPanel', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows loading state', () => {
    const wrapper = mount(StatusPanel, {
      props: { dashboard: null, isLoading: true, error: null },
    })

    expect(wrapper.text()).toContain('Chargement des données')
  })

  it('shows a stale data warning', () => {
    const wrapper = mount(StatusPanel, {
      props: {
        dashboard: {
          eventCount: 42,
          lastCollectedAt: '2026-09-12T14:00:00Z',
          mode: 'fixture',
          stale: true,
        },
        isLoading: false,
        error: null,
      },
    })

    expect(wrapper.text()).toContain('Données vieillissantes')
    expect(wrapper.text()).toContain('42')
  })

  it('shows a user-safe error message', () => {
    const wrapper = mount(StatusPanel, {
      props: { dashboard: null, isLoading: false, error: 'database password leaked' },
    })

    expect(wrapper.text()).toContain('Impossible de charger les données')
    expect(wrapper.text()).not.toContain('database password')
  })

  it('emits refresh and announces an in-progress refresh', async () => {
    const wrapper = mount(StatusPanel, {
      props: {
        dashboard: {
          eventCount: 42,
          lastCollectedAt: '2026-09-12T14:00:00Z',
          mode: 'fixture',
          stale: false,
        },
        isLoading: false,
        isRefreshing: false,
        error: null,
      },
    })

    await wrapper.get('button[aria-label="Actualiser les données"]').trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)

    await wrapper.setProps({ isRefreshing: true })
    expect(wrapper.text()).toContain('Actualisation des données')
    expect(wrapper.get('button[aria-label="Actualiser les données"]').attributes('disabled')).toBeDefined()
  })

  it('updates the freshness label as time passes without new props', async () => {
    vi.useFakeTimers()
    const collectedAt = new Date('2026-09-13T08:00:00Z')
    vi.setSystemTime(collectedAt)
    const wrapper = mount(StatusPanel, {
      props: {
        dashboard: {
          eventCount: 42,
          lastCollectedAt: collectedAt.toISOString(),
          mode: 'fixture',
          stale: false,
        },
        isLoading: false,
        error: null,
      },
    })

    expect(wrapper.text()).toContain('À l’instant — données à jour.')

    await vi.advanceTimersByTimeAsync(60_000)

    expect(wrapper.text()).toContain('Il y a 1 minute.')
    wrapper.unmount()
  })
})
