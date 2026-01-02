// Package game contains the core game logic for Lords of Conquest.
// This package is shared between client and server.
package game

// GameState represents the complete state of a game.
type GameState struct {
	ID                string                 `json:"id"`
	Settings          Settings               `json:"settings"`
	Round             int                    `json:"round"`
	Phase             Phase                  `json:"phase"`
	CurrentPlayerID   string                 `json:"currentPlayerId"`
	PlayerOrder       []string               `json:"playerOrder"`
	Players           map[string]*Player     `json:"players"`
	Territories       map[string]*Territory  `json:"territories"`
	WaterBodies       map[string]*WaterBody  `json:"waterBodies"`
	SkippedPhases     []PhaseSkipInfo        `json:"skippedPhases,omitempty"`     // Phases skipped in last transition
	ProductionPending bool                   `json:"productionPending,omitempty"` // True when production animation should play
}

// Settings contains the configurable game parameters.
type Settings struct {
	GameLevel      GameLevel   `json:"gameLevel"`
	ChanceLevel    ChanceLevel `json:"chanceLevel"`
	VictoryCities  int         `json:"victoryCities"`
	MapID          string      `json:"mapId"`
	MaxPlayers     int         `json:"maxPlayers"`
}

// GameLevel determines which features are available.
type GameLevel int

const (
	LevelBeginner GameLevel = iota
	LevelIntermediate
	LevelAdvanced
	LevelExpert
)

// ChanceLevel determines randomness in combat.
type ChanceLevel int

const (
	ChanceLow ChanceLevel = iota
	ChanceMedium
	ChanceHigh
)

// Phase represents the current phase of a game round.
type Phase int

const (
	PhaseTerritorySelection Phase = iota
	PhaseProduction
	PhaseTrade
	PhaseShipment
	PhaseConquest
	PhaseDevelopment
)

// String returns the phase name.
func (p Phase) String() string {
	switch p {
	case PhaseTerritorySelection:
		return "Territory Selection"
	case PhaseProduction:
		return "Production"
	case PhaseTrade:
		return "Trade"
	case PhaseShipment:
		return "Shipment"
	case PhaseConquest:
		return "Conquest"
	case PhaseDevelopment:
		return "Development"
	default:
		return "Unknown"
	}
}

// NewGame creates a new game with the given settings.
func NewGame(settings Settings, territories map[string]*Territory, waterBodies map[string]*WaterBody) *GameState {
	return &GameState{
		Settings:    settings,
		Round:       0,
		Phase:       PhaseTerritorySelection,
		Players:     make(map[string]*Player),
		Territories: territories,
		WaterBodies: waterBodies,
	}
}

// AddPlayer adds a player to the game.
func (g *GameState) AddPlayer(player *Player) {
	g.Players[player.ID] = player
}

// GetCurrentPlayer returns the player whose turn it is.
func (g *GameState) GetCurrentPlayer() *Player {
	return g.Players[g.CurrentPlayerID]
}

// IsGameOver checks if the game has ended.
func (g *GameState) IsGameOver() bool {
	// Check for single player remaining
	activePlayers := 0
	for _, p := range g.Players {
		if !p.Eliminated {
			activePlayers++
		}
	}
	if activePlayers <= 1 {
		return true
	}

	// Check for victory by cities
	for _, p := range g.Players {
		if g.CountCities(p.ID) >= g.Settings.VictoryCities {
			// Must be the only one at or above victory count
			othersAtVictory := false
			for _, other := range g.Players {
				if other.ID != p.ID && g.CountCities(other.ID) >= g.Settings.VictoryCities {
					othersAtVictory = true
					break
				}
			}
			if !othersAtVictory {
				return true
			}
		}
	}

	return false
}

// CountCities returns the number of cities owned by a player.
func (g *GameState) CountCities(playerID string) int {
	count := 0
	for _, t := range g.Territories {
		if t.Owner == playerID && t.HasCity {
			count++
		}
	}
	return count
}

// GetWinner returns the winning player, or nil if game is not over.
func (g *GameState) GetWinner() *Player {
	if !g.IsGameOver() {
		return nil
	}

	// Check for last player standing
	var lastPlayer *Player
	for _, p := range g.Players {
		if !p.Eliminated {
			if lastPlayer != nil {
				lastPlayer = nil // More than one player, not this victory type
				break
			}
			lastPlayer = p
		}
	}
	if lastPlayer != nil {
		return lastPlayer
	}

	// Check for city victory
	for _, p := range g.Players {
		if g.CountCities(p.ID) >= g.Settings.VictoryCities {
			return p
		}
	}

	return nil
}

