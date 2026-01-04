# Lords of Conquest - Release Script
# Creates a GitHub release with all platform builds
#
# Usage: .\release.ps1 -Version "1.0.0" [-Draft] [-Prerelease]
#
# Prerequisites:
#   - GitHub CLI (gh) installed and authenticated: gh auth login
#   - All platform builds in the build/ folder
#   - Clean git working directory (all changes committed)

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    
    [switch]$Draft,
    [switch]$Prerelease
)

$ErrorActionPreference = "Stop"

$APP_NAME = "lords-of-conquest"
$TAG = "v$Version"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Lords of Conquest - Release $TAG" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Check if gh CLI is installed
try {
    $null = gh --version
} catch {
    Write-Host "ERROR: GitHub CLI (gh) is not installed." -ForegroundColor Red
    Write-Host "Install it from: https://cli.github.com/" -ForegroundColor Yellow
    exit 1
}

# Check if authenticated
$authStatus = gh auth status 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Not authenticated with GitHub CLI." -ForegroundColor Red
    Write-Host "Run: gh auth login" -ForegroundColor Yellow
    exit 1
}

# Check for uncommitted changes
$gitStatus = git status --porcelain
if ($gitStatus) {
    Write-Host "ERROR: You have uncommitted changes." -ForegroundColor Red
    Write-Host "Please commit or stash your changes before creating a release." -ForegroundColor Yellow
    git status --short
    exit 1
}

# Define expected build files
$buildFiles = @(
    "build/${APP_NAME}-windows-v${Version}.zip",
    "build/${APP_NAME}-mac-v${Version}.zip",
    "build/${APP_NAME}-linux-amd64-v${Version}.tar.gz",
    "build/${APP_NAME}-linux-arm64-v${Version}.tar.gz"
)

# Check which build files exist
Write-Host "Checking for build files..." -ForegroundColor Yellow
$existingFiles = @()
$missingFiles = @()

foreach ($file in $buildFiles) {
    if (Test-Path $file) {
        $existingFiles += $file
        Write-Host "  [OK] $file" -ForegroundColor Green
    } else {
        $missingFiles += $file
        Write-Host "  [MISSING] $file" -ForegroundColor Yellow
    }
}

if ($existingFiles.Count -eq 0) {
    Write-Host ""
    Write-Host "ERROR: No build files found!" -ForegroundColor Red
    Write-Host "Run the build scripts first:" -ForegroundColor Yellow
    Write-Host "  Windows: .\build.ps1" -ForegroundColor White
    Write-Host "  macOS:   ./build-mac.sh (on Mac)" -ForegroundColor White
    Write-Host "  Linux:   ./build-linux.sh (on Linux/Mac)" -ForegroundColor White
    exit 1
}

if ($missingFiles.Count -gt 0) {
    Write-Host ""
    Write-Host "WARNING: Some platform builds are missing." -ForegroundColor Yellow
    $continue = Read-Host "Continue with available builds? (y/N)"
    if ($continue -ne "y" -and $continue -ne "Y") {
        Write-Host "Aborted." -ForegroundColor Red
        exit 1
    }
}

# Check if tag already exists
$existingTag = git tag -l $TAG
if ($existingTag) {
    Write-Host ""
    Write-Host "WARNING: Tag $TAG already exists." -ForegroundColor Yellow
    $overwrite = Read-Host "Delete and recreate? (y/N)"
    if ($overwrite -eq "y" -or $overwrite -eq "Y") {
        Write-Host "Deleting existing tag..." -ForegroundColor Yellow
        git tag -d $TAG
        git push origin --delete $TAG 2>$null
    } else {
        Write-Host "Aborted." -ForegroundColor Red
        exit 1
    }
}

# Create and push tag
Write-Host ""
Write-Host "Creating git tag $TAG..." -ForegroundColor Yellow
git tag -a $TAG -m "Release $TAG"
git push origin $TAG

# Build release notes
$releaseNotes = @"
## Lords of Conquest $TAG

### Downloads

| Platform | File |
|----------|------|
| Windows | ``${APP_NAME}-windows-v${Version}.zip`` |
| macOS | ``${APP_NAME}-mac-v${Version}.zip`` |
| Linux (amd64) | ``${APP_NAME}-linux-amd64-v${Version}.tar.gz`` |
| Linux (arm64) | ``${APP_NAME}-linux-arm64-v${Version}.tar.gz`` |

### Installation

**Windows:** Extract the ZIP and run ``${APP_NAME}-client.exe``

**macOS:** Extract the ZIP, then right-click the app → Open → Click "Open" (required for unsigned apps)

**Linux:** Extract with ``tar -xzvf <file>.tar.gz`` and run ``./lords-of-conquest-client``

### Running a Server

Each package includes a server executable. Run it with:
``````
./lords-of-conquest-server -port 8080
``````
"@

# Create the release
Write-Host "Creating GitHub release..." -ForegroundColor Yellow

$ghArgs = @("release", "create", $TAG)
$ghArgs += $existingFiles
$ghArgs += "--title"
$ghArgs += "Lords of Conquest $TAG"
$ghArgs += "--notes"
$ghArgs += $releaseNotes

if ($Draft) {
    $ghArgs += "--draft"
}
if ($Prerelease) {
    $ghArgs += "--prerelease"
}

& gh @ghArgs

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to create release." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=========================================" -ForegroundColor Green
Write-Host "Release $TAG created successfully!" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green
Write-Host ""

# Get the release URL
$releaseUrl = gh release view $TAG --json url --jq ".url"
Write-Host "View your release at:" -ForegroundColor Cyan
Write-Host "  $releaseUrl" -ForegroundColor White
Write-Host ""
