package maps

import (
	"fmt"
	"math/rand"
	"time"
)

// GeneratorOptions contains settings for map generation.
type GeneratorOptions struct {
	Size        MapSize        // S, M, L
	Territories TerritoryCount // Low, Medium, High
	WaterBorder bool           // Whether to surround map with water
	Islands     IslandAmount   // Low, Medium, High - controls land clustering
	Resources   ResourceAmount // Low, Medium, High
}

// MapSize represents map dimensions.
type MapSize int

const (
	MapSizeSmall MapSize = iota
	MapSizeMedium
	MapSizeLarge
)

// TerritoryCount represents number of territories.
type TerritoryCount int

const (
	TerritoryCountLow TerritoryCount = iota
	TerritoryCountMedium
	TerritoryCountHigh
)

// IslandAmount represents how spread out/clustered land is.
type IslandAmount int

const (
	IslandAmountLow IslandAmount = iota    // One big landmass
	IslandAmountMedium                      // A few landmasses
	IslandAmountHigh                        // Many small islands
)

// ResourceAmount represents resource density.
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

	switch opts.Size {
	case MapSizeSmall:
		g.width, g.height = 20, 15
	case MapSizeMedium:
		g.width, g.height = 28, 21
	case MapSizeLarge:
		g.width, g.height = 38, 28
	}

	return g
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

	g.steps = append(g.steps, GeneratorStep{IsComplete: true})
	return g.buildMap(), g.steps
}

func (g *Generator) calculateTerritoryCount() int {
	// Base on map size
	totalCells := g.width * g.height
	
	// How much of the map should be land vs water?
	var landRatio float64
	switch g.options.Islands {
	case IslandAmountLow:
		landRatio = 0.85 // Mostly land, little water
	case IslandAmountMedium:
		landRatio = 0.70 // Some water channels
	case IslandAmountHigh:
		landRatio = 0.50 // Lots of water, many islands
	}

	// Account for water border
	if g.options.WaterBorder {
		// Border takes up perimeter
		borderCells := 2*g.width + 2*(g.height-2)
		totalCells -= borderCells
	}

	landCells := int(float64(totalCells) * landRatio)

	// Average territory size based on territory count setting
	var avgSize int
	switch g.options.Territories {
	case TerritoryCountLow:
		avgSize = 12 // Fewer, larger
	case TerritoryCountMedium:
		avgSize = 9
	case TerritoryCountHigh:
		avgSize = 6 // More, smaller
	}

	count := landCells / avgSize
	if count < 6 {
		count = 6
	}
	if count > 60 {
		count = 60
	}
	return count
}

func (g *Generator) getTerritorySizeRange() (int, int) {
	switch g.options.Territories {
	case TerritoryCountLow:
		return 10, 15
	case TerritoryCountMedium:
		return 7, 12
	case TerritoryCountHigh:
		return 5, 9
	}
	return 5, 15
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

	// Minimum spacing between seeds - more spacing for more islands (more water between)
	var minSpacing int
	switch g.options.Islands {
	case IslandAmountLow:
		minSpacing = 2 // Seeds close together = connected land
	case IslandAmountMedium:
		minSpacing = 4 // Moderate spacing
	case IslandAmountHigh:
		minSpacing = 6 // Seeds far apart = separate islands with water between
	}

	// Try to place all seeds
	attempts := 0
	maxAttempts := count * 100

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
			if dist < minSpacing*minSpacing {
				tooClose = true
				break
			}
		}

		if !tooClose {
			seeds = append(seeds, [2]int{x, y})
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
		// Pick cell that creates compact shape (most neighbors of same territory)
		idx := g.pickCompactCell(frontier, terrID)
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

func (g *Generator) pickCompactCell(frontier [][2]int, terrID int) int {
	if len(frontier) <= 1 {
		return 0
	}

	// Score each cell by how many same-territory neighbors it has
	best := make([]int, 0)
	bestScore := -1
	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

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
		if score > bestScore {
			bestScore = score
			best = []int{i}
		} else if score == bestScore {
			best = append(best, i)
		}
	}

	return best[g.rng.Intn(len(best))]
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
	var ratio float64
	switch g.options.Resources {
	case ResourceAmountLow:
		ratio = 0.25
	case ResourceAmountMedium:
		ratio = 0.45
	case ResourceAmountHigh:
		ratio = 0.65
	}

	ids := make([]int, 0, len(g.territories))
	for tid := range g.territories {
		ids = append(ids, tid)
	}
	g.rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	numWithRes := int(float64(len(ids)) * ratio)
	resources := []string{"coal", "gold", "iron", "timber", "horses"}

	for i, tid := range ids {
		res := ""
		if i < numWithRes {
			res = resources[g.rng.Intn(len(resources))]
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
		Size:        MapSizeMedium,
		Territories: TerritoryCountMedium,
		WaterBorder: true,
		Islands:     IslandAmountMedium,
		Resources:   ResourceAmountMedium,
	}
}
