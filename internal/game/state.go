// Package game contains the core game logic for Lords of Conquest.
// This package is shared between client and server.
package game

import (
	"errors"
	"strings"
)

// GameState represents the complete state of a game.
type GameState struct {
	ID                        string                `json:"id"`
	Settings                  Settings              `json:"settings"`
	Round                     int                   `json:"round"`
	Phase                     Phase                 `json:"phase"`
	CurrentPlayerID           string                `json:"currentPlayerId"`
	PlayerOrder               []string              `json:"playerOrder"`
	Players                   map[string]*Player    `json:"players"`
	Territories               map[string]*Territory `json:"territories"`
	WaterBodies               map[string]*WaterBody `json:"waterBodies"`
	SkippedPhases             []PhaseSkipInfo       `json:"skippedPhases,omitempty"`             // Phases skipped in last transition
	ProductionPending         bool                  `json:"productionPending,omitempty"`         // True when production animation should play
	StockpilePlacementPending bool                  `json:"stockpilePlacementPending,omitempty"` // True when players need to place stockpiles
}

// Settings contains the configurable game parameters.
type Settings struct {
	ChanceLevel   ChanceLevel `json:"chanceLevel"`
	VictoryCities int         `json:"victoryCities"`
	MapID         string      `json:"mapId"`
	MaxPlayers    int         `json:"maxPlayers"`
}

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
	PhaseDevelopment              // First in year (but skipped in Year 1)
	PhaseProduction
	PhaseTrade
	PhaseShipment
	PhaseConquest // Last in year - end-game checked after this
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
	return g.IsEliminationVictory() || g.IsCityVictory()
}

// IsEliminationVictory checks if only one player remains (elimination victory).
// This should be checked immediately after combat.
func (g *GameState) IsEliminationVictory() bool {
	activePlayers := 0
	for _, p := range g.Players {
		if !p.Eliminated {
			activePlayers++
		}
	}
	return activePlayers <= 1
}

// IsCityVictory checks if any player has reached the victory city count
// AND has strictly more cities than all other players.
// This should only be checked at end of round (all players get a chance).
func (g *GameState) IsCityVictory() bool {
	for _, p := range g.Players {
		if p.Eliminated {
			continue
		}
		cityCount := g.CountCities(p.ID)
		if cityCount >= g.Settings.VictoryCities {
			// Must have strictly more cities than ALL other players
			hasStrictlyMore := true
			for _, other := range g.Players {
				if other.ID != p.ID && !other.Eliminated {
					if g.CountCities(other.ID) >= cityCount {
						hasStrictlyMore = false
						break
					}
				}
			}
			if hasStrictlyMore {
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

	// Check for last player standing (elimination victory)
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

	// Check for city victory - return the player with the highest city count
	// who is also at or above VictoryCities threshold
	var winner *Player
	highestCities := 0
	for _, p := range g.Players {
		if p.Eliminated {
			continue
		}
		cityCount := g.CountCities(p.ID)
		if cityCount >= g.Settings.VictoryCities && cityCount > highestCities {
			winner = p
			highestCities = cityCount
		}
	}

	return winner
}

// Surrender transfers all of a player's territories and stockpile to another player.
// Returns the number of territories transferred.
func (g *GameState) Surrender(surrenderPlayerID, targetPlayerID string) int {
	surrenderPlayer := g.Players[surrenderPlayerID]
	targetPlayer := g.Players[targetPlayerID]

	if surrenderPlayer == nil || targetPlayer == nil {
		return 0
	}

	// Count and transfer territories
	territoriesTransferred := 0
	for _, t := range g.Territories {
		if t.Owner == surrenderPlayerID {
			t.Owner = targetPlayerID
			territoriesTransferred++
		}
	}

	// Transfer stockpile resources
	if surrenderPlayer.Stockpile != nil && targetPlayer.Stockpile != nil {
		targetPlayer.Stockpile.Coal += surrenderPlayer.Stockpile.Coal
		targetPlayer.Stockpile.Gold += surrenderPlayer.Stockpile.Gold
		targetPlayer.Stockpile.Iron += surrenderPlayer.Stockpile.Iron
		targetPlayer.Stockpile.Timber += surrenderPlayer.Stockpile.Timber
	}

	// Clear surrendered player's stockpile
	surrenderPlayer.Stockpile = NewStockpile()
	surrenderPlayer.StockpileTerritory = ""

	// Mark player as eliminated
	surrenderPlayer.Eliminated = true

	return territoriesTransferred
}

// MaxTerritoryNameLength is the maximum allowed length for a territory name.
const MaxTerritoryNameLength = 30

// RenameTerritory renames a territory owned by the given player.
func (g *GameState) RenameTerritory(playerID, territoryID, newName string) error {
	territory, ok := g.Territories[territoryID]
	if !ok {
		return errors.New("territory not found")
	}

	if territory.Owner != playerID {
		return errors.New("you do not own this territory")
	}

	trimmed := strings.TrimSpace(newName)
	if trimmed == "" {
		return errors.New("territory name cannot be empty")
	}
	if len(trimmed) > MaxTerritoryNameLength {
		return errors.New("territory name is too long")
	}

	territory.Name = trimmed
	return nil
}
