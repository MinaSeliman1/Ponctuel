import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach } from 'vitest'
import VehicleTable from '../../src/components/VehicleTable.vue'

afterEach(() => {
  window.history.replaceState(null, '', '/')
})

function vehicle(index: number, routeId = '51', delaySeconds: number | null = 30) {
  return {
    vehicleId: `bus-${index}`,
    routeId,
    tripId: `trip-${index}`,
    latitude: 45.5,
    longitude: -73.5,
    recordedAt: '2026-09-12T14:00:00Z',
    delaySeconds,
  }
}

describe('VehicleTable', () => {
  it('shows an empty state', () => {
    const wrapper = mount(VehicleTable, {
      props: { vehicles: [], isLoading: false, error: null },
    })

    expect(wrapper.text()).toContain('Aucun autobus')
  })

  it('renders line, trip and delay status', () => {
    const wrapper = mount(VehicleTable, {
      props: {
        vehicles: [{
          vehicleId: '31-113',
          routeId: '165',
          tripId: 'trip-1',
          latitude: 45.5,
          longitude: -73.5,
          recordedAt: '2026-09-12T14:00:00Z',
          delaySeconds: 120,
        }],
        isLoading: false,
        error: null,
      },
    })

    expect(wrapper.text()).toContain('165')
    expect(wrapper.text()).toContain('31-113')
    expect(wrapper.text()).toContain('trip-1')
    expect(wrapper.text()).toContain('+2 min')
  })

  it('filters by route and delay, then resets the advanced filters', async () => {
    const wrapper = mount(VehicleTable, {
      props: {
        vehicles: [vehicle(1, '51', 30), vehicle(2, '51', 360), vehicle(3, '80', null)],
        isLoading: false,
        error: null,
      },
    })

    await wrapper.get('button[aria-label="Filtres actifs : 0"]').trigger('click')
    await wrapper.get('#route-filter').setValue('80')

    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.find('tbody').text()).toContain('bus-3')

    await wrapper.get('#delay-filter').setValue('late')
    expect(wrapper.find('.table-state').text()).toContain('Aucun autobus')

    const resetButton = wrapper.findAll('#vehicle-filters button').find((button) => button.text() === 'Réinitialiser')
    expect(resetButton).toBeDefined()
    await resetButton!.trigger('click')
    expect(wrapper.findAll('tbody tr')).toHaveLength(3)
  })

  it('paginates ten vehicles and exposes the current page', async () => {
    const wrapper = mount(VehicleTable, {
      props: {
        vehicles: Array.from({ length: 11 }, (_, index) => vehicle(index + 1)),
        isLoading: false,
        error: null,
      },
    })

    expect(wrapper.findAll('tbody tr')).toHaveLength(10)
    expect(wrapper.find('.page-label').text()).toBe('Page 1 sur 2')
    expect(wrapper.get('button[aria-label="Page précédente"]').attributes('disabled')).toBeDefined()

    await wrapper.get('button[aria-label="Page suivante"]').trigger('click')

    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.find('tbody').text()).toContain('bus-11')
    expect(wrapper.find('.page-label').text()).toBe('Page 2 sur 2')
    expect(wrapper.get('button[aria-label="Page suivante"]').attributes('disabled')).toBeDefined()
  })

  it('focuses the first filter and restores focus after Escape', async () => {
    const wrapper = mount(VehicleTable, {
      attachTo: document.body,
      props: { vehicles: [vehicle(1)], isLoading: false, error: null },
    })
    const filtersButton = wrapper.get('button[aria-label="Filtres actifs : 0"]')

    await filtersButton.trigger('click')
    await nextTick()
    expect(document.activeElement).toBe(wrapper.get('#route-filter').element)

    await wrapper.get('#route-filter').trigger('keydown', { key: 'Escape' })
    await nextTick()

    expect(wrapper.find('#vehicle-filters').exists()).toBe(false)
    expect(document.activeElement).toBe(filtersButton.element)
    wrapper.unmount()
  })

  it('enables CSV export only when filtered vehicles are available', async () => {
    const wrapper = mount(VehicleTable, {
      props: {
        vehicles: [vehicle(1, '51'), vehicle(2, '80')],
        isLoading: false,
        error: null,
      },
    })
    const exportButton = wrapper.get('button[aria-label="Exporter les véhicules en CSV"]')

    expect(exportButton.attributes('disabled')).toBeUndefined()
    await wrapper.get('input[type="search"]').setValue('inexistant')
    expect(wrapper.get('button[aria-label="Exporter les véhicules en CSV"]').attributes('disabled')).toBeDefined()
  })

  it('restores filters from the URL and updates the shareable state', async () => {
    window.history.replaceState(null, '', '/?q=bus&route=80&delay=late')
    const wrapper = mount(VehicleTable, {
      props: {
        vehicles: [vehicle(1, '51', 30), vehicle(2, '80', 360)],
        isLoading: false,
        error: null,
      },
    })

    expect((wrapper.get('input[type="search"]').element as HTMLInputElement).value).toBe('bus')
    expect(wrapper.find('#route-filter').exists()).toBe(false)
    await wrapper.get('button[aria-label="Filtres actifs : 2"]').trigger('click')
    expect((wrapper.get('#route-filter').element as HTMLSelectElement).value).toBe('80')
    expect((wrapper.get('#delay-filter').element as HTMLSelectElement).value).toBe('late')

    await wrapper.get('input[type="search"]').setValue('9988')
    await nextTick()
    expect(window.location.search).toBe('?q=9988&route=80&delay=late')
    wrapper.unmount()
  })
})
