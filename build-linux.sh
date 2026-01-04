#!/bin/bash
set -e

APP_NAME="lords-of-conquest"
VERSION="1.0.0"

echo "========================================="
echo "Lords of Conquest - Linux Build"
echo "========================================="
echo ""

# Create build directory
mkdir -p build

# Build for AMD64 (most common)
echo "Building for Linux amd64..."
GOOS=linux GOARCH=amd64 go build -o "build/${APP_NAME}-client-linux-amd64" ./cmd/client
GOOS=linux GOARCH=amd64 go build -o "build/${APP_NAME}-server-linux-amd64" ./cmd/server
echo "  Created amd64 binaries"

# Build for ARM64 (Raspberry Pi, ARM servers)
echo "Building for Linux arm64..."
GOOS=linux GOARCH=arm64 go build -o "build/${APP_NAME}-client-linux-arm64" ./cmd/client
GOOS=linux GOARCH=arm64 go build -o "build/${APP_NAME}-server-linux-arm64" ./cmd/server
echo "  Created arm64 binaries"

# Create AMD64 distribution
echo ""
echo "Creating amd64 distribution package..."
DIST_AMD64="build/${APP_NAME}-linux-amd64-v${VERSION}"
rm -rf "${DIST_AMD64}"
mkdir -p "${DIST_AMD64}"

cp "build/${APP_NAME}-client-linux-amd64" "${DIST_AMD64}/${APP_NAME}-client"
cp "build/${APP_NAME}-server-linux-amd64" "${DIST_AMD64}/${APP_NAME}-server"
chmod +x "${DIST_AMD64}/${APP_NAME}-client"
chmod +x "${DIST_AMD64}/${APP_NAME}-server"

# Copy assets if they exist
if [ -d "internal/client/assets" ]; then
    cp -r internal/client/assets "${DIST_AMD64}/"
fi

# Create helper scripts
cat > "${DIST_AMD64}/run-client.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
./lords-of-conquest-client
EOF
chmod +x "${DIST_AMD64}/run-client.sh"

cat > "${DIST_AMD64}/run-server.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
./lords-of-conquest-server "$@"
EOF
chmod +x "${DIST_AMD64}/run-server.sh"

# Create README
cat > "${DIST_AMD64}/README.txt" << EOF
Lords of Conquest - Linux (amd64)
=================================

Client:
  ./lords-of-conquest-client
  or
  ./run-client.sh

Server:
  ./lords-of-conquest-server
  or
  ./run-server.sh
  
  Default port: 8080
  Usage: ./lords-of-conquest-server [-port PORT] [-db DATABASE_FILE]

Version: ${VERSION}
EOF

# Create tar.gz for AMD64
cd build
rm -f "${APP_NAME}-linux-amd64-v${VERSION}.tar.gz"
tar -czvf "${APP_NAME}-linux-amd64-v${VERSION}.tar.gz" "${APP_NAME}-linux-amd64-v${VERSION}"
cd ..

# Create ARM64 distribution
echo "Creating arm64 distribution package..."
DIST_ARM64="build/${APP_NAME}-linux-arm64-v${VERSION}"
rm -rf "${DIST_ARM64}"
mkdir -p "${DIST_ARM64}"

cp "build/${APP_NAME}-client-linux-arm64" "${DIST_ARM64}/${APP_NAME}-client"
cp "build/${APP_NAME}-server-linux-arm64" "${DIST_ARM64}/${APP_NAME}-server"
chmod +x "${DIST_ARM64}/${APP_NAME}-client"
chmod +x "${DIST_ARM64}/${APP_NAME}-server"

# Copy assets if they exist
if [ -d "internal/client/assets" ]; then
    cp -r internal/client/assets "${DIST_ARM64}/"
fi

# Create helper scripts
cat > "${DIST_ARM64}/run-client.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
./lords-of-conquest-client
EOF
chmod +x "${DIST_ARM64}/run-client.sh"

cat > "${DIST_ARM64}/run-server.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
./lords-of-conquest-server "$@"
EOF
chmod +x "${DIST_ARM64}/run-server.sh"

# Create README
cat > "${DIST_ARM64}/README.txt" << EOF
Lords of Conquest - Linux (arm64)
=================================

Client:
  ./lords-of-conquest-client
  or
  ./run-client.sh

Server:
  ./lords-of-conquest-server
  or
  ./run-server.sh
  
  Default port: 8080
  Usage: ./lords-of-conquest-server [-port PORT] [-db DATABASE_FILE]

Version: ${VERSION}
EOF

# Create tar.gz for ARM64
cd build
rm -f "${APP_NAME}-linux-arm64-v${VERSION}.tar.gz"
tar -czvf "${APP_NAME}-linux-arm64-v${VERSION}.tar.gz" "${APP_NAME}-linux-arm64-v${VERSION}"
cd ..

echo ""
echo "========================================="
echo "Build complete!"
echo "========================================="
echo ""
echo "Distribution files:"
echo "  build/${APP_NAME}-linux-amd64-v${VERSION}.tar.gz  (Intel/AMD 64-bit)"
echo "  build/${APP_NAME}-linux-arm64-v${VERSION}.tar.gz  (ARM 64-bit)"
echo ""
echo "Each package contains:"
echo "  - ${APP_NAME}-client (Game client)"
echo "  - ${APP_NAME}-server (Game server)"
echo "  - assets/ (Game assets)"
echo "  - run-client.sh / run-server.sh (Helper scripts)"
echo ""
echo "Users extract and run with:"
echo "  tar -xzvf ${APP_NAME}-linux-amd64-v${VERSION}.tar.gz"
echo "  cd ${APP_NAME}-linux-amd64-v${VERSION}"
echo "  ./lords-of-conquest-client"
echo ""
