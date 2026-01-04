# Lords of Conquest

A modern client/server remake of the classic 1986 strategy game by Electronic Arts.

![Game Status](https://img.shields.io/badge/status-playable-green)

## About

Lords of Conquest is a turn-based strategy game where 2-8 players compete to conquer territories on a map. Players gather resources, build cities and weapons, form alliances, and wage war to achieve dominance.

This project is a faithful recreation of the original C64/Apple II game with:
- Original game mechanics and rules
- Random map generation with configurable parameters
- AI opponents
- Modern client/server architecture for online multiplayer

## Screenshots

*Coming soon*

## Quick Start

### Prerequisites

- **Go 1.24+** - [Download Go](https://golang.org/dl/)
- **Graphics dependencies** - See [Ebitengine requirements](https://ebitengine.org/en/documents/install.html)
  - **Windows**: No additional dependencies
  - **macOS**: Xcode command line tools (`xcode-select --install`)
  - **Linux**: `sudo apt-get install libgl1-mesa-dev xorg-dev` (Debian/Ubuntu)

### Build & Run

```bash
# Clone the repository
git clone https://github.com/yourusername/lords-of-conquest.git
cd lords-of-conquest

# Build server and client
go build -o server ./cmd/server
go build -o client ./cmd/client

# Start the server (in one terminal)
./server

# Start the client (in another terminal)
./client
```

On Windows, the executables will be `server.exe` and `client.exe`.

## Configuration

### Server Options

```bash
./server [options]

Options:
  -port string    Server port (default "30000")
  -db string      Database path (default "data/lords.db")
```

The server also respects environment variables:
- `PORT` - Server port (used by cloud platforms like Render)
- `DB_PATH` - Database file path

### Client Options

```bash
./client [options]

Options:
  -profile string    Profile name for separate config (useful for testing with multiple players)
```

The client stores its configuration in:
- **Windows**: `%APPDATA%\lords-of-conquest\config.json`
- **macOS**: `~/Library/Application Support/lords-of-conquest/config.json`
- **Linux**: `~/.config/lords-of-conquest/config.json`

## How to Play

### Starting a Game

1. Launch the client and enter your name
2. Enter the server address (default: `localhost:30000`)
3. Click **Create Game** to start a new game, or join an existing one
4. Configure game settings:
   - **Map**: Generate a random map with custom parameters (size, territories, islands, resources)
   - **Players**: 2-8 players (human or AI)
5. Click **Start Game** when ready

### Game Phases

Each game year consists of five phases:

1. **Development** - Build cities, weapons, and boats on your territories
2. **Production** - Territories automatically produce resources
3. **Trade** - Propose resource trades with other players (3+ players)
4. **Shipment** - Move resources using horses and boats
5. **Conquest** - Attack enemy territories to expand your empire

### Victory Conditions

Win by controlling more cities than any other player, or eliminate all opponents.

### Controls

- **Left-click** - Select territories, confirm actions
- **Right-click** - Cancel current action
- **Escape** - Close dialogs, deselect

## Running Multiple Clients Locally

For testing multiplayer locally, use profiles to run multiple clients:

```bash
# Terminal 1 - First player
./client -profile player1

# Terminal 2 - Second player
./client -profile player2
```

Each profile maintains separate configuration (name, server, etc.).

## Deployment

### Cloud Deployment (Render)

The project includes Docker support for cloud deployment:

1. Create a new Web Service on [Render](https://render.com)
2. Connect your GitHub repository
3. Set runtime to **Docker**
4. Configure environment variables for database persistence (optional):
   - `DO_SPACES_ENDPOINT`, `DO_SPACES_BUCKET`, `DO_SPACES_KEY`, `DO_SPACES_SECRET` for S3-compatible storage with Litestream

### Building Release Binaries

Platform-specific build scripts are included:

```powershell
# Windows (PowerShell)
.\build.ps1
```

```bash
# macOS
./build-mac.sh

# Linux
./build-linux.sh
```

## Project Structure

```
lords-of-conquest/
├── cmd/
│   ├── server/         # Server entry point
│   └── client/         # Client entry point
├── internal/
│   ├── game/           # Core game logic (authoritative)
│   ├── server/         # WebSocket server, game hub
│   ├── client/         # Ebitengine UI, network client
│   ├── database/       # SQLite persistence
│   └── protocol/       # Shared message types
├── pkg/
│   └── maps/           # Map generation
├── docs/               # Documentation
├── scripts/            # Deployment scripts
├── Dockerfile          # Docker build for cloud
└── litestream.yml      # Database replication config
```

## Technology Stack

- **Language**: Go 1.24
- **Game Engine**: [Ebitengine](https://ebitengine.org/) v2.9
- **Networking**: WebSocket ([coder/websocket](https://github.com/coder/websocket))
- **Database**: SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite))
- **Architecture**: Authoritative server with thin clients

## Documentation

- [Game Overview & Design](docs/overview.md)
- [Network Protocol](docs/protocol.md)
- [AI System](docs/ai.md)
- [Original Game Reference](docs/original%20game/wiki.md)

## Contributing

Contributions are welcome! This is a hobby project recreating a classic game.

## Legal

Lords of Conquest was originally developed by Eon Productions Ltd. and published by Electronic Arts in 1986. This is a fan remake for educational and nostalgic purposes. No copyright infringement is intended.

## License

MIT License - See [LICENSE](LICENSE) file
