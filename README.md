# Lords of Conquest

A modern client/server remake of the classic 1986 strategy game by Electronic Arts.

![Game Status](https://img.shields.io/badge/status-in%20development-yellow)

## About

Lords of Conquest is a turn-based strategy game where 2-7 players compete to build and maintain cities across a map of territories. Players gather resources, build armies, and wage war to achieve dominance.

This project is a faithful recreation of the original C64/Apple II game with:
- Identical game mechanics and rules
- All original maps recreated
- AI opponents with three personality types
- Modern client/server architecture for online multiplayer

## Technology

- **Language**: Go
- **Client**: Ebitengine (2D game engine)
- **Networking**: WebSocket
- **Architecture**: Authoritative server with thin clients

## Features

### From the Original
- [x] 2-7 player support (human or AI)
- [ ] Three AI personalities: Aggressive, Defensive, Passive
- [ ] Five resources: Coal, Gold, Iron, Timber, Horses
- [ ] Units: Cities, Weapons, Boats, Horses
- [ ] Full combat system with alliance voting
- [ ] Trading phase for 3+ players
- [ ] 20 predefined maps
- [ ] Random map generation
- [ ] Four game complexity levels
- [ ] Three randomness settings

### Modern Additions
- [ ] Online multiplayer
- [ ] Game lobbies
- [ ] Save/load games
- [ ] Spectator mode
- [ ] Game replays

## Building

### Prerequisites
- Go 1.22+
- For graphics: see [Ebitengine requirements](https://ebitengine.org/en/documents/install.html)

### Build Commands

```bash
# Build the server
go build -o bin/server ./cmd/server

# Build the client
go build -o bin/client ./cmd/client

# Run tests
go test ./...
```

## Running

### Start Server
```bash
./bin/server --port 8080
```

### Start Client
```bash
./bin/client --server localhost:8080
```

## Project Structure

```
lords-of-conquest/
├── cmd/
│   ├── server/         # Server executable
│   └── client/         # Client executable
├── internal/
│   ├── game/           # Core game logic
│   ├── server/         # Server implementation
│   ├── client/         # Client implementation
│   └── protocol/       # Network protocol
├── pkg/
│   └── maps/           # Map definitions
├── assets/             # Images, sounds, fonts
└── docs/               # Documentation
```

## Documentation

- [Overview & Build Plan](docs/overview.md)
- [Network Protocol](docs/protocol.md)
- [AI System](docs/ai.md)
- [Original Game Reference](docs/original%20game/wiki.md)

## Development Status

See [overview.md](docs/overview.md) for the detailed build phases and progress.

### Current Phase: Foundation

We're currently implementing the core game logic.

## Contributing

This is a hobby project recreating a classic game for educational purposes.

## Legal

Lords of Conquest was originally developed by Eon Productions Ltd. and published by Electronic Arts in 1986. This is a fan remake for educational and nostalgic purposes.

## License

MIT License - See LICENSE file
