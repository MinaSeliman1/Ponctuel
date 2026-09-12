param(
    [string]$ComposeFile = (Join-Path $PSScriptRoot '..\deploy\compose\docker-compose.yml'),
    [string]$ProjectName = 'ponctuel-smoke'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$composePath = (Resolve-Path -LiteralPath $ComposeFile).Path
$composeArgs = @('-p', $ProjectName, '-f', $composePath)

function Invoke-Compose {
    param([Parameter(Mandatory)][string[]]$Arguments)
    $null = & docker compose @composeArgs @Arguments 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose a échoué avec le code $LASTEXITCODE"
    }
}

function Wait-Ready {
    param([Parameter(Mandatory)][string]$Uri)
    for ($attempt = 1; $attempt -le 60; $attempt++) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri $Uri -TimeoutSec 3
            if ($response.StatusCode -eq 200) {
                return
            }
        } catch {
            # Les services démarrent en parallèle; on retente sans afficher la réponse amont.
        }
        Start-Sleep -Seconds 2
    }
    throw "Le service n'est pas prêt après 120 secondes: $Uri"
}

try {
    Invoke-Compose @('down', '--volumes', '--remove-orphans')
    Invoke-Compose @('up', '--build', '--detach')
    Wait-Ready -Uri 'http://127.0.0.1:8080/readyz'
    Wait-Ready -Uri 'http://127.0.0.1:3000/'

    $query = @{
        query = 'query Smoke { dashboard { eventCount mode stale } vehicles(limit: 20) { vehicleId routeId latitude longitude } }'
    } | ConvertTo-Json -Compress
    $result = Invoke-RestMethod -Method Post -Uri 'http://127.0.0.1:8080/query' -ContentType 'application/json' -Body $query
    $resultText = $result | ConvertTo-Json -Depth 12

    if ($resultText -match 'STM_API_KEY') {
        throw 'Le résultat du smoke test contient le nom de la clé STM.'
    }
    if (($result.PSObject.Properties.Name -contains 'errors') -and @($result.errors).Count -gt 0) {
        throw 'GraphQL a retourné une erreur pendant le smoke test.'
    }

    $eventCount = [int]$result.data.dashboard.eventCount
    $vehicleCount = @($result.data.vehicles).Count
    if ($eventCount -le 0) {
        throw "Le nombre d'événements doit être positif; valeur: $eventCount"
    }
    if ($vehicleCount -le 0) {
        throw "Le nombre de véhicules doit être positif; valeur: $vehicleCount"
    }

    Write-Host "Smoke test Ponctuel réussi: $eventCount événements, $vehicleCount véhicules."
} finally {
    Invoke-Compose @('down', '--volumes', '--remove-orphans')
}
