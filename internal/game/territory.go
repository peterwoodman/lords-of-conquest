package game

// Territory represents a single territory on the map.
type Territory struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Owner        string            `json:"owner"` // Player ID, empty if unclaimed
	Resource     ResourceType      `json:"resource"`
	HasCity      bool              `json:"hasCity"`
	HasWeapon    bool              `json:"hasWeapon"`
	HasHorse     bool              `json:"hasHorse"`
	Boats        map[string]int    `json:"boats"`       // Water body ID -> boat count
	Adjacent     []string          `json:"adjacent"`    // IDs of adjacent territories
	CoastalTiles int               `json:"coastalTiles"` // Number of coastal tiles (limits boats)
	WaterBodies  []string          `json:"waterBodies"`  // IDs of connected water bodies
}

// WaterBody represents a connected body of water.
type WaterBody struct {
	ID          string   `json:"id"`
	Territories []string `json:"territories"` // IDs of territories bordering this water
}

// IsCoastal returns true if the territory has any coastal tiles.
func (t *Territory) IsCoastal() bool {
	return t.CoastalTiles > 0
}

// TotalBoats returns the total number of boats at this territory across all water bodies.
func (t *Territory) TotalBoats() int {
	total := 0
	for _, count := range t.Boats {
		total += count
	}
	return total
}

// BoatsInWater returns the number of boats in a specific water body.
func (t *Territory) BoatsInWater(waterBodyID string) int {
	if t.Boats == nil {
		return 0
	}
	return t.Boats[waterBodyID]
}

// CanAddBoat returns true if another boat can be placed at this territory.
func (t *Territory) CanAddBoat() bool {
	return t.TotalBoats() < t.CoastalTiles
}

// CanAddBoatToWater returns true if a boat can be placed in a specific water body.
func (t *Territory) CanAddBoatToWater(waterBodyID string) bool {
	// Check if territory borders this water body
	borders := false
	for _, wbID := range t.WaterBodies {
		if wbID == waterBodyID {
			borders = true
			break
		}
	}
	if !borders {
		return false
	}
	// Check if we have room for more boats
	return t.TotalBoats() < t.CoastalTiles
}

// AddBoat adds a boat to a specific water body.
func (t *Territory) AddBoat(waterBodyID string) {
	if t.Boats == nil {
		t.Boats = make(map[string]int)
	}
	t.Boats[waterBodyID]++
}

// RemoveBoat removes a boat from a specific water body.
func (t *Territory) RemoveBoat(waterBodyID string) bool {
	if t.Boats == nil || t.Boats[waterBodyID] <= 0 {
		return false
	}
	t.Boats[waterBodyID]--
	if t.Boats[waterBodyID] == 0 {
		delete(t.Boats, waterBodyID)
	}
	return true
}

// HasUnits returns true if any military units are present.
func (t *Territory) HasUnits() bool {
	return t.HasWeapon || t.HasHorse || t.TotalBoats() > 0
}

// BaseStrength returns the territory's contribution to combat.
func (t *Territory) BaseStrength() int {
	strength := 1 // Territory itself
	if t.HasCity {
		strength += 2
	}
	if t.HasWeapon {
		strength += 3
	}
	if t.HasHorse {
		strength += 1
	}
	strength += t.TotalBoats() * 2
	return strength
}

