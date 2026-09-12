$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$required = @(
    'go.mod',
    '.env.example',
    'README.md',
    'Dockerfile.api',
    'Dockerfile.ingester',
    'Dockerfile.matcher',
    'Dockerfile.web',
    'deploy/compose/docker-compose.yml',
    'scripts/smoke.ps1'
)
foreach ($path in $required) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Fichier requis absent: $path"
    }
}

docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./...
if ($LASTEXITCODE -ne 0) {
    throw "Les tests Go ont échoué avec le code $LASTEXITCODE"
}

docker compose -f deploy/compose/docker-compose.yml config | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "La configuration Compose est invalide"
}

if (-not (Test-Path -LiteralPath 'web/node_modules')) {
    throw 'Installe les dépendances web avec npm --prefix web ci avant ce contrôle.'
}
npm --prefix web run test -- --run
npm --prefix web run typecheck
npm --prefix web run build
