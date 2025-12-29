package maps

import (
	"lords-of-conquest/internal/game"
	"strconv"
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

// fillLakes converts isolated water cells to land.
// Loose rule: any water completely surrounded by land becomes the majority neighbor.
func fillLakes(m *Map) {
	changed := true
	for changed {
		changed = false
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				if m.Grid[y][x] != 0 {
					continue // Not water
				}

				// Check all 4 orthogonal neighbors
				neighbors := getOrthogonalNeighbors(m, x, y)
				allLand := true
				landCounts := make(map[int]int)

				for _, n := range neighbors {
					if n == 0 {
						allLand = false
						break
					}
					landCounts[n]++
				}

				// If completely surrounded by land, fill with majority
				if allLand && len(neighbors) == 4 {
					majority := findMajority(landCounts)
					m.Grid[y][x] = majority
					changed = true
				}
			}
		}
	}
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
}

// parseResource converts a resource string to ResourceType.
func parseResource(s string) game.ResourceType {
	switch s {
	case "coal":
		return game.ResourceCoal
	case "gold":
		return game.ResourceGold
	case "iron":
		return game.ResourceIron
	case "timber", "wood":
		return game.ResourceTimber
	case "horses":
		return game.ResourceHorses
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

