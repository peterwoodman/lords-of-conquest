package game

// Territory represents a single territory on the map.
type Territory struct {
	ID           string
	Name         string
	Owner        string // Player ID, empty if unclaimed
	Resource     ResourceType
	HasCity      bool
	HasWeapon    bool
	HasHorse     bool
	Boats        int      // Number of boats (limited by coastal tiles)
	Adjacent     []string // IDs of adjacent territories
	CoastalTiles int      // Number of coastal tiles (limits boats)
	WaterBodies  []string // IDs of connected water bodies
}

// WaterBody represents a connected body of water.
type WaterBody struct {
	ID          string
	Territories []string // IDs of territories bordering this water
}

// IsCoastal returns true if the territory has any coastal tiles.
func (t *Territory) IsCoastal() bool {
	return t.CoastalTiles > 0
}

// CanAddBoat returns true if another boat can be placed here.
func (t *Territory) CanAddBoat() bool {
	return t.Boats < t.CoastalTiles
}

// HasUnits returns true if any military units are present.
func (t *Territory) HasUnits() bool {
	return t.HasWeapon || t.HasHorse || t.Boats > 0
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
	strength += t.Boats * 2
	return strength
}

