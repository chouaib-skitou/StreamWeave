$ErrorActionPreference = 'Stop'
$coverageProfile = Join-Path (Get-Location) 'identity-coverage.out'
& go test ./apps/identity/internal/application/identity "-coverprofile=$coverageProfile"
$total = (& go tool cover -func "$coverageProfile" | Select-String '^total:').ToString()
$match = [regex]::Match($total, '(\d+\.\d+)%')
if (-not $match.Success) { throw "Could not read coverage total: $total" }
$coverage = [double]$match.Groups[1].Value
Write-Output ("Identity application coverage: {0:N1}%" -f $coverage)
if ($coverage -lt 85.0) { throw "Identity application coverage is below the required 85% threshold." }
