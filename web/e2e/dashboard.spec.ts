import { expect, test, type Page } from '@playwright/test'

const dashboard = {
  eventCount: 24,
  lastCollectedAt: '2026-09-12T14:00:00Z',
  mode: 'STM',
  stale: false,
}

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
]

const errorSummary = [
  { routeId: '51', horizonSeconds: 300, sampleCount: 4, meanErrorSeconds: 60, onTimeRate: 0.75 },
  { routeId: '80', horizonSeconds: 300, sampleCount: 2, meanErrorSeconds: -30, onTimeRate: 1 },
]

async function routeDashboard(page: Page, options: { qualityUnavailable?: boolean } = {}) {
  await page.route('**/query', async (route) => {
    const payload = route.request().postDataJSON() as { query?: string }
    const query = payload.query ?? ''

    if (query.includes('ErrorSummary')) {
      if (options.qualityUnavailable) {
        await route.fulfill({ status: 503, body: 'unavailable' })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { errorSummary } }),
      })
      return
    }

    if (query.includes('Vehicles')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { vehicles } }),
      })
      return
    }

    if (query.includes('Dashboard')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { dashboard } }),
      })
      return
    }

    await route.abort()
  })
}

test('affiche le dashboard réseau et la qualité dans Chromium', async ({ page }) => {
  await routeDashboard(page)
  await page.goto('/')

  await expect(page.getByRole('heading', { name: 'Qualité des prédictions' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Véhicules en service' })).toBeVisible()
  await expect(page.getByRole('img', { name: 'Erreur moyenne par horizon' })).toBeVisible()
  await expect(page.getByText('+30 s', { exact: true })).toBeVisible()
  await expect(page.getByText('1234', { exact: true })).toBeVisible()
  await expect(page.getByText('Temps réel', { exact: true })).toBeVisible()
})

test('conserve les véhicules quand la qualité est indisponible', async ({ page }) => {
  await routeDashboard(page, { qualityUnavailable: true })
  await page.goto('/')

  await expect(page.getByRole('alert')).toContainText('Qualité indisponible pour le moment.')
  await expect(page.getByRole('heading', { name: 'Véhicules en service' })).toBeVisible()
  await expect(page.getByText('1234', { exact: true })).toBeVisible()
})
