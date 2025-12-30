// Package maps handles map loading, processing, and generation.
package maps

import "lords-of-conquest/internal/game"

// RawMap is the format stored in JSON files.
type RawMap struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Width       int                 `json:"width"`
	Height      int                 `json:"height"`
	Grid        [][]int             `json:"grid"` // Territory IDs, 0 = water
	Territories map[string]RawTerritory `json:"territories"`
}

// RawTerritory is territory data from the JSON file.
type RawTerritory struct {
	Name     string `json:"name"`
	Resource string `json:"resource,omitempty"` // coal, gold, iron, timber, grassland, or empty
}

// Map is the processed, runtime map data.
type Map struct {
	ID     string
	Name   string
	Width  int
	Height int

	// Grid of territory IDs (1+ for territories, 0 for water)
	Grid [][]int

	// Water body grid (0 = land, negative values = water body IDs)
	WaterGrid [][]int

	// Territory data indexed by ID
	Territories map[int]*Territory

	// Water bodies indexed by ID (negative numbers)
	WaterBodies map[int]*WaterBody
}

// Territory represents a territory on the map.
type Territory struct {
	ID                  int
	Name                string
	Resource            game.ResourceType
	Cells               [][2]int // List of [x,y] coordinates
	AdjacentTerritories []int    // Neighboring territory IDs
	AdjacentWaters      []int    // Water body IDs this touches (negative)
	CoastalCells        int      // Number of cells touching water (boat limit)
}

// WaterBody represents a connected body of water.
type WaterBody struct {
	ID                 int      // Negative: -1, -2, etc.
	Name               string   // Optional name like "Pacific"
	Cells              [][2]int // All water cells in this body
	CoastalTerritories []int    // Territory IDs that border this water
}

// GetTerritory returns a territory by ID.
func (m *Map) GetTerritory(id int) *Territory {
	return m.Territories[id]
}

// GetWaterBody returns a water body by ID.
func (m *Map) GetWaterBody(id int) *WaterBody {
	return m.WaterBodies[id]
}

// TerritoryAt returns the territory ID at the given coordinates.
// Returns 0 if water or out of bounds.
func (m *Map) TerritoryAt(x, y int) int {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return 0
	}
	return m.Grid[y][x]
}

// WaterBodyAt returns the water body ID at the given coordinates.
// Returns 0 if land or out of bounds.
func (m *Map) WaterBodyAt(x, y int) int {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return 0
	}
	return m.WaterGrid[y][x]
}

// TerritoryCount returns the number of territories.
func (m *Map) TerritoryCount() int {
	return len(m.Territories)
}

// CanBoatTravel checks if a boat can travel between two territories via water.
func (m *Map) CanBoatTravel(fromTerritoryID, toTerritoryID int) bool {
	from := m.Territories[fromTerritoryID]
	to := m.Territories[toTerritoryID]
	if from == nil || to == nil {
		return false
	}

	// Check if they share a water body
	for _, w1 := range from.AdjacentWaters {
		for _, w2 := range to.AdjacentWaters {
			if w1 == w2 {
				return true
			}
		}
	}
	return false
}

