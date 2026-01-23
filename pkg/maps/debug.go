package maps

import (
	"fmt"
	"strings"
)

// Debug returns a string visualization of the map.
func (m *Map) Debug() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Map: %s (%s)\n", m.Name, m.ID))
	sb.WriteString(fmt.Sprintf("Size: %dx%d\n", m.Width, m.Height))
	sb.WriteString(fmt.Sprintf("Territories: %d\n", len(m.Territories)))
	sb.WriteString(fmt.Sprintf("Water Bodies: %d\n\n", len(m.WaterBodies)))

	// Print grid with territory IDs
	sb.WriteString("Territory Grid:\n")
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			t := m.Grid[y][x]
			if t == 0 {
				sb.WriteString(" .")
			} else {
				sb.WriteString(fmt.Sprintf("%2d", t))
			}
		}
		sb.WriteString("\n")
	}

	// Print territory details
	sb.WriteString("\nTerritories:\n")
	for id := 1; id <= len(m.Territories); id++ {
		t := m.Territories[id]
		if t == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("  %d. %s\n", t.ID, t.Name))
		sb.WriteString(fmt.Sprintf("     Resource: %s\n", t.Resource))
		sb.WriteString(fmt.Sprintf("     Cells: %d\n", len(t.Cells)))
		sb.WriteString(fmt.Sprintf("     Coastal: %d cells\n", t.CoastalCells))
		sb.WriteString(fmt.Sprintf("     Adjacent territories: %v\n", t.AdjacentTerritories))
		sb.WriteString(fmt.Sprintf("     Adjacent waters: %v\n", t.AdjacentWaters))
	}

	// Print water body details
	sb.WriteString("\nWater Bodies:\n")
	for id := -1; id >= -len(m.WaterBodies); id-- {
		wb := m.WaterBodies[id]
		if wb == nil {
			continue
		}
		name := wb.Name
		if name == "" {
			name = "(unnamed)"
		}
		sb.WriteString(fmt.Sprintf("  %d. %s\n", wb.ID, name))
		sb.WriteString(fmt.Sprintf("     Cells: %d\n", len(wb.Cells)))
		sb.WriteString(fmt.Sprintf("     Coastal territories: %v\n", wb.CoastalTerritories))
	}

	return sb.String()
}

// PrintAdjacencyMatrix prints which territories are adjacent.
func (m *Map) PrintAdjacencyMatrix() string {
	var sb strings.Builder

	n := len(m.Territories)
	sb.WriteString("Adjacency Matrix:\n   ")
	for i := 1; i <= n; i++ {
		sb.WriteString(fmt.Sprintf("%2d ", i))
	}
	sb.WriteString("\n")

	for i := 1; i <= n; i++ {
		sb.WriteString(fmt.Sprintf("%2d:", i))
		t := m.Territories[i]
		for j := 1; j <= n; j++ {
			if i == j {
				sb.WriteString(" - ")
			} else if contains(t.AdjacentTerritories, j) {
				sb.WriteString(" X ")
			} else {
				sb.WriteString(" . ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

