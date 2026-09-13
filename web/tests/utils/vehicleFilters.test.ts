import { describe, expect, it, vi } from 'vitest'
import {
  buildVehicleFiltersUrl,
  buildVehicleFiltersShareUrl,
  copyVehicleFiltersLink,
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

  it('builds an absolute share URL while preserving unrelated parameters', () => {
    expect(buildVehicleFiltersShareUrl(
      { search: 'bus', route: '80', delay: 'late' },
      'https://example.test/dashboard?tab=vehicles#table',
    )).toBe('https://example.test/dashboard?tab=vehicles&q=bus&route=80&delay=late#table')
  })

  it('copies the absolute share URL through the Clipboard API', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })

    await copyVehicleFiltersLink({ search: 'bus', route: '80', delay: 'late' })

    expect(writeText).toHaveBeenCalledWith('http://localhost:3000/?q=bus&route=80&delay=late')
  })

  it('fails explicitly when Clipboard is unavailable', async () => {
    Object.assign(navigator, { clipboard: undefined })

    await expect(copyVehicleFiltersLink({ search: '', route: '', delay: 'all' }))
      .rejects.toThrow('Clipboard API indisponible')
  })
})
