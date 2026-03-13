param(
    [string]$Source = "..\\data\\itdb.db",
    [string]$TargetDir = "..\\data\\backups"
)

$ErrorActionPreference = "Stop"

$base = Resolve-Path (Join-Path $PSScriptRoot $Source)
$backupRoot = Join-Path $PSScriptRoot $TargetDir

if (!(Test-Path $backupRoot)) {
    New-Item -ItemType Directory -Path $backupRoot | Out-Null
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$target = Join-Path $backupRoot ("itdb-" + $timestamp + ".db")

Copy-Item -Path $base -Destination $target -Force
Write-Host "Backup created: $target"
