export type DelayFilter = 'all' | 'on-time' | 'late'

export interface VehicleFilterState {
  search: string
  route: string
  delay: DelayFilter
}

const delayFilters = new Set<DelayFilter>(['all', 'on-time', 'late'])

export function readVehicleFilters(query: string = window.location.search): VehicleFilterState {
  const params = new URLSearchParams(query)
  const delay = params.get('delay')

  return {
    search: params.get('q') ?? '',
    route: params.get('route') ?? '',
    delay: delay && delayFilters.has(delay as DelayFilter) ? delay as DelayFilter : 'all',
  }
}

export function buildVehicleFiltersUrl(state: VehicleFilterState, currentUrl: string = window.location.href): string {
  const url = new URL(currentUrl)
  url.searchParams.delete('q')
  url.searchParams.delete('route')
  url.searchParams.delete('delay')

  const search = state.search.trim()
  if (search) url.searchParams.set('q', search)
  if (state.route) url.searchParams.set('route', state.route)
  if (state.delay !== 'all') url.searchParams.set('delay', state.delay)

  return `${url.pathname}${url.search}${url.hash}`
}

export function syncVehicleFiltersToUrl(state: VehicleFilterState): void {
  window.history.replaceState(null, '', buildVehicleFiltersUrl(state))
}
