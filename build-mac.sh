#!/bin/bash
set -e

APP_NAME="Lords of Conquest"
BUNDLE_ID="com.lordsofconquest.game"
VERSION="1.0.0"
BINARY_NAME="lords-of-conquest"

echo "========================================="
echo "Lords of Conquest - macOS Build"
echo "========================================="
echo ""

# Create build directory
mkdir -p build

# Build client (universal binary)
echo "Building client (universal binary)..."
GOOS=darwin GOARCH=amd64 go build -o build/client-intel ./cmd/client
GOOS=darwin GOARCH=arm64 go build -o build/client-arm ./cmd/client
lipo -create -output "build/${BINARY_NAME}-client" build/client-intel build/client-arm
rm build/client-intel build/client-arm
echo "  Created build/${BINARY_NAME}-client"

# Build server (universal binary)
echo "Building server (universal binary)..."
GOOS=darwin GOARCH=amd64 go build -o build/server-intel ./cmd/server
GOOS=darwin GOARCH=arm64 go build -o build/server-arm ./cmd/server
lipo -create -output "build/${BINARY_NAME}-server" build/server-intel build/server-arm
rm build/server-intel build/server-arm
echo "  Created build/${BINARY_NAME}-server"

echo ""
echo "Creating .app bundle for client..."

# Clean and create the .app bundle structure
rm -rf "build/${APP_NAME}.app"
mkdir -p "build/${APP_NAME}.app/Contents/MacOS"
mkdir -p "build/${APP_NAME}.app/Contents/Resources"

# Copy client binary
cp "build/${BINARY_NAME}-client" "build/${APP_NAME}.app/Contents/MacOS/${APP_NAME}"

# Copy assets if they exist
if [ -d "internal/client/assets" ]; then
    cp -r internal/client/assets "build/${APP_NAME}.app/Contents/Resources/"
    echo "  Copied assets folder"
fi

# Create Info.plist
cat > "build/${APP_NAME}.app/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIdentifier</key>
    <string>${BUNDLE_ID}</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleDisplayName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
</dict>
</plist>
EOF

echo "  Created ${APP_NAME}.app"

# Create distribution folder
echo ""
echo "Creating distribution package..."
DIST_FOLDER="build/${BINARY_NAME}-mac-v${VERSION}"
rm -rf "${DIST_FOLDER}"
mkdir -p "${DIST_FOLDER}"

# Copy .app bundle
cp -r "build/${APP_NAME}.app" "${DIST_FOLDER}/"

# Copy server binary
cp "build/${BINARY_NAME}-server" "${DIST_FOLDER}/"

# Create README
cat > "${DIST_FOLDER}/README.txt" << EOF
Lords of Conquest - macOS
=========================

Client:
  Double-click "Lords of Conquest.app" to play.
  
  IMPORTANT: Since this app is not signed, you need to:
    1. Right-click the app -> Open -> Click "Open" in the dialog
    OR
    2. Run in Terminal: xattr -cr "Lords of Conquest.app"

Server:
  Open Terminal and run: ./${BINARY_NAME}-server
  Default port: 8080
  
  Usage: ./${BINARY_NAME}-server [-port PORT] [-db DATABASE_FILE]

Version: ${VERSION}
EOF

# Create ZIP
echo "Creating ZIP archive..."
cd build
rm -f "${BINARY_NAME}-mac-v${VERSION}.zip"
zip -r "${BINARY_NAME}-mac-v${VERSION}.zip" "${BINARY_NAME}-mac-v${VERSION}"
cd ..

echo ""
echo "========================================="
echo "Build complete!"
echo "========================================="
echo ""
echo "Distribution file:"
echo "  build/${BINARY_NAME}-mac-v${VERSION}.zip"
echo ""
echo "Contents:"
echo "  - ${APP_NAME}.app (Game client)"
echo "  - ${BINARY_NAME}-server (Game server)"
echo ""
echo "Note: Users need to bypass Gatekeeper (right-click -> Open)"
echo ""
