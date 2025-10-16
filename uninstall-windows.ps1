#Requires -RunAsAdministrator

<#
.SYNOPSIS
    Uninstalls data-splitter from Windows
.DESCRIPTION
    This script removes the data-splitter binary and cleans up the PATH
.PARAMETER InstallDir
    Installation directory to remove (default: C:\Program Files\data-splitter)
.EXAMPLE
    .\uninstall-windows.ps1
    .\uninstall-windows.ps1 -InstallDir "C:\Program Files\data-splitter"
#>

param(
    [string]$InstallDir = "$env:ProgramFiles\data-splitter"
)

Write-Host "🗑️  Uninstalling data-splitter from Windows..." -ForegroundColor Yellow

# Check if installation exists
if (!(Test-Path $InstallDir)) {
    Write-Host "ℹ️  Installation not found at: $InstallDir" -ForegroundColor Cyan
    Write-Host "ℹ️  Checking user directory..." -ForegroundColor Cyan

    $userDir = "$env:USERPROFILE\bin"
    if (Test-Path $userDir) {
        $InstallDir = $userDir
        Write-Host "📁 Found user installation at: $InstallDir" -ForegroundColor Green
    } else {
        Write-Host "❌ No installation found. Nothing to uninstall." -ForegroundColor Red
        exit 0
    }
}

# Remove binary
$binaryPath = Join-Path $InstallDir "data-splitter.exe"
if (Test-Path $binaryPath) {
    Write-Host "🗑️  Removing binary: $binaryPath" -ForegroundColor Yellow
    Remove-Item $binaryPath -Force
} else {
    Write-Host "⚠️  Binary not found: $binaryPath" -ForegroundColor Yellow
}

# Remove directory if empty
$dirContents = Get-ChildItem $InstallDir -ErrorAction SilentlyContinue
if ($dirContents.Count -eq 0) {
    Write-Host "🗑️  Removing empty directory: $InstallDir" -ForegroundColor Yellow
    Remove-Item $InstallDir -Force -Recurse
} else {
    Write-Host "⚠️  Directory not empty, keeping: $InstallDir" -ForegroundColor Yellow
}

# Remove from PATH
$currentPath = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
$userPath = [System.Environment]::GetEnvironmentVariable("Path", "User")

$pathUpdated = $false

if ($currentPath -like "*$InstallDir*") {
    Write-Host "🔗 Removing from system PATH..." -ForegroundColor Yellow
    $newPath = ($currentPath -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
    [System.Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
    $pathUpdated = $true
}

if ($userPath -like "*$InstallDir*") {
    Write-Host "🔗 Removing from user PATH..." -ForegroundColor Yellow
    $newPath = ($userPath -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
    [System.Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $pathUpdated = $true
}

if ($pathUpdated) {
    # Refresh PATH in current session
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    Write-Host "✅ PATH updated" -ForegroundColor Green
} else {
    Write-Host "ℹ️  Not found in PATH" -ForegroundColor Cyan
}

Write-Host "✅ Uninstallation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Note: Your config files and logs remain untouched." -ForegroundColor Cyan