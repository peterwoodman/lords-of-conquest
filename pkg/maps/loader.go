package maps

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
)

//go:embed data/*.json
var mapFiles embed.FS

// Registry holds all loaded maps.
var Registry = make(map[string]*Map)

// LoadAll loads all embedded maps.
func LoadAll() error {
	entries, err := mapFiles.ReadDir("data")
	if err != nil {
		return fmt.Errorf("failed to read map directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		mapData, err := Load(entry.Name())
		if err != nil {
			return fmt.Errorf("failed to load map %s: %w", entry.Name(), err)
		}

		Registry[mapData.ID] = mapData
	}

	return nil
}

// Load loads a single map by filename.
func Load(filename string) (*Map, error) {
	data, err := mapFiles.ReadFile(path.Join("data", filename))
	if err != nil {
		return nil, fmt.Errorf("failed to read map file: %w", err)
	}

	var raw RawMap
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse map JSON: %w", err)
	}

	// Validate
	if err := validate(&raw); err != nil {
		return nil, fmt.Errorf("invalid map: %w", err)
	}

	// Process
	return Process(&raw), nil
}

// Get retrieves a map from the registry by ID.
func Get(id string) *Map {
	return Registry[id]
}

// List returns all map IDs and names.
func List() []MapInfo {
	infos := make([]MapInfo, 0, len(Registry))
	for _, m := range Registry {
		infos = append(infos, MapInfo{
			ID:             m.ID,
			Name:           m.Name,
			Width:          m.Width,
			Height:         m.Height,
			TerritoryCount: len(m.Territories),
		})
	}
	return infos
}

// MapInfo contains basic map information for listing.
type MapInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	TerritoryCount int    `json:"territory_count"`
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

