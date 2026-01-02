# Lords of Conquest - Client Build Script
# Builds the client executable and packages assets for distribution

$ErrorActionPreference = "Stop"

Write-Host "Building Lords of Conquest client..." -ForegroundColor Cyan

# Clean and create bin directory
if (Test-Path "bin") {
    Remove-Item -Recurse -Force "bin"
}
New-Item -ItemType Directory -Path "bin" | Out-Null

# Build the client
Write-Host "Compiling client..." -ForegroundColor Yellow
go build -o bin/loc.exe ./cmd/client
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
Write-Host "Compiled bin/loc.exe" -ForegroundColor Green

# Create assets directory structure
Write-Host "Copying assets..." -ForegroundColor Yellow
New-Item -ItemType Directory -Path "bin/assets" | Out-Null
New-Item -ItemType Directory -Path "bin/assets/icons" | Out-Null
New-Item -ItemType Directory -Path "bin/assets/sound" | Out-Null

# Copy title screens
$assetSource = "internal/client/assets"
Copy-Item "$assetSource/8-bit-title-screen.gif" "bin/assets/" -ErrorAction SilentlyContinue
Copy-Item "$assetSource/title-screen.png" "bin/assets/" -ErrorAction SilentlyContinue

# Copy icons
Get-ChildItem "$assetSource/icons/*.png" | ForEach-Object {
    Copy-Item $_.FullName "bin/assets/icons/"
    Write-Host "  Copied icons/$($_.Name)" -ForegroundColor Gray
}

# Copy sounds
Get-ChildItem "$assetSource/sound/*.ogg" | ForEach-Object {
    Copy-Item $_.FullName "bin/assets/sound/"
    Write-Host "  Copied sound/$($_.Name)" -ForegroundColor Gray
}
Get-ChildItem "$assetSource/sound/*.mp3" | ForEach-Object {
    Copy-Item $_.FullName "bin/assets/sound/"
    Write-Host "  Copied sound/$($_.Name)" -ForegroundColor Gray
}

Write-Host ""
Write-Host "Build complete!" -ForegroundColor Green
Write-Host "Distribution folder: bin/" -ForegroundColor Cyan
Write-Host ""
Write-Host "Contents:" -ForegroundColor Cyan
Get-ChildItem -Recurse "bin" | ForEach-Object {
    $indent = "  " * ($_.FullName.Split("\").Count - (Get-Location).Path.Split("\").Count - 1)
    Write-Host "$indent$($_.Name)" -ForegroundColor White
}
