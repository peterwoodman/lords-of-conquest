# Lords of Conquest - Modern Remake

A modern client/server implementation of the classic 1986 strategy game "Lords of Conquest" by Electronic Arts, originally released for the C64 and Apple II.

## Project Overview

### Technology Stack
- **Language**: Go (Golang)
- **Client UI**: Ebitengine (2D game engine for Go)
- **Architecture**: Client/Server with authoritative server
- **Persistence**: SQLite (game state survives server restarts)
- **Networking**: WebSocket for real-time communication
- **Data Format**: JSON for protocol

### Architecture Overview

```
┌─────────────────┐     WebSocket      ┌─────────────────┐
│   Game Client   │◄──────────────────►│   Game Server   │
│   (Ebitengine)  │                    │   (Authoritative)│
└─────────────────┘                    └─────────────────┘
        │                                      │
        ▼                                      ▼
┌─────────────────┐                    ┌─────────────────┐
│  Local Renderer │                    │     SQLite      │
│  Input Handler  │                    │   Game State    │
│  Audio System   │                    │   AI Players    │
└─────────────────┘                    └─────────────────┘
```

### Async Turn-Based Model
- Games persist in SQLite database across server restarts
- Players don't need to stay connected - game pauses between turns
- All players must join before game starts (no late joins)
- When it's your turn, you can connect, take your turn, and disconnect
- Real-time updates via WebSocket when connected

### Simple Player Connection (No Accounts)
- Players enter a display name when connecting
- Server generates a **player token** (stored locally by client)
- Player token identifies you for reconnecting to your games
- **Join codes** for private games (e.g., `ABCD-1234`)
- Public games visible in lobby browser

---

## Game Rules Summary (from Original)

### Players
- 2-7 players (any combination of human/AI)
- AI personalities: Aggressive, Defensive, Passive
- Player order rotates each round

### Victory Conditions
- Build and maintain 3-8 cities (configurable)
- OR eliminate all opponents
- If multiple players reach city goal simultaneously, game continues until one drops below or builds another

### Resources
| Resource | Symbol | Used For |
|----------|--------|----------|
| Coal | �ite | Cities, Weapons |
| Gold | 🪙 | Cities, Weapons, Boats (universal currency) |
| Iron | �ite | Cities, Weapons |
| Timber | 🪵 | Cities, Boats |
| Horses | 🐴 | Special: spread on map, not stockpiled |

### Units & Buildings
| Unit | Cost | Strength | Notes |
|------|------|----------|-------|
| City | 1 Coal + 1 Gold + 1 Iron + 1 Wood (or 4 Gold) | +2 | Doubles adjacent resource production, one per territory |
| Weapon | 1 Coal + 1 Iron (or 2 Gold) | +3 | Move 1 space, one per territory |
| Boat | 3 Wood (or 3 Gold) | +2 | Unlimited water movement, can carry horse + weapon |
| Horse | Auto-produced | +1 | Move 2 spaces, can carry weapon, one per territory |

### Game Phases (per Round)
1. **Production** (25% chance to skip) - Resources generated based on controlled territories
2. **Commerce** - Combined trade and shipment phase; each player trades then ships before the next player goes
   - *Trade sub-phase* (3+ players only, 25% chance to skip) - Negotiate resource exchanges
   - *Shipment sub-phase* (25% chance to skip) - Move stockpile OR one unit
3. **Conquest** - Up to 2 attacks; failing first attack ends phase
4. **Development** - Purchase cities, weapons, boats

### Combat System
- **Attack Strength**: Sum of adjacent owned territories + unit bonuses
- **Defense Strength**: Territory (1) + adjacent owned territories + unit bonuses
- **Tie Resolution**: Configurable (attacker wins / defender wins / 50-50 random)
- **Bringing Forces**: Can bring one unit from non-adjacent territory for bonus
- **Failed Attack Penalty**: All brought-in units destroyed

### Game Levels
| Level | Resources | Boats | Transport |
|-------|-----------|-------|-----------|
| Beginner | Horses + Gold only | No | Stockpile only |
| Intermediate | All 5 | No | Stockpile only |
| Advanced | All 5 | Yes | Stockpile only |
| Expert | All 5 | Yes | Full (units movable) |

### Chance Settings
| Setting | Attack Success | Phase Skipping |
|---------|----------------|----------------|
| Low | Attacker ≥ Defender | Never |
| Medium | Attacker > Defender (ties random) | Sometimes |
| High | Probability-based | Sometimes |

---

## Project Structure

```
lords-of-conquest/
├── cmd/
│   ├── server/           # Server executable
│   │   └── main.go
│   └── client/           # Client executable
│       └── main.go
├── internal/
│   ├── game/             # Core game logic (shared)
│   │   ├── state.go      # Game state representation
│   │   ├── territory.go  # Territory management
│   │   ├── player.go     # Player state
│   │   ├── resources.go  # Resource types and stockpile
│   │   ├── combat.go     # Combat resolution
│   │   ├── phases.go     # Phase management
│   │   └── errors.go     # Game error types
│   ├── server/           # Server-specific code
│   │   ├── server.go     # HTTP + WebSocket server
│   │   ├── hub.go        # WebSocket connection hub
│   │   ├── client.go     # Connected client handling
│   │   ├── lobby.go      # Game lobby management
│   │   ├── handlers.go   # Message handlers
│   │   └── ai/           # AI player implementations
│   │       ├── ai.go     # AI interface
│   │       ├── aggressive.go
│   │       ├── defensive.go
│   │       └── passive.go
│   ├── database/         # SQLite persistence layer
│   │   ├── database.go   # Database connection
│   │   ├── schema.go     # Table definitions
│   │   ├── games.go      # Game CRUD operations
│   │   ├── players.go    # Player token management
│   │   └── migrations.go # Schema migrations
│   ├── client/           # Ebitengine client code
│   │   ├── client.go     # Main game struct
│   │   ├── network.go    # WebSocket client
│   │   ├── config.go     # Local config/token storage
│   │   ├── scenes/       # Game scenes/screens
│   │   │   ├── scene.go      # Scene interface
│   │   │   ├── connect.go    # Server connection screen
│   │   │   ├── lobby.go      # Game browser/creation
│   │   │   ├── waiting.go    # Waiting room
│   │   │   ├── gameplay.go   # Main game screen
│   │   │   └── results.go    # Game over screen
│   │   ├── ui/           # UI components
│   │   │   ├── button.go
│   │   │   ├── textinput.go
│   │   │   ├── list.go
│   │   │   └── panel.go
│   │   └── renderer/     # Rendering subsystem
│   │       ├── renderer.go
│   │       ├── map.go
│   │       └── sprites.go
│   └── protocol/         # Network protocol definitions
│       ├── messages.go   # Message envelope
│       └── payloads.go   # All payload types
├── pkg/
│   └── maps/             # Map definitions
│       ├── loader.go
│       ├── generator.go
│       └── predefined/   # Built-in maps
├── assets/               # Game assets
│   ├── images/
│   ├── sounds/
│   └── fonts/
├── data/                 # Runtime data (gitignored)
│   └── lords.db          # SQLite database
├── docs/
│   ├── overview.md
│   ├── protocol.md
│   ├── ai.md
│   └── original game/
│       └── wiki.md
├── go.mod
├── go.sum
└── README.md
```

---

## Build Phases

### Phase 1: Foundation (Core Game Logic)
**Goal**: Implement the core game rules as a standalone library

- [ ] Define data structures for game state
  - [ ] Territory (position, owner, resources, units)
  - [ ] Player (color, stockpile, territories)
  - [ ] Map (territories, adjacencies, water bodies)
  - [ ] Game settings (level, chance, victory conditions)
- [ ] Implement resource system
  - [ ] Resource types and stockpile management
  - [ ] Production calculation (including city doubling)
  - [ ] Horse spreading mechanics
- [ ] Implement unit system
  - [ ] Unit placement and movement rules
  - [ ] Unit carrying (horse+weapon, boat+horse+weapon)
  - [ ] Building costs and validation
- [ ] Implement combat system
  - [ ] Strength calculation (territories + units)
  - [ ] Combat resolution (all three chance modes)
  - [ ] Bringing forces from non-adjacent territories
  - [ ] Unit capture/destruction on victory/defeat
- [ ] Implement phase management
  - [ ] Phase transitions with skip probability
  - [ ] Action validation per phase
  - [ ] Turn/round management

### Phase 2: Network Protocol
**Goal**: Define and implement client-server communication

- [ ] Design message protocol
  - [ ] Game lobby messages (create, join, leave, settings)
  - [ ] Game flow messages (phase changes, turn notifications)
  - [ ] Action messages (select territory, attack, build, trade)
  - [ ] State synchronization messages
- [ ] Implement WebSocket server
  - [ ] Connection handling
  - [ ] Session management
  - [ ] Message routing
- [ ] Implement client network layer
  - [ ] Connection management
  - [ ] Message sending/receiving
  - [ ] Reconnection handling

### Phase 3: Server Implementation
**Goal**: Create authoritative game server

- [ ] Implement game lobby
  - [ ] Create/join/leave games
  - [ ] Game settings configuration
  - [ ] Player ready states
- [ ] Implement game session
  - [ ] Turn execution with validation
  - [ ] State broadcasting
  - [ ] Timeout handling
- [ ] Implement basic AI
  - [ ] Territory selection strategy
  - [ ] Attack evaluation
  - [ ] Development priorities
  - [ ] Three personality types

### Phase 4: Client - Basic UI
**Goal**: Create functional game client with Ebitengine

- [ ] Set up Ebitengine project structure
- [ ] Implement scene management
  - [ ] Main menu
  - [ ] Lobby screen
  - [ ] Map selection
  - [ ] Game screen
  - [ ] Results screen
- [ ] Implement map rendering
  - [ ] Territory display
  - [ ] Resource icons
  - [ ] Unit sprites
  - [ ] Player colors
- [ ] Implement basic input handling
  - [ ] Territory selection
  - [ ] Menu navigation
  - [ ] Button interactions

### Phase 5: Client - Full Gameplay
**Goal**: Complete all gameplay interactions

- [ ] Territory selection phase UI
- [ ] Production phase display
- [ ] Trade interface (for 3+ player games)
  - [ ] Resource offering
  - [ ] Trade negotiation
- [ ] Shipment phase UI
  - [ ] Unit selection
  - [ ] Movement visualization
- [ ] Combat phase UI
  - [ ] Attack planning
  - [ ] Force calculation display
  - [ ] "Bring forces" interface
  - [ ] Combat animation
  - [ ] Alliance voting (multiplayer)
- [ ] Development phase UI
  - [ ] Building placement
  - [ ] Cost display

### Phase 6: Maps & Content
**Goal**: Implement map system and predefined maps

- [ ] Map data format
- [ ] Map loading/parsing
- [ ] Random map generator
  - [ ] Configurable land/water ratio
  - [ ] Resource placement
- [ ] Predefined maps (recreate 20 from original)
  - [ ] North America
  - [ ] Europe
  - [ ] Mediterranean
  - [ ] China
  - [ ] World
  - [ ] etc.
- [ ] Map editor (stretch goal)

### Phase 7: Polish & Audio
**Goal**: Add visual polish and audio

- [ ] Animations
  - [ ] Combat animations
  - [ ] Unit movement
  - [ ] Building construction
  - [ ] Resource collection
- [ ] Sound effects
  - [ ] Combat sounds
  - [ ] UI feedback
  - [ ] Phase transitions
- [ ] Music
  - [ ] Menu music
  - [ ] Game music
  - [ ] Victory/defeat themes
- [ ] Visual polish
  - [ ] Better sprites
  - [ ] UI theming
  - [ ] Visual feedback

### Phase 8: Advanced Features
**Goal**: Complete feature parity and beyond

- [ ] Save/Load games
- [ ] Game replays
- [ ] Statistics tracking
- [ ] Advanced AI improvements
- [ ] Local hot-seat mode (optional)
- [ ] Spectator mode

---

## Technical Decisions

### Why WebSocket?
- Real-time bidirectional communication
- Low latency for turn-based updates
- Wide client support
- Easy to debug with browser tools

### State Synchronization Strategy
- **Authoritative Server**: All game logic runs on server
- **Client Prediction**: None initially (turn-based doesn't need it)
- **State Broadcasting**: Full state sent after each action
- **Optimization**: Delta updates can be added later

### AI Architecture
- AI runs server-side only
- Same interface as human players
- Configurable "thinking time" for realism
- Three personalities affect:
  - Territory selection priorities
  - Attack/defense balance
  - Risk tolerance
  - Trade behavior

---

## Development Guidelines

### Code Style
- Follow standard Go conventions
- Use `gofmt` and `golint`
- Meaningful package and function names
- Document public interfaces

### Testing Strategy
- Unit tests for game logic
- Integration tests for server
- Manual playtesting for client
- AI vs AI games for balance testing

### Version Control
- Feature branches
- Meaningful commit messages
- Tag releases

---

## Milestones

| Milestone | Description | Target |
|-----------|-------------|--------|
| M1 | Core game logic with tests | Week 2 |
| M2 | Server with lobby, basic AI | Week 4 |
| M3 | Client renders map and basic gameplay | Week 6 |
| M4 | Full gameplay loop (2 players) | Week 8 |
| M5 | Multiplayer (3-7 players) with trade | Week 10 |
| M6 | All maps, polish, audio | Week 12 |
| M7 | Release candidate | Week 14 |

---

## References

- [Original Game Wiki Documentation](./original%20game/wiki.md)
- [Ebitengine Documentation](https://ebitengine.org/)
- [Go WebSocket Library (gorilla/websocket)](https://github.com/gorilla/websocket)

