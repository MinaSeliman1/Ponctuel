import { afterEach, describe, expect, it, vi } from 'vitest'
import { downloadCsv, vehiclesToCsv } from '../../src/utils/csv'

const vehicle = {
  vehicleId: '31-113',
  routeId: '51, express',
  tripId: 'Trip "special"\nsoir',
  latitude: 45.5,
  longitude: -73.5,
  recordedAt: '2026-09-12T14:00:00Z',
  delaySeconds: null,
}

describe('csv utilities', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('serializes vehicle values with stable headers and CSV escaping', () => {
    const csv = vehiclesToCsv([vehicle])

    expect(csv).toContain('\uFEFFroute_id,vehicle_id,trip_id,latitude,longitude,recorded_at_utc,delay_seconds')
    expect(csv).toContain('"51, express",31-113,"Trip ""special""\nsoir",45.5,-73.5,2026-09-12T14:00:00Z,')
    expect(csv.endsWith('\r\n')).toBe(true)
  })

  it('neutralizes formula prefixes in text fields without changing numeric values', () => {
    const csv = vehiclesToCsv([{
      ...vehicle,
      routeId: '=HYPERLINK("https://example.com")',
      vehicleId: '+cmd|/C calc!A0',
      tripId: '@SUM(1+1)',
      longitude: -73.5,
      delaySeconds: -90,
    }])

    expect(csv).toContain('"\'=HYPERLINK(""https://example.com"")"')
    expect(csv).toContain("'+cmd|/C calc!A0")
    expect(csv).toContain("'@SUM(1+1)")

    const minusFormulaCsv = vehiclesToCsv([{ ...vehicle, routeId: '-1+1' }])
    expect(minusFormulaCsv).toContain("'-1+1")
    expect(csv).toContain(',45.5,-73.5,2026-09-12T14:00:00Z,-90\r\n')
  })

  it('creates a local browser download without sending data elsewhere', () => {
    vi.useFakeTimers()
    const createObjectUrl = vi.fn(() => 'blob:ponctuel')
    const revokeObjectUrl = vi.fn()
    vi.stubGlobal('URL', { createObjectURL: createObjectUrl, revokeObjectURL: revokeObjectUrl })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    downloadCsv('ponctuel.csv', 'vehicle_id\n31-113\r\n')

    expect(createObjectUrl).toHaveBeenCalledOnce()
    expect(click).toHaveBeenCalledOnce()
    const anchor = click.mock.instances[0] as HTMLAnchorElement
    expect(anchor.download).toBe('ponctuel.csv')
    expect(anchor.href).toBe('blob:ponctuel')
    vi.runAllTimers()
    expect(revokeObjectUrl).toHaveBeenCalledWith('blob:ponctuel')
  })
})
