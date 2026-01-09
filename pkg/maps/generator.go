package maps

import (
	"fmt"
	"math/rand"
	"time"
)

// GeneratorOptions contains settings for map generation.
// All values are now numeric for fine-grained control via sliders.
type GeneratorOptions struct {
	Width       int  // Map width: 20-60
	Territories int  // Target territory count: 24-120
	WaterBorder bool // Whether to surround map with water
	Islands     int  // Island spread: 1-5 (1=one landmass, 5=many islands)
	Resources   int  // Resource coverage percentage: 10-80
}

// Legacy enum types kept for backwards compatibility during transition
// TODO: Remove these once all code is updated

// MapSize represents map dimensions (legacy).
type MapSize int

const (
	MapSizeSmall MapSize = iota
	MapSizeMedium
	MapSizeLarge
)

// TerritoryCount represents number of territories (legacy).
type TerritoryCount int

const (
	TerritoryCountLow TerritoryCount = iota
	TerritoryCountMedium
	TerritoryCountHigh
)

// IslandAmount represents how spread out/clustered land is (legacy).
type IslandAmount int

const (
	IslandAmountLow IslandAmount = iota    // One big landmass
	IslandAmountMedium                      // A few landmasses
	IslandAmountHigh                        // Many small islands
)

// ResourceAmount represents resource density (legacy).
type ResourceAmount int

const (
	ResourceAmountLow ResourceAmount = iota
	ResourceAmountMedium
	ResourceAmountHigh
)

// GeneratorStep represents one territory being placed.
type GeneratorStep struct {
	TerritoryID int
	Cells       [][2]int
	IsWater     bool
	IsComplete  bool
}

// Generator handles procedural map generation.
type Generator struct {
	options     GeneratorOptions
	rng         *rand.Rand
	width       int
	height      int
	grid        [][]int // 0 = water, 1+ = territory ID
	territories map[int]*terrData
	steps       []GeneratorStep
}

type terrData struct {
	id    int
	cells [][2]int
}

// NewGenerator creates a new map generator.
func NewGenerator(opts GeneratorOptions) *Generator {
	g := &Generator{
		options:     opts,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		territories: make(map[int]*terrData),
		steps:       make([]GeneratorStep, 0),
	}

	// Use numeric width directly, calculate height as 75% of width (aspect ratio)
	g.width = clamp(opts.Width, 20, 60)
	g.height = g.width * 3 / 4
	if g.height < 15 {
		g.height = 15
	}

	return g
}

// clamp restricts a value to a range
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// Generate creates the map. Water is whatever is left after territories are placed.
func (g *Generator) Generate() (*Map, []GeneratorStep) {
	// Initialize grid as all water
	g.grid = make([][]int, g.height)
	for y := range g.grid {
		g.grid[y] = make([]int, g.width)
		// 0 = water (default)
	}

	// Determine number of territories based on settings
	numTerritories := g.calculateTerritoryCount()

	// Place territory seeds based on islands setting
	// More islands = more spread out seeds = more water between them
	seeds := g.placeSeeds(numTerritories)

	// Grow each territory one at a time
	for i, seed := range seeds {
		terrID := i + 1
		
		// Target size: 5-15 cells
		minSize, maxSize := g.getTerritorySizeRange()
		targetSize := minSize + g.rng.Intn(maxSize-minSize+1)

		cells := g.growTerritory(terrID, seed[0], seed[1], targetSize)
		if len(cells) > 0 {
			g.territories[terrID] = &terrData{id: terrID, cells: cells}
			g.steps = append(g.steps, GeneratorStep{
				TerritoryID: terrID,
				Cells:       cells,
			})
		}
	}

	// Fix any diagonal-only connections (split disconnected parts)
	g.fixDiagonalConnections()
	
	// Merge tiny territories (< 5 cells) into neighbors
	g.mergeTinyTerritories(5)

	// Note: Single-pixel lakes are filled by fillLakes() in process.go
	// when Process(raw) is called in buildMap()

	g.steps = append(g.steps, GeneratorStep{IsComplete: true})
	return g.buildMap(), g.steps
}

// fixDiagonalConnections ensures all cells in a territory are orthogonally connected.
// Any disconnected parts are reassigned to neighbors or converted to water.
func (g *Generator) fixDiagonalConnections() {
	for terrID, terr := range g.territories {
		if len(terr.cells) == 0 {
			continue
		}
		
		// Find all orthogonally connected components
		components := g.findConnectedComponents(terr.cells, terrID)
		
		if len(components) <= 1 {
			continue // All cells are connected
		}
		
		// Keep the largest component, reassign others
		largestIdx := 0
		largestSize := 0
		for i, comp := range components {
			if len(comp) > largestSize {
				largestSize = len(comp)
				largestIdx = i
			}
		}
		
		// Update territory to only have the largest component
		g.territories[terrID].cells = components[largestIdx]
		
		// Reassign other components to neighbors or water
		for i, comp := range components {
			if i == largestIdx {
				continue
			}
			
			for _, cell := range comp {
				x, y := cell[0], cell[1]
				// Find best orthogonal neighbor
				neighbor := g.findOrthogonalNeighborTerritory(x, y, terrID)
				if neighbor > 0 {
					g.grid[y][x] = neighbor
					g.territories[neighbor].cells = append(g.territories[neighbor].cells, cell)
				} else {
					// No neighbor, convert to water
					g.grid[y][x] = 0
				}
			}
		}
	}
}

// findConnectedComponents finds all orthogonally connected groups of cells.
func (g *Generator) findConnectedComponents(cells [][2]int, terrID int) [][][2]int {
	// Create a set of cells for quick lookup
	cellSet := make(map[[2]int]bool)
	for _, c := range cells {
		cellSet[c] = true
	}
	
	visited := make(map[[2]int]bool)
	var components [][][2]int
	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	
	for _, startCell := range cells {
		if visited[startCell] {
			continue
		}
		
		// BFS to find all connected cells
		component := make([][2]int, 0)
		queue := [][2]int{startCell}
		visited[startCell] = true
		
		for len(queue) > 0 {
			cell := queue[0]
			queue = queue[1:]
			component = append(component, cell)
			
			x, y := cell[0], cell[1]
			for _, d := range dirs {
				neighbor := [2]int{x + d[0], y + d[1]}
				if cellSet[neighbor] && !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		
		components = append(components, component)
	}
	
	return components
}

// findOrthogonalNeighborTerritory finds a different territory orthogonally adjacent to this cell.
func (g *Generator) findOrthogonalNeighborTerritory(x, y, excludeID int) int {
	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	counts := make(map[int]int)
	
	for _, d := range dirs {
		nx, ny := x+d[0], y+d[1]
		if nx >= 0 && nx < g.width && ny >= 0 && ny < g.height {
			tid := g.grid[ny][nx]
			if tid > 0 && tid != excludeID {
				counts[tid]++
			}
		}
	}
	
	bestID := 0
	bestCount := 0
	for tid, count := range counts {
		if count > bestCount {
			bestCount = count
			bestID = tid
		}
	}
	return bestID
}

// mergeTinyTerritories merges territories smaller than minSize into adjacent territories.
func (g *Generator) mergeTinyTerritories(minSize int) {
	changed := true
	for changed {
		changed = false
		
		// Find territories that are too small
		for terrID, terr := range g.territories {
			if len(terr.cells) >= minSize {
				continue
			}
			
			// Find the best neighbor to merge into
			neighborCounts := make(map[int]int)
			dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
			
			for _, cell := range terr.cells {
				x, y := cell[0], cell[1]
				for _, d := range dirs {
					nx, ny := x+d[0], y+d[1]
					if nx >= 0 && nx < g.width && ny >= 0 && ny < g.height {
						neighborID := g.grid[ny][nx]
						if neighborID != 0 && neighborID != terrID {
							neighborCounts[neighborID]++
						}
					}
				}
			}
			
			// Find neighbor with most shared edges
			bestNeighbor := 0
			bestCount := 0
			for nid, count := range neighborCounts {
				if count > bestCount {
					bestCount = count
					bestNeighbor = nid
				}
			}
			
			if bestNeighbor == 0 {
				// No land neighbor found, convert to water
				for _, cell := range terr.cells {
					g.grid[cell[1]][cell[0]] = 0
				}
				delete(g.territories, terrID)
				changed = true
				break
			}
			
			// Merge into best neighbor
			for _, cell := range terr.cells {
				g.grid[cell[1]][cell[0]] = bestNeighbor
				g.territories[bestNeighbor].cells = append(g.territories[bestNeighbor].cells, cell)
			}
			delete(g.territories, terrID)
			changed = true
			break // Restart loop since we modified the map
		}
	}
}

func (g *Generator) calculateTerritoryCount() int {
	// Use the requested territory count directly
	requested := clamp(g.options.Territories, 24, 120)

	// Cap based on what can physically fit on the map
	totalCells := g.width * g.height

	// Account for water border
	if g.options.WaterBorder {
		borderCells := 2*g.width + 2*(g.height-2)
		totalCells -= borderCells
	}

	// Territory count is limited only by map size, not islands setting
	// Islands setting only affects how spread out the land is, not how much we try to place
	// Minimum ~5 cells per territory for it to be playable
	maxPossible := totalCells / 5
	if maxPossible < 6 {
		maxPossible = 6
	}

	// Return the smaller of requested vs what can fit
	if requested > maxPossible {
		return maxPossible
	}
	return requested
}

func (g *Generator) getTerritorySizeRange() (int, int) {
	// Fixed territory size range - consistent regardless of territory count
	// This creates natural variation in territory shapes while keeping them similar in size
	return 6, 12
}

func (g *Generator) placeSeeds(count int) [][2]int {
	seeds := make([][2]int, 0, count)

	// Determine valid area (inside border if water border enabled)
	minX, maxX := 0, g.width-1
	minY, maxY := 0, g.height-1
	if g.options.WaterBorder {
		minX, maxX = 1, g.width-2
		minY, maxY = 1, g.height-2
	}

	// Base spacing ensures territories have room to grow (minimum ~3 cells radius)
	// Islands setting adds extra spacing for water between landmasses
	// Islands 1 = connected landmass (base spacing only)
	// Islands 5 = many islands with water between (base + 4 extra spacing)
	islandLevel := clamp(g.options.Islands, 1, 5)
	baseSpacing := 3                        // Minimum spacing so territories can grow
	extraSpacing := islandLevel - 1         // 0-4 extra based on islands setting
	startSpacing := baseSpacing + extraSpacing

	// Try to place all seeds, reducing spacing if needed to hit territory count
	// Territory count takes priority over island spacing
	minAllowedSpacing := 2 // Never go below 2, or territories will be too cramped
	
	for spacing := startSpacing; spacing >= minAllowedSpacing; spacing-- {
		seeds = seeds[:0] // Reset seeds for each spacing attempt
		attempts := 0
		maxAttempts := count * 150

		for len(seeds) < count && attempts < maxAttempts {
			attempts++

			x := minX + g.rng.Intn(maxX-minX+1)
			y := minY + g.rng.Intn(maxY-minY+1)

			// Check spacing from existing seeds
			tooClose := false
			for _, s := range seeds {
				dx := x - s[0]
				dy := y - s[1]
				dist := dx*dx + dy*dy
				if dist < spacing*spacing {
					tooClose = true
					break
				}
			}

			if !tooClose {
				seeds = append(seeds, [2]int{x, y})
			}
		}

		// If we placed enough seeds, we're done
		if len(seeds) >= count {
			break
		}
	}

	return seeds
}

func (g *Generator) growTerritory(terrID, startX, startY, targetSize int) [][2]int {
	// Check if start is valid (not in border if water border)
	if g.options.WaterBorder {
		if startX == 0 || startX == g.width-1 || startY == 0 || startY == g.height-1 {
			return nil
		}
	}

	// If start cell already taken, find nearby empty cell
	if g.grid[startY][startX] != 0 {
		found := false
		for r := 1; r < 8 && !found; r++ {
			for dy := -r; dy <= r && !found; dy++ {
				for dx := -r; dx <= r && !found; dx++ {
					nx, ny := startX+dx, startY+dy
					if g.isValidLandCell(nx, ny) && g.grid[ny][nx] == 0 {
						startX, startY = nx, ny
						found = true
					}
				}
			}
		}
		if !found {
			return nil
		}
	}

	cells := make([][2]int, 0, targetSize)
	frontier := make([][2]int, 0)
	inFrontier := make(map[[2]int]bool)

	// Claim starting cell
	g.grid[startY][startX] = terrID
	cells = append(cells, [2]int{startX, startY})

	// Add valid neighbors to frontier
	g.addValidNeighbors(startX, startY, &frontier, inFrontier)

	// Grow until target size or no more frontier
	for len(cells) < targetSize && len(frontier) > 0 {
		// Pick cell with some randomness for organic shapes
		idx := g.pickGrowthCell(frontier, terrID)
		cell := frontier[idx]
		
		// Remove from frontier
		frontier[idx] = frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]
		delete(inFrontier, cell)

		x, y := cell[0], cell[1]

		// Skip if already claimed
		if g.grid[y][x] != 0 {
			continue
		}

		// Claim cell
		g.grid[y][x] = terrID
		cells = append(cells, cell)

		// Add new neighbors
		g.addValidNeighbors(x, y, &frontier, inFrontier)
	}

	return cells
}

func (g *Generator) isValidLandCell(x, y int) bool {
	if x < 0 || x >= g.width || y < 0 || y >= g.height {
		return false
	}
	// If water border, cells on edge are always water
	if g.options.WaterBorder {
		if x == 0 || x == g.width-1 || y == 0 || y == g.height-1 {
			return false
		}
	}
	return true
}

func (g *Generator) addValidNeighbors(x, y int, frontier *[][2]int, inFrontier map[[2]int]bool) {
	// Only cardinal directions - diagonals don't count as neighbors
	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	
	for _, d := range dirs {
		nx, ny := x+d[0], y+d[1]
		cell := [2]int{nx, ny}
		if g.isValidLandCell(nx, ny) && g.grid[ny][nx] == 0 && !inFrontier[cell] {
			*frontier = append(*frontier, cell)
			inFrontier[cell] = true
		}
	}
}

func (g *Generator) pickGrowthCell(frontier [][2]int, terrID int) int {
	if len(frontier) <= 1 {
		return 0
	}

	// 40% chance to pick completely random cell for irregular shapes
	if g.rng.Float32() < 0.40 {
		return g.rng.Intn(len(frontier))
	}

	// 30% chance to pick a cell with low connectivity (creates branches/tendrils)
	if g.rng.Float32() < 0.30 {
		return g.pickLowConnectivity(frontier, terrID)
	}

	// Otherwise pick a moderately compact cell (not always the most compact)
	return g.pickModerateCell(frontier, terrID)
}

func (g *Generator) pickLowConnectivity(frontier [][2]int, terrID int) int {
	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	
	// Find cells with exactly 1 neighbor (creates branches)
	lowCells := make([]int, 0)
	for i, cell := range frontier {
		count := 0
		x, y := cell[0], cell[1]
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && nx < g.width && ny >= 0 && ny < g.height {
				if g.grid[ny][nx] == terrID {
					count++
				}
			}
		}
		if count == 1 {
			lowCells = append(lowCells, i)
		}
	}
	
	if len(lowCells) > 0 {
		return lowCells[g.rng.Intn(len(lowCells))]
	}
	return g.rng.Intn(len(frontier))
}

func (g *Generator) pickModerateCell(frontier [][2]int, terrID int) int {
	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	
	// Group cells by score
	byScore := make(map[int][]int)
	for i, cell := range frontier {
		score := 0
		x, y := cell[0], cell[1]
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && nx < g.width && ny >= 0 && ny < g.height {
				if g.grid[ny][nx] == terrID {
					score++
				}
			}
		}
		byScore[score] = append(byScore[score], i)
	}
	
	// Weighted random: prefer score 1-2 over 3-4 for more organic shapes
	weights := map[int]int{1: 5, 2: 4, 3: 2, 4: 1}
	choices := make([]int, 0)
	for score, indices := range byScore {
		w := weights[score]
		if w == 0 {
			w = 1
		}
		for _, idx := range indices {
			for i := 0; i < w; i++ {
				choices = append(choices, idx)
			}
		}
	}
	
	if len(choices) > 0 {
		return choices[g.rng.Intn(len(choices))]
	}
	return g.rng.Intn(len(frontier))
}

func (g *Generator) buildMap() *Map {
	raw := &RawMap{
		ID:          fmt.Sprintf("gen_%d", time.Now().Unix()),
		Name:        "Generated Map",
		Width:       g.width,
		Height:      g.height,
		Grid:        g.grid,
		Territories: make(map[string]RawTerritory),
	}

	g.assignResources(raw)
	return Process(raw)
}

func (g *Generator) assignResources(raw *RawMap) {
	// Resources setting is a percentage (10-80)
	resourcePct := clamp(g.options.Resources, 10, 80)
	ratio := float64(resourcePct) / 100.0

	ids := make([]int, 0, len(g.territories))
	for tid := range g.territories {
		ids = append(ids, tid)
	}
	g.rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	numWithRes := int(float64(len(ids)) * ratio)
	if numWithRes < 5 {
		numWithRes = 5 // Minimum to guarantee one of each type
	}
	if numWithRes > len(ids) {
		numWithRes = len(ids)
	}

	// Build guaranteed resources list
	// Always include at least one of each critical resource
	guaranteed := []string{"coal", "gold", "iron", "timber", "grassland"}
	
	// On island maps (level 4-5), add extra timber and gold for boat building
	islandLevel := clamp(g.options.Islands, 1, 5)
	if islandLevel >= 4 {
		guaranteed = append(guaranteed, "timber", "gold", "timber") // Extra boat resources
	} else if islandLevel >= 3 {
		guaranteed = append(guaranteed, "timber", "gold") // Some extra
	}

	// Assign resources
	resources := []string{"coal", "gold", "iron", "timber", "grassland"}
	
	for i, tid := range ids {
		res := ""
		if i < numWithRes {
			if i < len(guaranteed) {
				// First territories get guaranteed resources
				res = guaranteed[i]
			} else {
				// Rest are random
				res = resources[g.rng.Intn(len(resources))]
			}
		}
		raw.Territories[fmt.Sprintf("%d", tid)] = RawTerritory{
			Name:     g.genName(tid),
			Resource: res,
		}
	}
}

func (g *Generator) genName(id int) string {
	prefixes := []string{"North", "South", "East", "West", "New", "Old", "Upper", "Lower"}
	names := []string{"Plains", "Valley", "Hills", "Forest", "Woods", "Fields", "Meadows", "Ridge",
		"Haven", "Landing", "Point", "Glen", "Dale", "Hollow", "Brook", "Springs"}
	suffixes := []string{"", "land", "ton", "ville", "burg", "ford", "shire"}

	r := rand.New(rand.NewSource(int64(id * 7919)))
	switch r.Intn(3) {
	case 0:
		return prefixes[r.Intn(len(prefixes))] + " " + names[r.Intn(len(names))]
	case 1:
		return names[r.Intn(len(names))] + suffixes[r.Intn(len(suffixes))]
	default:
		return names[r.Intn(len(names))]
	}
}

// DefaultOptions returns default generator options.
func DefaultOptions() GeneratorOptions {
	return GeneratorOptions{
		Width:       30,  // Medium width
		Territories: 40,  // Moderate territory count
		WaterBorder: true,
		Islands:     3,   // Medium islands (1-5 scale)
		Resources:   45,  // 45% resource coverage
	}
}
