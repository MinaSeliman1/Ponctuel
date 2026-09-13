import { readFile } from 'node:fs/promises'
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

async function routeDashboard(page: Page, options: { qualityUnavailable?: boolean; vehicles?: typeof vehicles } = {}) {
  const requestCounts = { dashboard: 0, vehicles: 0, quality: 0 }
  await page.route('**/query', async (route) => {
    const payload = route.request().postDataJSON() as { query?: string }
    const query = payload.query ?? ''

    if (query.includes('ErrorSummary')) {
      requestCounts.quality += 1
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
      requestCounts.vehicles += 1
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { vehicles: options.vehicles ?? vehicles } }),
      })
      return
    }

    if (query.includes('Dashboard')) {
      requestCounts.dashboard += 1
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { dashboard } }),
      })
      return
    }

    await route.abort()
  })
  return requestCounts
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

test('ouvre les détails d’un autobus depuis la carte au clavier', async ({ page }) => {
  await routeDashboard(page)
  await page.goto('/')

  const marker = page.getByRole('button', { name: 'Autobus 1234', exact: true })
  await expect(marker).toHaveAttribute('tabindex', '0')
  await marker.focus()
  await page.keyboard.press('Enter')

  await expect(page.getByRole('dialog')).toContainText('Autobus 1234')
  await expect(page.getByRole('dialog')).toContainText('trip-51')
  await expect(marker).toHaveAttribute('aria-pressed', 'true')

  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(marker).toBeFocused()

  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('button', { name: 'Fermer le détail de l’autobus 1234' }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(marker).toBeFocused()
})

test('contrôle les couches de la carte depuis le navigateur', async ({ page }) => {
  await routeDashboard(page)
  await page.goto('/')

  const vehicleLayer = page.getByRole('checkbox', { name: 'Positions des autobus' })
  const roadLayer = page.getByRole('checkbox', { name: 'Réseau routier' })
  await expect(vehicleLayer).toBeChecked()
  await expect(roadLayer).toBeChecked()
  await expect(page.getByRole('group', { name: 'Légende des retards' })).toContainText('Retard important')

  await vehicleLayer.uncheck()
  await expect(page.getByRole('button', { name: 'Autobus 1234', exact: true })).toHaveCount(0)
  await roadLayer.uncheck()
  await expect(page.locator('.map-streets')).toHaveCount(0)

  await vehicleLayer.check()
  await roadLayer.check()
  await expect(page.getByRole('button', { name: 'Autobus 1234', exact: true })).toBeVisible()
  await expect(page.locator('.map-streets')).toBeVisible()
})

test('conserve les véhicules quand la qualité est indisponible', async ({ page }) => {
  await routeDashboard(page, { qualityUnavailable: true })
  await page.goto('/')

  await expect(page.getByRole('alert')).toContainText('Qualité indisponible pour le moment.')
  await expect(page.getByRole('heading', { name: 'Véhicules en service' })).toBeVisible()
  await expect(page.getByText('1234', { exact: true })).toBeVisible()
})

test('filtre et pagine les véhicules depuis le navigateur', async ({ page }) => {
  const fleet = [
    ...vehicles,
    ...Array.from({ length: 9 }, (_, index) => ({
      ...vehicles[0],
      vehicleId: `51-${index + 1}`,
      tripId: `trip-51-${index + 1}`,
    })),
    { ...vehicles[0], vehicleId: '9988', routeId: '80', tripId: 'trip-80', delaySeconds: 360 },
  ]
  await routeDashboard(page, { vehicles: fleet })
  await page.goto('/')

  await expect(page.getByText('Affichage de 1 à 10 sur 11 véhicules')).toBeVisible()
  await page.locator('input[type="search"]').fill('1234')
  await expect(page.getByRole('button', { name: 'Filtres actifs : 1' })).toBeVisible()
  await expect(page.getByRole('group', { name: 'Filtres actifs' })).toContainText('Recherche : « 1234 »')
  await page.locator('input[type="search"]').fill('')
  await expect(page.getByRole('button', { name: 'Filtres actifs : 0' })).toBeVisible()
  await page.getByRole('button', { name: 'Filtres actifs : 0' }).click()
  await page.getByRole('combobox', { name: 'Ligne', exact: true }).selectOption('80')

  await expect(page.getByText('9988', { exact: true })).toBeVisible()
  await expect(page.getByText('1234', { exact: true })).not.toBeVisible()

  await page.getByRole('combobox', { name: 'Retard', exact: true }).selectOption('on-time')
  await expect(page.getByText('Aucun autobus ne correspond à cette recherche ou à ces filtres.')).toBeVisible()

  await page.locator('#vehicle-filters').getByRole('button', { name: 'Réinitialiser', exact: true }).click()
  await page.getByRole('button', { name: 'Page suivante' }).click()
  await expect(page.getByText('Page 2 sur 2')).toBeVisible()
  await expect(page.getByText('9988', { exact: true })).toBeVisible()
})

test('trie les véhicules et remet la pagination à zéro', async ({ page }) => {
  const fleet = [
    ...vehicles,
    ...Array.from({ length: 9 }, (_, index) => ({
      ...vehicles[0],
      vehicleId: `51-${index + 1}`,
      tripId: `trip-51-${index + 1}`,
    })),
    { ...vehicles[0], vehicleId: '9988', routeId: '80', tripId: 'trip-80', delaySeconds: 360 },
  ]
  await routeDashboard(page, { vehicles: fleet })
  await page.goto('/')

  await page.getByRole('button', { name: 'Page suivante' }).click()
  await expect(page.getByText('Page 2 sur 2')).toBeVisible()
  await page.getByRole('combobox', { name: 'Trier', exact: true }).selectOption('delay')

  await expect(page.getByText('Page 1 sur 2')).toBeVisible()
  await expect(page.getByRole('region', { name: 'Véhicules en service' }).locator('tbody tr').first().getByText('9988', { exact: true })).toBeVisible()
})

test('restaure les filtres et le tri avec retour navigateur', async ({ page }) => {
  const fleet = [
    vehicles[0],
    { ...vehicles[0], vehicleId: '9988', routeId: '80', tripId: 'trip-80', delaySeconds: 360 },
  ]
  await routeDashboard(page, { vehicles: fleet })
  await page.goto('/')

  await page.evaluate(() => history.pushState(null, '', '/?route=80&sort=delay'))
  await page.dispatchEvent('body', 'popstate')
  await expect(page).toHaveURL(/route=80&sort=delay/)
  await expect(page.getByText('9988', { exact: true })).toBeVisible()
  await expect(page.getByText('1234', { exact: true })).not.toBeVisible()

  await page.evaluate(() => history.pushState(null, '', '/?route=51'))
  await page.goBack()
  await expect(page).toHaveURL(/route=80&sort=delay/)
  await page.getByRole('button', { name: 'Filtres actifs : 1' }).click()
  await expect(page.getByRole('combobox', { name: 'Ligne', exact: true })).toHaveValue('80')
  await expect(page.getByRole('combobox', { name: 'Trier', exact: true })).toHaveValue('delay')
})

test('actualise les données depuis le navigateur', async ({ page }) => {
  const requestCounts = await routeDashboard(page)
  await page.goto('/')

  await expect.poll(() => requestCounts.dashboard).toBe(1)
  await expect(page.getByRole('button', { name: 'Actualiser les données' })).toBeEnabled()

  await page.getByRole('button', { name: 'Actualiser les données' }).click()
  await expect.poll(() => requestCounts.dashboard).toBe(2)
  await expect.poll(() => requestCounts.vehicles).toBe(2)
  await expect.poll(() => requestCounts.quality).toBe(2)
  await expect(page.getByText('1234', { exact: true })).toBeVisible()
})

test('permet de suspendre le rafraîchissement automatique', async ({ page }) => {
  const requestCounts = await routeDashboard(page)
  await page.goto('/')

  await page.getByRole('button', { name: 'Préférences' }).click()
  await expect(page.getByRole('dialog', { name: 'Préférences' })).toBeVisible()
  const autoRefresh = page.getByRole('checkbox', { name: 'Actualisation automatique' })
  await expect(autoRefresh).toBeChecked()
  await autoRefresh.uncheck()

  await expect(page.getByText('Actualisation automatique suspendue')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Actualiser les données' })).toBeEnabled()
  await page.getByRole('button', { name: 'Actualiser les données' }).click()
  await expect.poll(() => requestCounts.dashboard).toBe(2)
  await expect.poll(() => requestCounts.vehicles).toBe(2)
  await expect.poll(() => requestCounts.quality).toBe(2)
})

test('gère les panneaux flottants au clavier', async ({ page }) => {
  await routeDashboard(page)
  await page.goto('/')

  const preferencesButton = page.getByRole('button', { name: 'Préférences' })
  await preferencesButton.click()
  const autoRefresh = page.getByRole('checkbox', { name: 'Actualisation automatique' })
  await expect(autoRefresh).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog', { name: 'Préférences' })).toBeHidden()
  await expect(preferencesButton).toBeFocused()

  const filtersButton = page.getByRole('button', { name: 'Filtres actifs : 0' })
  await filtersButton.click()
  const routeFilter = page.getByRole('combobox', { name: 'Ligne', exact: true })
  await expect(routeFilter).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(page.locator('#vehicle-filters')).toBeHidden()
  await expect(filtersButton).toBeFocused()
})

test('télécharge un CSV limité aux véhicules filtrés', async ({ page }) => {
  const fleet = [
    vehicles[0],
    { ...vehicles[0], vehicleId: '9988', routeId: '80', tripId: 'trip-80' },
  ]
  await routeDashboard(page, { vehicles: fleet })
  await page.goto('/')

  await page.getByRole('button', { name: 'Filtres actifs : 0' }).click()
  await page.getByRole('combobox', { name: 'Ligne', exact: true }).selectOption('80')

  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Exporter les véhicules en CSV' }).click()
  const download = await downloadPromise
  expect(download.suggestedFilename()).toMatch(/^ponctuel-vehicules-[0-9TZ]+\.csv$/)
  const downloadPath = await download.path()
  expect(downloadPath).not.toBeNull()
  const content = await readFile(downloadPath!, 'utf8')

  expect(content).toContain('80,9988,trip-80')
  expect(content).not.toContain('51,1234,trip-51')
})

test('restaure et partage les filtres depuis l’URL', async ({ page }) => {
  const fleet = [
    vehicles[0],
    { ...vehicles[0], vehicleId: '9988', routeId: '80', tripId: 'trip-80', delaySeconds: 360 },
  ]
  await routeDashboard(page, { vehicles: fleet })
  await page.goto('/?route=80&delay=late&sort=delay')

  await expect(page.getByText('9988', { exact: true })).toBeVisible()
  await expect(page.getByText('1234', { exact: true })).not.toBeVisible()
  await page.getByRole('button', { name: 'Filtres actifs : 2' }).click()
  await expect(page.getByRole('combobox', { name: 'Ligne', exact: true })).toHaveValue('80')
  await expect(page.getByRole('combobox', { name: 'Retard', exact: true })).toHaveValue('late')
  await expect(page.getByRole('combobox', { name: 'Trier', exact: true })).toHaveValue('delay')

  await page.locator('input[type="search"]').fill('9988')
  await expect(page).toHaveURL(/q=9988&route=80&delay=late&sort=delay/)
})

test('copie le lien absolu des filtres actifs', async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: async (value: string) => {
          ;(window as Window & { __copiedVehicleLink?: string }).__copiedVehicleLink = value
        },
      },
    })
  })
  await routeDashboard(page, { vehicles: [vehicles[0], { ...vehicles[0], vehicleId: '9988', routeId: '80', delaySeconds: 360 }] })
  await page.goto('/?route=80&delay=late&sort=delay')

  await page.getByRole('button', { name: 'Copier le lien des filtres' }).click()
  await expect(page.getByRole('status')).toContainText('Le lien des filtres a été copié.')
  await expect.poll(() => page.evaluate(() => (window as Window & { __copiedVehicleLink?: string }).__copiedVehicleLink))
    .toBe('http://127.0.0.1:4173/?route=80&delay=late&sort=delay')
})

test('supprime les filtres actifs depuis le résumé visible', async ({ page }) => {
  const fleet = [
    vehicles[0],
    { ...vehicles[0], vehicleId: '9988', routeId: '80', tripId: 'trip-80', delaySeconds: 360 },
  ]
  await routeDashboard(page, { vehicles: fleet })
  await page.goto('/?q=bus&route=80&delay=late')

  await expect(page.getByRole('group', { name: 'Filtres actifs' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Supprimer le filtre de ligne 80' })).toBeVisible()
  await page.getByRole('button', { name: 'Supprimer le filtre de ligne 80' }).click()
  await expect(page).toHaveURL(/q=bus&delay=late/)
  await expect(page.getByRole('button', { name: 'Supprimer le filtre de ligne 80' })).toHaveCount(0)

  await page.getByRole('button', { name: 'Réinitialiser tous les filtres' }).click()
  await expect(page).toHaveURL('http://127.0.0.1:4173/')
  await expect(page.getByRole('group', { name: 'Filtres actifs' })).toHaveCount(0)
})
