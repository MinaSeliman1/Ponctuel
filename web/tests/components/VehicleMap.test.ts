import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import VehicleMap from '../../src/components/VehicleMap.vue'

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
  {
    vehicleId: '9988',
    routeId: '80',
    tripId: null,
    latitude: 45.54,
    longitude: -73.61,
    recordedAt: '2026-09-12T14:01:00Z',
    delaySeconds: 360,
  },
]

describe('VehicleMap', () => {
  it('sélectionne un autobus au clic et affiche ses détails', async () => {
    const wrapper = mount(VehicleMap, { props: { vehicles, isLoading: false } })

    await wrapper.get('[aria-label="Autobus 1234"]').trigger('click')

    expect(wrapper.get('[role="dialog"]').text()).toContain('Autobus 1234')
    expect(wrapper.get('[role="dialog"]').text()).toContain('trip-51')
    expect(wrapper.get('[aria-label="Autobus 1234"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('[aria-label="Autobus 1234"]').attributes('tabindex')).toBe('0')
  })

  it('sélectionne un autobus avec Entrée et ferme la fiche au clavier', async () => {
    const wrapper = mount(VehicleMap, { props: { vehicles, isLoading: false } })

    await wrapper.get('[aria-label="Autobus 9988"]').trigger('keydown', { key: 'Enter' })
    expect(wrapper.get('[role="dialog"]').text()).toContain('Autobus 9988')
    expect(wrapper.get('[role="dialog"]').text()).toContain('Non identifié')

    await wrapper.get('[aria-label="Fermer le détail de l’autobus 9988"]').trigger('click')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })

  it('retire la fiche si le véhicule sélectionné disparaît des données', async () => {
    const wrapper = mount(VehicleMap, { props: { vehicles, isLoading: false } })

    await wrapper.get('[aria-label="Autobus 1234"]').trigger('click')
    await wrapper.setProps({ vehicles: [vehicles[1]] })

    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })

  it('contrôle les couches disponibles et expose une légende des retards', async () => {
    const wrapper = mount(VehicleMap, { props: { vehicles, isLoading: false } })

    expect(wrapper.get('[aria-label="Légende des retards"]').text()).toContain('À l’heure')
    expect(wrapper.find('[aria-label="Arrêts"]').exists()).toBe(false)
    expect(wrapper.find('[aria-label="Zones de service"]').exists()).toBe(false)

    await wrapper.get('[aria-label="Positions des autobus"]').setValue(false)
    expect(wrapper.find('.vehicle-markers').exists()).toBe(false)

    await wrapper.get('[aria-label="Réseau routier"]').setValue(false)
    expect(wrapper.find('.map-streets').exists()).toBe(false)

    await wrapper.get('[aria-label="Positions des autobus"]').setValue(true)
    expect(wrapper.find('.vehicle-markers').exists()).toBe(true)
  })

  it('ferme la fiche quand les positions sont masquées', async () => {
    const wrapper = mount(VehicleMap, { props: { vehicles, isLoading: false } })

    await wrapper.get('[aria-label="Autobus 1234"]').trigger('click')
    await wrapper.get('[aria-label="Positions des autobus"]').setValue(false)

    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })
})
