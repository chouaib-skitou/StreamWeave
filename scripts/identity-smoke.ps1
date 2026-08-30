$ErrorActionPreference = 'Stop'
$base = if ($env:IDENTITY_SMOKE_BASE_URL) { $env:IDENTITY_SMOKE_BASE_URL } else { 'http://localhost:8080' }
$mailhog = if ($env:IDENTITY_SMOKE_MAILHOG_URL) { $env:IDENTITY_SMOKE_MAILHOG_URL } else { 'http://localhost:8025' }
$email = "smoke-$([Guid]::NewGuid().ToString('N'))@example.test"
$password = 'correct-horse-battery-staple'
$body = @{ email = $email; password = $password } | ConvertTo-Json
$registration = Invoke-RestMethod -Method Post -Uri "$base/.well-known/register" -ContentType 'application/json' -Body $body
if (-not $registration.id) { throw 'Registration did not return a user identifier' }

$verificationToken = $null
$deadline = (Get-Date).AddSeconds(20)
while ((Get-Date) -lt $deadline -and -not $verificationToken) {
    try {
        $mailbox = Invoke-RestMethod -Method Get -Uri "$mailhog/api/v2/messages?limit=50"
        $message = @($mailbox.items) | Where-Object {
            $recipients = @($_.Content.Headers.To) -join ','
            $recipients -like "*$email*" -and $_.Content.Headers.Subject -like '*Verify*'
        } | Select-Object -First 1
        if ($message) {
            $match = [regex]::Match([string]$message.Content.Body, 'verify-email\?token=([A-Za-z0-9_-]+)')
            if ($match.Success) { $verificationToken = $match.Groups[1].Value }
        }
    } catch {
        # MailHog may still be starting; retry until the bounded deadline.
    }
    if (-not $verificationToken) { Start-Sleep -Milliseconds 250 }
}
if (-not $verificationToken) { throw 'Verification email was not captured by MailHog' }

$verificationBody = @{ token = $verificationToken } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$base/v1/auth/email-verification/confirm" -ContentType 'application/json' -Body $verificationBody | Out-Null
$login = Invoke-RestMethod -Method Post -Uri "$base/v1/auth/login" -ContentType 'application/json' -Body $body
if (-not $login.access_token -or -not $login.refresh_token) { throw 'Login did not return credentials' }
$headers = @{ Authorization = "Bearer $($login.access_token)" }
$sessions = Invoke-RestMethod -Method Get -Uri "$base/v1/auth/sessions" -Headers $headers
if (-not $sessions.sessions -or $sessions.sessions.Count -lt 1) { throw 'Session listing did not return the authenticated session' }
$refreshBody = @{ refresh_token = $login.refresh_token } | ConvertTo-Json
$refreshed = Invoke-RestMethod -Method Post -Uri "$base/v1/auth/refresh" -ContentType 'application/json' -Body $refreshBody
if (-not $refreshed.access_token -or $refreshed.refresh_token -eq $login.refresh_token) { throw 'Refresh token rotation did not occur' }
$health = Invoke-RestMethod -Method Get -Uri "$base/health/ready"
if ($health.status -ne 'ready') { throw 'Identity is not ready' }
Write-Output 'Identity smoke test passed.'
