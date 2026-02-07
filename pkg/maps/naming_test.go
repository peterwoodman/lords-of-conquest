package maps

import (
	"fmt"
	"testing"
)

// TestNamingUniqueness generates several maps with different settings and
// verifies that every territory on each map has a unique name.
func TestNamingUniqueness(t *testing.T) {
	configs := []GeneratorOptions{
		{Width: 30, Territories: 40, WaterBorder: true, Islands: 1, Resources: 50},
		{Width: 40, Territories: 80, WaterBorder: true, Islands: 3, Resources: 70},
		{Width: 60, Territories: 120, WaterBorder: true, Islands: 5, Resources: 100},
		{Width: 20, Territories: 24, WaterBorder: false, Islands: 1, Resources: 30},
	}

	for ci, cfg := range configs {
		t.Run(fmt.Sprintf("config_%d_terr%d", ci, cfg.Territories), func(t *testing.T) {
			gen := NewGenerator(cfg)
			m, _ := gen.Generate()

			seen := make(map[string]int) // name -> first territory ID
			for tid, terr := range m.Territories {
				if prev, dup := seen[terr.Name]; dup {
					t.Errorf("duplicate name %q: territories %d and %d", terr.Name, prev, tid)
				}
				seen[terr.Name] = tid
			}

			t.Logf("generated %d territories, all names unique", len(m.Territories))
		})
	}
}

// TestNamingSpatialPrefixes generates a map and checks that directional
// prefixes (North/South/East/West) are roughly consistent with territory
// position.  We allow generous tolerance since prefixes are probabilistic.
func TestNamingSpatialPrefixes(t *testing.T) {
	gen := NewGenerator(GeneratorOptions{
		Width: 40, Territories: 60, WaterBorder: true, Islands: 2, Resources: 60,
	})
	m, _ := gen.Generate()

	// For every territory whose name starts with a cardinal direction,
	// check that it is at least vaguely in the right part of the map.
	for _, terr := range m.Territories {
		if len(terr.Cells) == 0 {
			continue
		}
		cx, cy := centroid(terr.Cells)
		nx := cx / float64(m.Width)
		ny := cy / float64(m.Height)

		name := terr.Name
		switch {
		case len(name) > 6 && name[:6] == "North ":
			if ny > 0.66 {
				t.Errorf("territory %d %q has North prefix but centroid y=%.2f (south half)", terr.ID, name, ny)
			}
		case len(name) > 6 && name[:6] == "South ":
			if ny < 0.33 {
				t.Errorf("territory %d %q has South prefix but centroid y=%.2f (north half)", terr.ID, name, ny)
			}
		case len(name) > 5 && name[:5] == "East ":
			if nx < 0.33 {
				t.Errorf("territory %d %q has East prefix but centroid x=%.2f (west third)", terr.ID, name, nx)
			}
		case len(name) > 5 && name[:5] == "West ":
			if nx > 0.66 {
				t.Errorf("territory %d %q has West prefix but centroid x=%.2f (east third)", terr.ID, name, nx)
			}
		}
	}
}

// TestNamingVariety generates a large map and checks that names come from
// multiple pools (resource-themed, coastal, generic).
func TestNamingVariety(t *testing.T) {
	gen := NewGenerator(GeneratorOptions{
		Width: 50, Territories: 100, WaterBorder: true, Islands: 3, Resources: 80,
	})
	m, _ := gen.Generate()

	// Collect all names and check we see a mix.
	hasCoastal := false
	hasGeneric := false
	hasResource := false

	coastalSet := toSet(namesCoastal)
	genericSet := toSet(namesGeneric)
	resourceSets := make(map[string]bool)
	for _, pool := range resourceNamePools {
		for _, n := range pool {
			resourceSets[n] = true
		}
	}

	for _, terr := range m.Territories {
		// Strip any prefix to get the base name.
		base := stripPrefix(terr.Name)
		if coastalSet[base] {
			hasCoastal = true
		}
		if genericSet[base] {
			hasGeneric = true
		}
		if resourceSets[base] {
			hasResource = true
		}
	}

	if !hasResource {
		t.Error("expected at least one resource-themed name")
	}
	if !hasGeneric {
		t.Error("expected at least one generic/historical name")
	}
	// Coastal is probabilistic, only warn.
	if !hasCoastal {
		t.Log("warning: no coastal-themed names appeared (probabilistic, may happen occasionally)")
	}

	t.Logf("variety check passed: resource=%v generic=%v coastal=%v", hasResource, hasGeneric, hasCoastal)
}

// --- helpers ---

func centroid(cells [][2]int) (float64, float64) {
	sx, sy := 0.0, 0.0
	for _, c := range cells {
		sx += float64(c[0])
		sy += float64(c[1])
	}
	n := float64(len(cells))
	return sx / n, sy / n
}

func toSet(names []string) map[string]bool {
	s := make(map[string]bool, len(names))
	for _, n := range names {
		s[n] = true
	}
	return s
}

func stripPrefix(name string) string {
	prefixes := []string{
		"Northwest ", "Northeast ", "Southwest ", "Southeast ",
		"North ", "South ", "East ", "West ",
		"Central ", "Inner ", "Old ", "Greater ",
		"Port ", "Cape ", "Bay ",
	}
	for _, p := range prefixes {
		if len(name) > len(p) && name[:len(p)] == p {
			return name[len(p):]
		}
	}
	return name
}
