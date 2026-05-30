<#
.SYNOPSIS
  Register the AlaTube cookie rotation task in Windows Task Scheduler.

.DESCRIPTION
  Creates a daily scheduled task named "AlaTube Cookie Rotation" that runs
  scheduled-rotate.ps1. The script no-ops if there is no pending file, so
  daily firing is cheap.

  Run this script once, in an elevated (Administrator) PowerShell prompt.

.PARAMETER Time
  Daily fire time (local). Default 09:30.

.PARAMETER TaskName
  Task Scheduler entry name. Default 'AlaTube Cookie Rotation'.

.PARAMETER Server
  SSH target like user@host. Baked into the task action so the scheduled
  task does not depend on env vars at fire time. Defaults to
  $env:ALATUBE_SERVER.

.PARAMETER Key
  SSH identity file. Baked into the task action. Defaults to
  $env:ALATUBE_SSH_KEY.

.EXAMPLE
  # In an elevated PowerShell prompt:
  cd D:\Code\AlaTube
  .\scripts\install-scheduled-task.ps1 -Server root@prod.example.com -Key C:\path\to\ssh\key
#>
[CmdletBinding()]
param(
    [string]$Time = '09:30',
    [string]$TaskName = 'AlaTube Cookie Rotation',
    [string]$Server = $env:ALATUBE_SERVER,
    [string]$Key = $env:ALATUBE_SSH_KEY
)

$ErrorActionPreference = 'Stop'

if (-not $Server) { throw 'Server is required. Pass -Server or set $env:ALATUBE_SERVER.' }
if (-not $Key) { throw 'SSH key is required. Pass -Key or set $env:ALATUBE_SSH_KEY.' }
if (-not (Test-Path $Key)) { throw "SSH key not found: $Key" }

$here = Split-Path -Parent $PSCommandPath
$scheduled = Join-Path $here 'scheduled-rotate.ps1'
if (-not (Test-Path $scheduled)) { throw "Missing $scheduled" }

# Ensure the pending drop folder exists so the user has somewhere to put files.
$pendingDir = 'C:\AlaTube\pending'
$archiveDir = 'C:\AlaTube\archive'
foreach ($d in @($pendingDir, $archiveDir)) {
    if (-not (Test-Path $d)) { New-Item -ItemType Directory -Force -Path $d | Out-Null }
}

$action = New-ScheduledTaskAction `
    -Execute 'powershell.exe' `
    -Argument "-NoProfile -ExecutionPolicy Bypass -File `"$scheduled`" -Server `"$Server`" -Key `"$Key`""

$trigger = New-ScheduledTaskTrigger -Daily -At $Time

$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 10)

$principal = New-ScheduledTaskPrincipal `
    -UserId "$env:USERDOMAIN\$env:USERNAME" `
    -LogonType Interactive `
    -RunLevel Limited

if (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
}

Register-ScheduledTask `
    -TaskName $TaskName `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -Principal $principal `
    -Description 'Run AlaTube scheduled-rotate.ps1 daily. Picks up a freshly-exported cookies.txt from C:\AlaTube\pending\, rotates it on the prod server, and archives it.' | Out-Null

Get-ScheduledTask -TaskName $TaskName | Select-Object TaskName, State, @{N='NextRun';E={ ($_ | Get-ScheduledTaskInfo).NextRunTime }}

Write-Host ""
Write-Host "Pending drop folder: $pendingDir"
Write-Host "Archive folder:      $archiveDir"
Write-Host "Log file:            $env:LOCALAPPDATA\AlaTube\rotate.log"
Write-Host ""
Write-Host "To rotate, export cookies.txt from a logged-in YouTube tab and save it as:"
Write-Host "  $pendingDir\cookies.txt"
Write-Host "The next daily run will pick it up; or run scheduled-rotate.ps1 directly to do it now."
