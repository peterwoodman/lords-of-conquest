package game

import (
	"fmt"

	"github.com/google/uuid"
)

// InitializeGame creates a new game state from a map and players.
func InitializeGame(mapData MapData, players []*Player, settings Settings) (*GameState, error) {
	if len(players) < 2 {
		return nil, fmt.Errorf("need at least 2 players")
	}
	if len(players) > 7 {
		return nil, fmt.Errorf("max 7 players")
	}

	state := &GameState{
		ID:          uuid.New().String(),
		Settings:    settings,
		Round:       0,
		Phase:       PhaseTerritorySelection,
		Players:     make(map[string]*Player),
		Territories: make(map[string]*Territory),
		WaterBodies: make(map[string]*WaterBody),
	}

	// Add players
	state.PlayerOrder = make([]string, len(players))
	for i, p := range players {
		state.Players[p.ID] = p
		state.PlayerOrder[i] = p.ID
		p.ResetTurn()
	}

	// Random player order for territory selection
	shufflePlayerOrder(state)
	state.CurrentPlayerID = state.PlayerOrder[0]

	// Create territories from map
	for id, t := range mapData.Territories {
		state.Territories[id] = &Territory{
			ID:           id,
			Name:         t.Name,
			Owner:        "", // Unclaimed
			Resource:     t.Resource,
			Adjacent:     t.Adjacent,
			CoastalTiles: t.CoastalTiles,
			WaterBodies:  t.WaterBodies,
		}
	}

	// Create water bodies
	for id, w := range mapData.WaterBodies {
		state.WaterBodies[id] = &WaterBody{
			ID:          id,
			Territories: w.Territories,
		}
	}

	return state, nil
}

// MapData is the data extracted from a map for game initialization.
type MapData struct {
	ID          string
	Name        string
	Territories map[string]TerritoryData
	WaterBodies map[string]WaterBodyData
}

// TerritoryData contains territory information from the map.
type TerritoryData struct {
	Name         string
	Resource     ResourceType
	Adjacent     []string
	CoastalTiles int
	WaterBodies  []string
}

// WaterBodyData contains water body information.
type WaterBodyData struct {
	Territories []string
}

// shufflePlayerOrder randomizes the player order.
func shufflePlayerOrder(state *GameState) {
	// Fisher-Yates shuffle
	for i := len(state.PlayerOrder) - 1; i > 0; i-- {
		j := randomInt(i + 1)
		state.PlayerOrder[i], state.PlayerOrder[j] = state.PlayerOrder[j], state.PlayerOrder[i]
	}
}

// randomInt returns a random integer from 0 to n-1.
func randomInt(n int) int {
	// Simple random for now
	return int(uint32(n) * uint32(uuid.New().ID()) % uint32(n))
}

