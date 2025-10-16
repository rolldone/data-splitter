#Requires -RunAsAdministrator

<#
.SYNOPSIS
    Installs data-splitter globally on Windows
.DESCRIPTION
    This script installs the data-splitter binary to Program Files and adds it to the system PATH
.PARAMETER BinaryPath
    Path to the data-splitter-windows-amd64.exe file
.EXAMPLE
    .\install-windows.ps1 -BinaryPath "C:\Downloads\data-splitter-windows-amd64.exe"
#>

param(
    [Parameter(Mandatory=$true)]
    [string]$BinaryPath,

    [string]$InstallDir = "$env:ProgramFiles\data-splitter"
)

Write-Host "🔨 Installing data-splitter on Windows..." -ForegroundColor Green

# Check if binary exists
if (!(Test-Path $BinaryPath)) {
    Write-Error "Binary not found at: $BinaryPath"
    exit 1
}

# Create installation directory
Write-Host "📁 Creating installation directory: $InstallDir" -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Copy binary
$binaryName = "data-splitter.exe"
$installPath = Join-Path $InstallDir $binaryName
Write-Host "📦 Copying binary to: $installPath" -ForegroundColor Yellow
Copy-Item $BinaryPath $installPath -Force

# Add to system PATH
$currentPath = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
if ($currentPath -notlike "*$InstallDir*") {
    Write-Host "🔗 Adding to system PATH..." -ForegroundColor Yellow
    $newPath = $currentPath + ";$InstallDir"
    [System.Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")

    # Refresh PATH in current session
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
} else {
    Write-Host "ℹ️  Already in PATH" -ForegroundColor Cyan
}

Write-Host "✅ Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "🚀 You can now run 'data-splitter' from anywhere:" -ForegroundColor Cyan
Write-Host "   data-splitter --info" -ForegroundColor White
Write-Host "   data-splitter --config C:\path\to\config.yaml" -ForegroundColor White
Write-Host ""
Write-Host "📁 Make sure config.yaml and .env are in your working directory" -ForegroundColor Yellow