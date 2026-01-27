package maps

import (
	"lords-of-conquest/internal/game"
	"strconv"
	"strings"
)

// Process takes a raw map and computes all derived data.
func Process(raw *RawMap) *Map {
	m := &Map{
		ID:          raw.ID,
		Name:        raw.Name,
		Width:       raw.Width,
		Height:      raw.Height,
		Territories: make(map[int]*Territory),
		WaterBodies: make(map[int]*WaterBody),
	}

	// Copy grid
	m.Grid = make([][]int, raw.Height)
	for y := range m.Grid {
		m.Grid[y] = make([]int, raw.Width)
		copy(m.Grid[y], raw.Grid[y])
	}

	// Step 1: Fill lakes (loose rule)
	fillLakes(m)

	// Step 2: Initialize water grid
	m.WaterGrid = make([][]int, m.Height)
	for y := range m.WaterGrid {
		m.WaterGrid[y] = make([]int, m.Width)
		// 0 = unprocessed water or land
	}

	// Step 3: Flood fill water bodies
	floodFillWaterBodies(m)

	// Step 4: Create territory objects and collect cells
	createTerritories(m, raw)

	// Step 5: Compute adjacencies
	computeAdjacencies(m)

	return m
}

// fillLakes converts isolated water cells (1-2 cell lakes) to land.
// Any small water body completely surrounded by land becomes the majority neighbor.
func fillLakes(m *Map) {
	// Find all water regions using flood fill
	visited := make([][]bool, m.Height)
	for y := range visited {
		visited[y] = make([]bool, m.Width)
	}

	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.Grid[y][x] != 0 || visited[y][x] {
				continue // Not water or already visited
			}

			// Flood fill to find connected water region
			cells := floodFillWater(m, x, y, visited)

			// Only fill small lakes (1-2 cells)
			if len(cells) > 2 {
				continue
			}

			// Check if this water region is completely surrounded by land
			// (no water neighbors outside the region, and all cells have 4 neighbors)
			allSurroundedByLand := true
			landCounts := make(map[int]int)

			for _, cell := range cells {
				cx, cy := cell[0], cell[1]
				dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

				for _, d := range dirs {
					nx, ny := cx+d[0], cy+d[1]

					// Check if out of bounds (edge of map = not surrounded)
					if nx < 0 || nx >= m.Width || ny < 0 || ny >= m.Height {
						allSurroundedByLand = false
						break
					}

					neighborVal := m.Grid[ny][nx]
					if neighborVal == 0 {
						// Water neighbor - check if it's part of our region
						isPartOfRegion := false
						for _, c := range cells {
							if c[0] == nx && c[1] == ny {
								isPartOfRegion = true
								break
							}
						}
						if !isPartOfRegion {
							allSurroundedByLand = false
							break
						}
					} else {
						landCounts[neighborVal]++
					}
				}

				if !allSurroundedByLand {
					break
				}
			}

			// If completely surrounded by land, fill with majority
			if allSurroundedByLand && len(landCounts) > 0 {
				majority := findMajority(landCounts)
				for _, cell := range cells {
					m.Grid[cell[1]][cell[0]] = majority
				}
			}
		}
	}
}

// floodFillWater finds all connected water cells from a starting point.
func floodFillWater(m *Map, startX, startY int, visited [][]bool) [][2]int {
	cells := make([][2]int, 0)
	queue := [][2]int{{startX, startY}}
	visited[startY][startX] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		x, y := current[0], current[1]

		cells = append(cells, [2]int{x, y})

		dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && nx < m.Width && ny >= 0 && ny < m.Height {
				if m.Grid[ny][nx] == 0 && !visited[ny][nx] {
					visited[ny][nx] = true
					queue = append(queue, [2]int{nx, ny})
				}
			}
		}
	}

	return cells
}

// getOrthogonalNeighbors returns the 4 orthogonal neighbors (up, down, left, right).
func getOrthogonalNeighbors(m *Map, x, y int) []int {
	neighbors := make([]int, 0, 4)
	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	for _, d := range dirs {
		nx, ny := x+d[0], y+d[1]
		if nx >= 0 && nx < m.Width && ny >= 0 && ny < m.Height {
			neighbors = append(neighbors, m.Grid[ny][nx])
		}
	}
	return neighbors
}

// findMajority returns the most common value in the counts map.
func findMajority(counts map[int]int) int {
	maxCount := 0
	majority := 0
	for val, count := range counts {
		if count > maxCount {
			maxCount = count
			majority = val
		}
	}
	return majority
}

// floodFillWaterBodies identifies and numbers connected water regions.
func floodFillWaterBodies(m *Map) {
	visited := make([][]bool, m.Height)
	for y := range visited {
		visited[y] = make([]bool, m.Width)
	}

	waterBodyID := -1 // Water bodies use negative IDs

	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.Grid[y][x] == 0 && !visited[y][x] {
				// Found unvisited water, flood fill it
				cells := floodFill(m, x, y, visited)

				wb := &WaterBody{
					ID:    waterBodyID,
					Name:  "", // Can be named later
					Cells: cells,
				}
				m.WaterBodies[waterBodyID] = wb

				// Mark water grid
				for _, cell := range cells {
					m.WaterGrid[cell[1]][cell[0]] = waterBodyID
				}

				waterBodyID--
			}
		}
	}
}

// floodFill does a flood fill from a starting point, returning all connected water cells.
func floodFill(m *Map, startX, startY int, visited [][]bool) [][2]int {
	cells := make([][2]int, 0)
	queue := [][2]int{{startX, startY}}
	visited[startY][startX] = true

	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		x, y := current[0], current[1]

		cells = append(cells, [2]int{x, y})

		// Check 4 orthogonal neighbors
		dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && nx < m.Width && ny >= 0 && ny < m.Height {
				if m.Grid[ny][nx] == 0 && !visited[ny][nx] {
					visited[ny][nx] = true
					queue = append(queue, [2]int{nx, ny})
				}
			}
		}
	}

	return cells
}

// createTerritories initializes territory objects from raw data.
func createTerritories(m *Map, raw *RawMap) {
	// First pass: find all territory IDs and their cells
	territoryCells := make(map[int][][2]int)

	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			tid := m.Grid[y][x]
			if tid > 0 {
				territoryCells[tid] = append(territoryCells[tid], [2]int{x, y})
			}
		}
	}

	// Create territory objects
	for tid, cells := range territoryCells {
		rawT, ok := raw.Territories[strconv.Itoa(tid)]
		name := ""
		resource := game.ResourceNone
		if ok {
			name = rawT.Name
			resource = parseResource(rawT.Resource)
		}
		if name == "" {
			name = "Territory " + strconv.Itoa(tid)
		}

		m.Territories[tid] = &Territory{
			ID:       tid,
			Name:     name,
			Resource: resource,
			Cells:    cells,
		}
	}

	// Renumber territories to ensure consecutive IDs (1, 2, 3, ...)
	// This fixes issues where merging leaves gaps in territory IDs
	renumberTerritories(m)
}

// renumberTerritories reassigns territory IDs to be consecutive starting from 1.
// This ensures no gaps in territory numbering after merging operations.
// It also handles orphaned grid cells that reference IDs not in the territories map
// by merging them into adjacent territories.
func renumberTerritories(m *Map) {
	if len(m.Territories) == 0 {
		return
	}

	// First pass: find ALL unique territory IDs in the grid (not just in m.Territories)
	// This catches orphaned cells that fillLakes might have assigned to deleted territory IDs
	gridIDs := make(map[int]bool)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			tid := m.Grid[y][x]
			if tid > 0 {
				gridIDs[tid] = true
			}
		}
	}

	// Find orphaned IDs (in grid but not in territories)
	orphanedIDs := make(map[int]bool)
	for id := range gridIDs {
		if _, exists := m.Territories[id]; !exists {
			orphanedIDs[id] = true
		}
	}

	// Fix orphaned cells by merging them into adjacent territories
	if len(orphanedIDs) > 0 {
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				tid := m.Grid[y][x]
				if !orphanedIDs[tid] {
					continue
				}

				// Find an adjacent valid territory to merge into
				dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
				newID := 0
				for _, d := range dirs {
					nx, ny := x+d[0], y+d[1]
					if nx >= 0 && nx < m.Width && ny >= 0 && ny < m.Height {
						neighborID := m.Grid[ny][nx]
						if neighborID > 0 && !orphanedIDs[neighborID] {
							newID = neighborID
							break
						}
					}
				}

				if newID > 0 {
					m.Grid[y][x] = newID
					// Add cell to the territory's cell list
					if terr, ok := m.Territories[newID]; ok {
						terr.Cells = append(terr.Cells, [2]int{x, y})
					}
				} else {
					// No valid neighbor found, convert to water
					m.Grid[y][x] = 0
				}
			}
		}
	}

	// Now collect existing IDs in sorted order (only from m.Territories)
	oldIDs := make([]int, 0, len(m.Territories))
	for id := range m.Territories {
		oldIDs = append(oldIDs, id)
	}
	// Sort IDs to ensure consistent renumbering
	for i := 0; i < len(oldIDs)-1; i++ {
		for j := i + 1; j < len(oldIDs); j++ {
			if oldIDs[j] < oldIDs[i] {
				oldIDs[i], oldIDs[j] = oldIDs[j], oldIDs[i]
			}
		}
	}

	// Check if renumbering is needed (are IDs already consecutive from 1?)
	needsRenumber := false
	for i, id := range oldIDs {
		if id != i+1 {
			needsRenumber = true
			break
		}
	}
	if !needsRenumber {
		return
	}

	// Create mapping from old ID to new ID
	oldToNew := make(map[int]int)
	for i, oldID := range oldIDs {
		oldToNew[oldID] = i + 1
	}

	// Update grid with new IDs
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			oldID := m.Grid[y][x]
			if newID, ok := oldToNew[oldID]; ok {
				m.Grid[y][x] = newID
			}
		}
	}

	// Rebuild territories map with new IDs
	newTerritories := make(map[int]*Territory)
	for oldID, terr := range m.Territories {
		newID := oldToNew[oldID]
		terr.ID = newID
		newTerritories[newID] = terr
	}
	m.Territories = newTerritories
}

// parseResource converts a resource string to ResourceType.
func parseResource(s string) game.ResourceType {
	switch strings.ToLower(s) {
	case "coal":
		return game.ResourceCoal
	case "gold":
		return game.ResourceGold
	case "iron":
		return game.ResourceIron
	case "timber", "wood":
		return game.ResourceTimber
	case "grassland":
		return game.ResourceGrassland
	default:
		return game.ResourceNone
	}
}

// computeAdjacencies calculates which territories border each other and water.
func computeAdjacencies(m *Map) {
	// For each territory, find adjacent territories and water bodies
	for _, t := range m.Territories {
		adjTerritories := make(map[int]bool)
		adjWaters := make(map[int]bool)
		coastalCount := 0

		for _, cell := range t.Cells {
			x, y := cell[0], cell[1]

			// Check 4 orthogonal neighbors
			dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
			for _, d := range dirs {
				nx, ny := x+d[0], y+d[1]

				// Check bounds
				if nx < 0 || nx >= m.Width || ny < 0 || ny >= m.Height {
					continue
				}

				neighborTerr := m.Grid[ny][nx]
				neighborWater := m.WaterGrid[ny][nx]

				if neighborTerr > 0 && neighborTerr != t.ID {
					adjTerritories[neighborTerr] = true
				}

				if neighborWater < 0 {
					adjWaters[neighborWater] = true
					coastalCount++
				}
			}
		}

		// Convert maps to slices
		t.AdjacentTerritories = make([]int, 0, len(adjTerritories))
		for id := range adjTerritories {
			t.AdjacentTerritories = append(t.AdjacentTerritories, id)
		}

		t.AdjacentWaters = make([]int, 0, len(adjWaters))
		for id := range adjWaters {
			t.AdjacentWaters = append(t.AdjacentWaters, id)
		}

		t.CoastalCells = coastalCount
	}

	// For each water body, find coastal territories
	for _, wb := range m.WaterBodies {
		coastalTerr := make(map[int]bool)

		for _, cell := range wb.Cells {
			x, y := cell[0], cell[1]

			// Check 4 orthogonal neighbors
			dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
			for _, d := range dirs {
				nx, ny := x+d[0], y+d[1]

				if nx < 0 || nx >= m.Width || ny < 0 || ny >= m.Height {
					continue
				}

				neighborTerr := m.Grid[ny][nx]
				if neighborTerr > 0 {
					coastalTerr[neighborTerr] = true
				}
			}
		}

		wb.CoastalTerritories = make([]int, 0, len(coastalTerr))
		for id := range coastalTerr {
			wb.CoastalTerritories = append(wb.CoastalTerritories, id)
		}
	}
}
