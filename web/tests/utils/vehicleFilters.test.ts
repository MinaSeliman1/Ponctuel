import { describe, expect, it, vi } from 'vitest'
import {
  buildVehicleFiltersUrl,
  readVehicleFilters,
  syncVehicleFiltersToUrl,
} from '../../src/utils/vehicleFilters'

describe('vehicle filter URL state', () => {
  it('reads valid values and safely ignores an unknown delay', () => {
    expect(readVehicleFilters('?q=late%20bus&route=80&delay=late')).toEqual({
      search: 'late bus',
      route: '80',
      delay: 'late',
    })
    expect(readVehicleFilters('?delay=unknown')).toEqual({ search: '', route: '', delay: 'all' })
  })

  it('builds a stable URL and removes default values', () => {
    expect(buildVehicleFiltersUrl(
      { search: '  bus  ', route: '80', delay: 'late' },
      'https://example.test/dashboard?old=value#vehicles',
    )).toBe('/dashboard?old=value&q=bus&route=80&delay=late#vehicles')
    expect(buildVehicleFiltersUrl(
      { search: '', route: '', delay: 'all' },
      'https://example.test/dashboard?q=old&route=51&delay=late',
    )).toBe('/dashboard')
  })

  it('updates the current URL without creating browser history entries', () => {
    const replaceState = vi.spyOn(window.history, 'replaceState')

    syncVehicleFiltersToUrl({ search: 'bus', route: '', delay: 'on-time' })

    expect(replaceState).toHaveBeenCalledWith(null, '', '/?q=bus&delay=on-time')
  })
})
