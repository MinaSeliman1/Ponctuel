export interface Dashboard {
  eventCount: number
  lastCollectedAt: string | null
  mode: string
  stale: boolean
}

export interface Vehicle {
  vehicleId: string
  routeId: string | null
  tripId: string | null
  latitude: number
  longitude: number
  recordedAt: string
  delaySeconds: number | null
}

export interface ErrorSummary {
  routeId: string | null
  horizonSeconds: number
  sampleCount: number
  meanErrorSeconds: number
  onTimeRate: number
}
