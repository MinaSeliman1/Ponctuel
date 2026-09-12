$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$required = @('go.mod', '.env.example', 'README.md')
foreach ($path in $required) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Fichier requis absent: $path"
    }
}

docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./...
