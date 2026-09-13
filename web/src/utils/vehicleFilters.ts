export type DelayFilter = 'all' | 'on-time' | 'late'
export type VehicleSort = 'arrival' | 'route' | 'vehicle' | 'delay'

export interface VehicleFilterState {
  search: string
  route: string
  delay: DelayFilter
  sort?: VehicleSort
}

const delayFilters = new Set<DelayFilter>(['all', 'on-time', 'late'])
const vehicleSorts = new Set<VehicleSort>(['arrival', 'route', 'vehicle', 'delay'])

export function readVehicleFilters(query: string = window.location.search): VehicleFilterState {
  const params = new URLSearchParams(query)
  const delay = params.get('delay')
  const sort = params.get('sort')

  return {
    search: params.get('q') ?? '',
    route: params.get('route') ?? '',
    delay: delay && delayFilters.has(delay as DelayFilter) ? delay as DelayFilter : 'all',
    sort: sort && vehicleSorts.has(sort as VehicleSort) ? sort as VehicleSort : 'arrival',
  }
}

export function buildVehicleFiltersUrl(state: VehicleFilterState, currentUrl: string = window.location.href): string {
  const url = new URL(currentUrl)
  url.searchParams.delete('q')
  url.searchParams.delete('route')
  url.searchParams.delete('delay')
  url.searchParams.delete('sort')

  const search = state.search.trim()
  if (search) url.searchParams.set('q', search)
  if (state.route) url.searchParams.set('route', state.route)
  if (state.delay !== 'all') url.searchParams.set('delay', state.delay)
  if (state.sort && state.sort !== 'arrival') url.searchParams.set('sort', state.sort)

  return `${url.pathname}${url.search}${url.hash}`
}

export function buildVehicleFiltersShareUrl(state: VehicleFilterState, currentUrl: string = window.location.href): string {
  const current = new URL(currentUrl)
  return new URL(buildVehicleFiltersUrl(state, currentUrl), current.origin).toString()
}

export async function copyVehicleFiltersLink(state: VehicleFilterState): Promise<void> {
  if (!navigator.clipboard?.writeText) {
    throw new Error('Clipboard API indisponible')
  }

  await navigator.clipboard.writeText(buildVehicleFiltersShareUrl(state))
}

export function syncVehicleFiltersToUrl(state: VehicleFilterState): void {
  window.history.replaceState(null, '', buildVehicleFiltersUrl(state))
}
