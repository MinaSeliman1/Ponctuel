import { mount } from '@vue/test-utils'
import VehicleTable from '../../src/components/VehicleTable.vue'

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
})
