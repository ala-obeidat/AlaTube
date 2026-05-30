<#
.SYNOPSIS
  Watch a pending folder for a fresh cookies.txt and rotate if found.

.DESCRIPTION
  Designed to be triggered by Windows Task Scheduler. On each run:
    1. If $PendingPath does not exist, log "no pending cookies, skip" and exit 0.
    2. Else, hand the file to rotate-cookies.ps1.
    3. On rotation success, move the pending file into $ArchiveDir with a
       timestamp suffix so it is not re-rotated.
    4. On rotation failure, leave the pending file in place and exit 1.

  Logs go to $env:LOCALAPPDATA\AlaTube\rotate.log. Cookies are never logged.

.PARAMETER PendingPath
  Path the operator drops a freshly-exported cookies.txt at.
  Default: C:\AlaTube\pending\cookies.txt.

.PARAMETER ArchiveDir
  Directory where rotated files are archived after a successful rotation.
  Default: C:\AlaTube\archive.

.PARAMETER Server
  SSH target for rotate-cookies.ps1. Defaults to $env:ALATUBE_SERVER.

.PARAMETER Key
  SSH identity file. Defaults to $env:ALATUBE_SSH_KEY.

.EXAMPLE
  # Either pass them explicitly each run, or set env vars in your user profile
  # and let install-scheduled-task.ps1 bake them into the task action.
  .\scheduled-rotate.ps1 -Server root@prod.example.com -Key C:\path\to\ssh\key
#>
[CmdletBinding()]
param(
    [string]$PendingPath = 'C:\AlaTube\pending\cookies.txt',
    [string]$ArchiveDir = 'C:\AlaTube\archive',
    [string]$Server = $env:ALATUBE_SERVER,
    [string]$Key = $env:ALATUBE_SSH_KEY
)

$ErrorActionPreference = 'Stop'

$logDir = Join-Path $env:LOCALAPPDATA 'AlaTube'
if (-not (Test-Path $logDir)) { New-Item -ItemType Directory -Force -Path $logDir | Out-Null }
$logFile = Join-Path $logDir 'rotate.log'

function Log {
    param([string]$Level, [string]$Message)
    $line = "{0} [{1}] {2}" -f (Get-Date -Format 'yyyy-MM-ddTHH:mm:ssK'), $Level, $Message
    Add-Content -LiteralPath $logFile -Value $line
}

if (-not (Test-Path -LiteralPath $PendingPath)) {
    Log INFO "no pending cookies at $PendingPath, skip"
    exit 0
}

if (-not $Server -or -not $Key) {
    Log ERROR 'Server and Key are required. Pass -Server/-Key or set ALATUBE_SERVER/ALATUBE_SSH_KEY env vars. Re-run install-scheduled-task.ps1 with the right args to bake them into the task.'
    exit 1
}

if (-not (Test-Path -LiteralPath $ArchiveDir)) {
    New-Item -ItemType Directory -Force -Path $ArchiveDir | Out-Null
}

$here = Split-Path -Parent $PSCommandPath
$rotate = Join-Path $here 'rotate-cookies.ps1'
if (-not (Test-Path $rotate)) {
    Log ERROR "rotate-cookies.ps1 not found at $rotate"
    exit 1
}

Log INFO "found pending file at $PendingPath, running rotate-cookies.ps1"
$startedAt = Get-Date
try {
    & powershell -NoProfile -ExecutionPolicy Bypass -File $rotate -Path $PendingPath -Server $Server -Key $Key 2>&1 |
        ForEach-Object { Log INFO "rotate: $_" }
    $exit = $LASTEXITCODE
} catch {
    Log ERROR "rotate threw: $_"
    exit 1
}

if ($exit -ne 0) {
    Log ERROR "rotation reported exit $exit, leaving pending file in place"
    exit 1
}

$stamp = (Get-Date -Format 'yyyyMMddTHHmmssZ')
$dest = Join-Path $ArchiveDir ("cookies-$stamp.txt")
Move-Item -LiteralPath $PendingPath -Destination $dest
$dur = ((Get-Date) - $startedAt).TotalSeconds
Log INFO ("rotation OK in {0:N1}s, archived to {1}" -f $dur, $dest)
