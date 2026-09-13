import { mount } from '@vue/test-utils'
import StatusPanel from '../../src/components/StatusPanel.vue'

describe('StatusPanel', () => {
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
})
