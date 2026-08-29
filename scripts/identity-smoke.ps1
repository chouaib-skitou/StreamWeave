$ErrorActionPreference = 'Stop'
$base = if ($env:IDENTITY_SMOKE_BASE_URL) { $env:IDENTITY_SMOKE_BASE_URL } else { 'http://localhost:8080' }
$email = "smoke-$([Guid]::NewGuid().ToString('N'))@example.test"
$password = 'correct-horse-battery-staple'
$body = @{ email = $email; password = $password } | ConvertTo-Json
$registration = Invoke-RestMethod -Method Post -Uri "$base/.well-known/register" -ContentType 'application/json' -Body $body
if (-not $registration.id) { throw 'Registration did not return a user identifier' }
$login = Invoke-RestMethod -Method Post -Uri "$base/v1/auth/login" -ContentType 'application/json' -Body $body
if (-not $login.access_token -or -not $login.refresh_token) { throw 'Login did not return credentials' }
$headers = @{ Authorization = "Bearer $($login.access_token)" }
$health = Invoke-RestMethod -Method Get -Uri "$base/health/ready"
if ($health.status -ne 'ready') { throw 'Identity is not ready' }
Write-Output 'Identity smoke test passed.'
