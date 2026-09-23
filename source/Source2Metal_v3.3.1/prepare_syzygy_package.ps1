$ErrorActionPreference = 'Stop'

$url = 'https://github.com/hpoiters/SyzygyCheck/releases/download/v2.2.0/SyzygyCheck_v2.2.0_RELEASE.zip'
$expected = 'd80f8fd70f81b4f06cbd89547a7e346aa08688853b2ce2df25b30e93ccd6cb2c'
$destination = Join-Path $PSScriptRoot 'syzygycheck_package.zip'

Invoke-WebRequest -Uri $url -OutFile $destination
$actual = (Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actual -ne $expected) {
    Remove-Item -LiteralPath $destination -Force
    throw "Downloaded SyzygyCheck package has SHA-256 $actual; expected $expected"
}

Write-Host "Verified SyzygyCheck v2.2.0 package: $destination"
