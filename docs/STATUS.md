# Lords of Conquest - Current Status

## ✅ Completed

### Infrastructure
- **Server**: WebSocket server with SQLite persistence
- **Client**: Ebitengine-based client with multiple scenes
- **Database**: Full schema with player tokens, games, game state
- **Protocol**: Complete message types for lobby and gameplay
- **Maps**: Grid-based system with lake filling and adjacency computation

### Features Working
1. **Player Connection**
   - Token-based auth (no accounts)
   - Local config storage
   - Reconnection support

2. **Game Lobby**
   - Create games (public/private)
   - Join by code (format: `XXXX-XXXX`)
   - Add AI players
   - Ready states
   - Start game

3. **Map System**
   - JSON map format (grid-based)
   - Lake filling (loose rule)
   - Flood-fill water bodies
   - Adjacency computation
   - 1 test map (8 territories)

### Code Organization
```
lords-of-conquest/
├── cmd/
│   ├── server/main.go          # Server entry point
│   └── client/main.go          # Client entry point
├── internal/
│   ├── game/                   # Core game logic
│   │   ├── state.go            # GameState
│   │   ├── player.go           # Player
│   │   ├── territory.go        # Territory
│   │   ├── resources.go        # Resources & costs
│   │   ├── combat.go           # Combat system
│   │   ├── phases.go           # Phase management
│   │   ├── initialize.go       # NEW: Game initialization
│   │   └── selection.go        # NEW: Territory selection
│   ├── server/
│   │   ├── server.go           # WebSocket server & hub
│   │   └── handlers.go         # Message handlers
│   ├── client/
│   │   ├── client.go           # Main game struct
│   │   ├── network.go          # WebSocket client
│   │   ├── config.go           # Local config
│   │   ├── ui.go               # UI components
│   │   └── scenes.go           # Game scenes
│   ├── database/               # SQLite persistence
│   │   ├── database.go
│   │   ├── games.go
│   │   └── players.go
│   └── protocol/               # Network messages
│       ├── messages.go
│       └── payloads.go
└── pkg/maps/                   # Map system
    ├── types.go                # Map & Territory types
    ├── process.go              # Lake fill & adjacency
    ├── loader.go               # Load from JSON
    ├── debug.go                # Debug visualization
    └── data/test.json          # Test map
```

---

## 🚧 In Progress: Territory Selection

### What's Built
- `internal/game/initialize.go` - Creates GameState from map
- `internal/game/selection.go` - Territory selection logic
- Handler stubs in `internal/server/handlers.go`

### What's Needed

1. **Server Side**
   - [ ] Complete `initializeGameState()` - convert map to game state
   - [ ] Complete `handleSelectTerritory()` - process selections
   - [ ] Complete `handlePlaceStockpile()` - first production phase
   - [ ] Complete `broadcastGameState()` - send state to clients

2. **Client Side**  
   - [ ] Create gameplay scene
   - [ ] Render the map (territories with colors)
   - [ ] Handle click to select territory
   - [ ] Show whose turn it is
   - [ ] Show available territories

3. **Protocol Updates**
   - [ ] Add map data to `GameStartedPayload`
   - [ ] Create `GameStatePayload` with full state
   - [ ] Create `TerritorySelectedPayload` for updates

---

## 📋 Next Steps

### Phase 1: Complete Territory Selection
1. Implement server-side game state initialization
2. Add map data serialization for client
3. Create basic map renderer on client
4. Handle territory selection clicks
5. Test full territory selection flow

### Phase 2: Production & Stockpile Placement
1. Process resource production
2. Handle stockpile placement UI
3. Display resource counts

### Phase 3: Development Phase
1. Build cities, weapons, boats
2. Show costs and availability

### Phase 4: Combat
1. Attack planning UI
2. Combat resolution
3. Unit movement

---

## 🎯 Current Goal

**Get territory selection working end-to-end:**
- Players can see the map
- Take turns clicking territories to claim them
- See other players' territories in their colors
- Move to production phase when done

This will be the first playable interaction!

