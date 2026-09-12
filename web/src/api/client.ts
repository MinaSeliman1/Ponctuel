import type { Dashboard, Vehicle } from '../types'

const endpoint = import.meta.env.VITE_API_URL || '/query'

interface GraphQLResponse<T> {
  data?: T
  errors?: Array<{ message?: string }>
}

async function query<T>(body: string, variables: Record<string, unknown> | undefined, signal?: AbortSignal): Promise<T> {
  const response = await fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query: body, variables }),
    signal,
  })

  if (!response.ok) {
    throw new Error('api-unavailable')
  }

  const payload = (await response.json()) as GraphQLResponse<T>
  if (!payload.data || payload.errors?.length) {
    throw new Error('api-unavailable')
  }
  return payload.data
}

export async function fetchDashboard(signal?: AbortSignal): Promise<Dashboard> {
  const payload = await query<{ dashboard: Dashboard }>(
    `query Dashboard { dashboard { eventCount lastCollectedAt mode stale } }`,
    undefined,
    signal,
  )
  return payload.dashboard
}

export async function fetchVehicles(limit = 500, signal?: AbortSignal): Promise<Vehicle[]> {
  const payload = await query<{ vehicles: Vehicle[] }>(
    `query Vehicles($limit: Int) { vehicles(limit: $limit) { vehicleId routeId tripId latitude longitude recordedAt delaySeconds } }`,
    { limit },
    signal,
  )
  return payload.vehicles
}
