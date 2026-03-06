package maps

import (
	"encoding/json"
	"fmt"
)

// Registry holds all loaded maps.
var Registry = make(map[string]*Map)

// Get retrieves a map from the registry by ID.
func Get(id string) *Map {
	return Registry[id]
}

// validate checks a raw map for errors.
func validate(raw *RawMap) error {
	if raw.ID == "" {
		return fmt.Errorf("map ID is required")
	}
	if raw.Name == "" {
		return fmt.Errorf("map name is required")
	}
	if raw.Width <= 0 || raw.Height <= 0 {
		return fmt.Errorf("invalid dimensions: %dx%d", raw.Width, raw.Height)
	}
	if len(raw.Grid) != raw.Height {
		return fmt.Errorf("grid height mismatch: expected %d, got %d", raw.Height, len(raw.Grid))
	}
	for y, row := range raw.Grid {
		if len(row) != raw.Width {
			return fmt.Errorf("row %d width mismatch: expected %d, got %d", y, raw.Width, len(row))
		}
	}
	return nil
}

// LoadFromJSON loads a map from JSON bytes (for custom/uploaded maps).
func LoadFromJSON(data []byte) (*Map, error) {
	var raw RawMap
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse map JSON: %w", err)
	}

	if err := validate(&raw); err != nil {
		return nil, fmt.Errorf("invalid map: %w", err)
	}

	return Process(&raw), nil
}

// Register adds a map to the registry.
func Register(m *Map) {
	if m != nil && m.ID != "" {
		Registry[m.ID] = m
	}
}

