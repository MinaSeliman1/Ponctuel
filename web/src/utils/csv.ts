import type { Vehicle } from '../types'

type CsvValue = string | number | null

const vehicleColumns: Array<{ label: string; value: (vehicle: Vehicle) => CsvValue }> = [
  { label: 'route_id', value: (vehicle) => vehicle.routeId },
  { label: 'vehicle_id', value: (vehicle) => vehicle.vehicleId },
  { label: 'trip_id', value: (vehicle) => vehicle.tripId },
  { label: 'latitude', value: (vehicle) => vehicle.latitude },
  { label: 'longitude', value: (vehicle) => vehicle.longitude },
  { label: 'recorded_at_utc', value: (vehicle) => vehicle.recordedAt },
  { label: 'delay_seconds', value: (vehicle) => vehicle.delaySeconds },
]

function escapeCsvValue(value: CsvValue): string {
  const text = value === null ? '' : String(value)
  return /[",\r\n]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text
}

export function vehiclesToCsv(vehicles: Vehicle[]): string {
  const header = vehicleColumns.map((column) => column.label).join(',')
  const rows = vehicles.map((vehicle) => vehicleColumns
    .map((column) => escapeCsvValue(column.value(vehicle)))
    .join(','))
  return `\uFEFF${[header, ...rows].join('\r\n')}\r\n`
}

export function downloadCsv(filename: string, content: string): void {
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  window.setTimeout(() => {
    link.remove()
    URL.revokeObjectURL(url)
  }, 1_000)
}
