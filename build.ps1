# Lords of Conquest - Windows Build Script
# Builds client and server executables with assets for distribution

$ErrorActionPreference = "Stop"

$VERSION = "1.0.0"
$APP_NAME = "lords-of-conquest"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Lords of Conquest - Windows Build" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Clean and create build directory
if (Test-Path "build") {
    Remove-Item -Recurse -Force "build"
}
New-Item -ItemType Directory -Path "build" | Out-Null

# Build the client
Write-Host "Building client..." -ForegroundColor Yellow
go build -o "build/${APP_NAME}-client.exe" ./cmd/client
if ($LASTEXITCODE -ne 0) {
    Write-Host "Client build failed!" -ForegroundColor Red
    exit 1
}
Write-Host "  Created build/${APP_NAME}-client.exe" -ForegroundColor Green

# Build the server
Write-Host "Building server..." -ForegroundColor Yellow
go build -o "build/${APP_NAME}-server.exe" ./cmd/server
if ($LASTEXITCODE -ne 0) {
    Write-Host "Server build failed!" -ForegroundColor Red
    exit 1
}
Write-Host "  Created build/${APP_NAME}-server.exe" -ForegroundColor Green

# Create distribution folder
$distFolder = "build/${APP_NAME}-windows-v${VERSION}"
Write-Host ""
Write-Host "Creating distribution package..." -ForegroundColor Yellow

New-Item -ItemType Directory -Path $distFolder | Out-Null
New-Item -ItemType Directory -Path "$distFolder/assets" | Out-Null
New-Item -ItemType Directory -Path "$distFolder/assets/icons" | Out-Null
New-Item -ItemType Directory -Path "$distFolder/assets/sound" | Out-Null

# Copy executables
Copy-Item "build/${APP_NAME}-client.exe" "$distFolder/"
Copy-Item "build/${APP_NAME}-server.exe" "$distFolder/"

# Copy assets
$assetSource = "internal/client/assets"

# Copy title screens
Copy-Item "$assetSource/8-bit-title-screen.gif" "$distFolder/assets/" -ErrorAction SilentlyContinue
Copy-Item "$assetSource/title-screen.png" "$distFolder/assets/" -ErrorAction SilentlyContinue

# Copy icons
if (Test-Path "$assetSource/icons") {
    Get-ChildItem "$assetSource/icons/*.png" -ErrorAction SilentlyContinue | ForEach-Object {
        Copy-Item $_.FullName "$distFolder/assets/icons/"
    }
}

# Copy sounds
if (Test-Path "$assetSource/sound") {
    Get-ChildItem "$assetSource/sound/*.ogg" -ErrorAction SilentlyContinue | ForEach-Object {
        Copy-Item $_.FullName "$distFolder/assets/sound/"
    }
    Get-ChildItem "$assetSource/sound/*.mp3" -ErrorAction SilentlyContinue | ForEach-Object {
        Copy-Item $_.FullName "$distFolder/assets/sound/"
    }
}

# Create README for the distribution
@"
Lords of Conquest - Windows
===========================

Client:
  Double-click lords-of-conquest-client.exe to play.

Server:
  Run lords-of-conquest-server.exe to host a game server.
  Default port: 8080
  
  Usage: lords-of-conquest-server.exe [-port PORT] [-db DATABASE_FILE]

Version: $VERSION
"@ | Out-File -FilePath "$distFolder/README.txt" -Encoding UTF8

# Create ZIP file
Write-Host "Creating ZIP archive..." -ForegroundColor Yellow
Compress-Archive -Path $distFolder -DestinationPath "build/${APP_NAME}-windows-v${VERSION}.zip" -Force

Write-Host ""
Write-Host "=========================================" -ForegroundColor Green
Write-Host "Build complete!" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Distribution file:" -ForegroundColor Cyan
Write-Host "  build/${APP_NAME}-windows-v${VERSION}.zip" -ForegroundColor White
Write-Host ""
Write-Host "Contents:" -ForegroundColor Cyan
Write-Host "  - ${APP_NAME}-client.exe (Game client)" -ForegroundColor White
Write-Host "  - ${APP_NAME}-server.exe (Game server)" -ForegroundColor White
Write-Host "  - assets/ (Game assets)" -ForegroundColor White
Write-Host ""
