<#
.SYNOPSIS
  Rotate the YouTube cookies file on the prod AlaTube box.

.DESCRIPTION
  Validates the local cookies.txt is in Netscape format, scp's it to the
  server, then runs scripts/rotate-cookies-remote.sh which atomically
  swaps the live file, backs up the prior copy, and runs yt-dlp once
  against a known video to confirm the new session works.

.PARAMETER Path
  Local path to the new cookies.txt (Netscape format).

.PARAMETER Server
  SSH target. Default root@178.105.197.8.

.PARAMETER Key
  SSH identity file. Default C:\key2\alfajer.

.PARAMETER Video
  YouTube URL used for the post-swap smoke test.

.EXAMPLE
  .\rotate-cookies.ps1 -Path C:\Users\me\Downloads\cookies.txt
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$Path,

    [string]$Server = 'root@178.105.197.8',
    [string]$Key = 'C:\key2\alfajer',
    [string]$Video = 'https://www.youtube.com/watch?v=jNQXAC9IVRw'
)

$ErrorActionPreference = 'Stop'

if (-not (Test-Path $Path)) { throw "File not found: $Path" }

$file = Get-Item $Path
$first = (Get-Content $file -TotalCount 1).TrimStart([char]0xFEFF)
if ($first -notlike '*Netscape HTTP Cookie File*') {
    throw "Not a Netscape cookies file (first line: '$first')"
}

$here = Split-Path -Parent $PSCommandPath
$remoteHelper = Join-Path $here 'rotate-cookies-remote.sh'
if (-not (Test-Path $remoteHelper)) { throw "Missing helper: $remoteHelper" }

Write-Host "Uploading $($file.FullName) ($($file.Length) bytes) -> ${Server}:/etc/alatube/cookies.txt.new"
& scp -i $Key $file.FullName "${Server}:/etc/alatube/cookies.txt.new"
if ($LASTEXITCODE -ne 0) { throw 'scp upload failed' }

Write-Host "Uploading helper -> ${Server}:/tmp/rotate-cookies-remote.sh"
& scp -i $Key $remoteHelper "${Server}:/tmp/rotate-cookies-remote.sh"
if ($LASTEXITCODE -ne 0) { throw 'helper upload failed' }

Write-Host "Running remote helper (swap + smoke test)"
& ssh -i $Key $Server "bash /tmp/rotate-cookies-remote.sh '$Video' ; rc=`$? ; rm -f /tmp/rotate-cookies-remote.sh ; exit `$rc"
if ($LASTEXITCODE -ne 0) {
    Write-Warning "Rotation reported a failure. Investigate journalctl -u alatube before declaring done."
    exit 1
}

Write-Host "Done. Cookies rotated and smoke-tested."
