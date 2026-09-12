[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^https://')]
    [string]$Url,

    [Parameter(Mandatory = $true)]
    [string]$Destination,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9a-fA-F]{64}$')]
    [string]$ExpectedSha256
)

$ErrorActionPreference = 'Stop'
$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$dataRoot = [IO.Path]::GetFullPath((Join-Path $repoRoot 'data'))
$dataRootPrefix = $dataRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
$destinationPath = [IO.Path]::GetFullPath([IO.Path]::Combine($repoRoot, $Destination))

if (-not $destinationPath.StartsWith($dataRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Destination must stay under the repository data directory."
}

$destinationDirectory = Split-Path -Parent $destinationPath
New-Item -ItemType Directory -Force -Path $destinationDirectory | Out-Null
$temporaryPath = Join-Path $destinationDirectory ('.download-' + [Guid]::NewGuid().ToString('N') + '.tmp')

try {
    Invoke-WebRequest -Uri $Url -OutFile $temporaryPath -UseBasicParsing
    $actualSha256 = (Get-FileHash -LiteralPath $temporaryPath -Algorithm SHA256).Hash
    if ($actualSha256 -ine $ExpectedSha256) {
        throw "SHA-256 mismatch. Expected $ExpectedSha256 but received $actualSha256."
    }
    Move-Item -LiteralPath $temporaryPath -Destination $destinationPath -Force
    Write-Output "Downloaded and verified $destinationPath"
}
finally {
    if (Test-Path -LiteralPath $temporaryPath) {
        Remove-Item -LiteralPath $temporaryPath -Force
    }
}
