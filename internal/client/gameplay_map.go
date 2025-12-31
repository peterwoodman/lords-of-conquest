package client

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawMap renders the game map with territories, colors, and boundaries
func (s *GameplayScene) drawMap(screen *ebiten.Image) {
	if s.mapData == nil {
		return
	}

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))
	grid := s.mapData["grid"].([]interface{})

	// Border inset - half the border width to keep color inside borders
	borderInset := float32(2)

	// Helper to get territory ID at position (returns -1 for out of bounds)
	getTerritoryAt := func(gx, gy int) int {
		if gx < 0 || gx >= width || gy < 0 || gy >= height {
			return -1
		}
		r := grid[gy].([]interface{})
		return int(r[gx].(float64))
	}

	// Draw territories
	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			territoryID := int(row[x].(float64))
			tid := fmt.Sprintf("t%d", territoryID) // Territory ID string

			sx, sy := s.gridToScreen(x, y)

			// Determine color
			var cellColor color.RGBA
			if territoryID == 0 {
				// Water
				cellColor = color.RGBA{20, 60, 120, 255}
			} else {
				// Land - get owner color
				if terr, ok := s.territories[tid].(map[string]interface{}); ok {
					owner := terr["owner"].(string)
					if owner != "" {
						if player, ok := s.players[owner].(map[string]interface{}); ok {
							playerColor := player["color"].(string)
							if pc, ok := PlayerColors[playerColor]; ok {
								cellColor = pc
							} else {
								cellColor = ColorPanelLight
							}
						} else {
							cellColor = ColorPanelLight
						}
					} else {
						// Unclaimed
						cellColor = color.RGBA{100, 100, 100, 255}
					}
				} else {
					cellColor = ColorPanelLight
				}
			}

			// Highlight selected territory (for shipment phase)
			if s.selectedTerritory != "" && tid == s.selectedTerritory {
				// Selection highlight
				cellColor.R = min(cellColor.R+60, 255)
				cellColor.G = min(cellColor.G+80, 255)
				cellColor.B = min(cellColor.B+60, 255)
			}

			// Highlight hovered cell
			if x == s.hoveredCell[0] && y == s.hoveredCell[1] {
				cellColor.R = min(cellColor.R+40, 255)
				cellColor.G = min(cellColor.G+40, 255)
				cellColor.B = min(cellColor.B+40, 255)
			}

			// Calculate insets based on borders with different territories
			leftInset := float32(0)
			topInset := float32(0)
			rightInset := float32(0)
			bottomInset := float32(0)

			// Check each neighbor - inset where there's a border
			if getTerritoryAt(x-1, y) != territoryID {
				leftInset = borderInset
			}
			if getTerritoryAt(x+1, y) != territoryID {
				rightInset = borderInset
			}
			if getTerritoryAt(x, y-1) != territoryID {
				topInset = borderInset
			}
			if getTerritoryAt(x, y+1) != territoryID {
				bottomInset = borderInset
			}

			// Draw the cell with insets
			cellX := float32(sx) + leftInset
			cellY := float32(sy) + topInset
			cellW := float32(s.cellSize) - leftInset - rightInset
			cellH := float32(s.cellSize) - topInset - bottomInset

			vector.DrawFilledRect(screen, cellX, cellY, cellW, cellH, cellColor, false)
		}
	}

	// Draw territory boundaries
	s.drawTerritoryBoundaries(screen, width, height, grid)

	// Draw territory icons (resources, buildings, units - but not boats)
	s.drawTerritoryIcons(screen)

	// Draw boats in water cells
	s.drawBoatsInWater(screen)
}

// drawTerritoryBoundaries draws lines between different territories with rounded corners
func (s *GameplayScene) drawTerritoryBoundaries(screen *ebiten.Image, width, height int, grid []interface{}) {
	borderColor := color.RGBA{0, 0, 0, 220}
	cornerRadius := float32(6) // Radius for rounded corners
	lineWidth := float32(4)    // Thicker borders like the original game

	// Helper to get territory ID at position (returns -1 for out of bounds)
	getTerritoryAt := func(x, y int) int {
		if x < 0 || x >= width || y < 0 || y >= height {
			return -1
		}
		row := grid[y].([]interface{})
		return int(row[x].(float64))
	}

	// First pass: draw the main border lines (shortened to leave room for corners)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			territoryID := getTerritoryAt(x, y)
			sx, sy := s.gridToScreen(x, y)

			// Check right neighbor - vertical line
			rightID := getTerritoryAt(x+1, y)
			if rightID != territoryID {
				lineX := float32(sx + s.cellSize)
				topY := float32(sy)
				bottomY := float32(sy + s.cellSize)

				// Check if we need to shorten for corners
				// Top corner: check if there's also a horizontal border above
				topID := getTerritoryAt(x, y-1)
				topRightID := getTerritoryAt(x+1, y-1)
				if (topID != territoryID || topRightID != rightID) && y > 0 {
					topY += cornerRadius
				}

				// Bottom corner: check if there's also a horizontal border below
				bottomID := getTerritoryAt(x, y+1)
				bottomRightID := getTerritoryAt(x+1, y+1)
				if (bottomID != territoryID || bottomRightID != rightID) && y < height-1 {
					bottomY -= cornerRadius
				}

				if topY < bottomY {
					vector.StrokeLine(screen, lineX, topY, lineX, bottomY, lineWidth, borderColor, false)
				}
			}

			// Check bottom neighbor - horizontal line
			bottomID := getTerritoryAt(x, y+1)
			if bottomID != territoryID {
				lineY := float32(sy + s.cellSize)
				leftX := float32(sx)
				rightX := float32(sx + s.cellSize)

				// Check if we need to shorten for corners
				// Left corner: check if there's also a vertical border to the left
				leftID := getTerritoryAt(x-1, y)
				bottomLeftID := getTerritoryAt(x-1, y+1)
				if (leftID != territoryID || bottomLeftID != bottomID) && x > 0 {
					leftX += cornerRadius
				}

				// Right corner: check if there's also a vertical border to the right
				rightTID := getTerritoryAt(x+1, y)
				bottomRightID := getTerritoryAt(x+1, y+1)
				if (rightTID != territoryID || bottomRightID != bottomID) && x < width-1 {
					rightX -= cornerRadius
				}

				if leftX < rightX {
					vector.StrokeLine(screen, leftX, lineY, rightX, lineY, lineWidth, borderColor, false)
				}
			}
		}
	}

	// Second pass: draw rounded corners
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Check each corner of this cell for rounded corners
			// We check the bottom-right corner of each cell
			sx, sy := s.gridToScreen(x, y)
			cornerX := float32(sx + s.cellSize)
			cornerY := float32(sy + s.cellSize)

			// Get the four territories meeting at this corner
			tl := getTerritoryAt(x, y)     // top-left
			tr := getTerritoryAt(x+1, y)   // top-right
			bl := getTerritoryAt(x, y+1)   // bottom-left
			br := getTerritoryAt(x+1, y+1) // bottom-right

			// Count unique territories at this corner
			hasVerticalBorder := tl != tr || bl != br
			hasHorizontalBorder := tl != bl || tr != br

			// Only draw corner if we have borders meeting
			if hasVerticalBorder && hasHorizontalBorder {
				// Determine which type of corner arc to draw based on territory configuration
				s.drawCornerArc(screen, cornerX, cornerY, cornerRadius, lineWidth, borderColor, tl, tr, bl, br)
			}
		}
	}
}

// drawCornerArc draws a rounded corner arc at the specified position
func (s *GameplayScene) drawCornerArc(screen *ebiten.Image, cx, cy, radius, lineWidth float32, col color.RGBA, tl, tr, bl, br int) {
	segments := 6 // Number of segments for the arc

	// Determine which quadrant(s) need arcs based on territory configuration
	// An arc is needed where two different territories meet at a corner

	// Check each of the four possible arc positions
	// Top-left arc (from top to left)
	if tl != tr && tl != bl {
		s.drawArcSegment(screen, cx, cy, radius, lineWidth, col, 180, 270, segments)
	}
	// Top-right arc (from right to top)
	if tr != tl && tr != br {
		s.drawArcSegment(screen, cx, cy, radius, lineWidth, col, 270, 360, segments)
	}
	// Bottom-right arc (from bottom to right)
	if br != bl && br != tr {
		s.drawArcSegment(screen, cx, cy, radius, lineWidth, col, 0, 90, segments)
	}
	// Bottom-left arc (from left to bottom)
	if bl != tl && bl != br {
		s.drawArcSegment(screen, cx, cy, radius, lineWidth, col, 90, 180, segments)
	}
}

// drawArcSegment draws a quarter arc
func (s *GameplayScene) drawArcSegment(screen *ebiten.Image, cx, cy, radius, lineWidth float32, col color.RGBA, startAngle, endAngle float64, segments int) {
	degToRad := 3.14159265358979323846 / 180.0 // degrees to radians

	for i := 0; i < segments; i++ {
		a1 := (startAngle + (endAngle-startAngle)*float64(i)/float64(segments)) * degToRad
		a2 := (startAngle + (endAngle-startAngle)*float64(i+1)/float64(segments)) * degToRad

		x1 := cx + radius*float32(cosApprox(a1))
		y1 := cy + radius*float32(sinApprox(a1))
		x2 := cx + radius*float32(cosApprox(a2))
		y2 := cy + radius*float32(sinApprox(a2))

		vector.StrokeLine(screen, x1, y1, x2, y2, lineWidth, col, false)
	}
}

// Simple sin/cos approximations to avoid importing math
func sinApprox(x float64) float64 {
	// Normalize to [-pi, pi]
	for x > 3.14159265358979323846 {
		x -= 2 * 3.14159265358979323846
	}
	for x < -3.14159265358979323846 {
		x += 2 * 3.14159265358979323846
	}
	// Taylor series approximation
	x2 := x * x
	return x * (1 - x2/6 + x2*x2/120 - x2*x2*x2/5040)
}

func cosApprox(x float64) float64 {
	return sinApprox(x + 3.14159265358979323846/2)
}

// drawTerritoryIcons draws icons for all territory contents (resources, buildings, units, stockpiles)
func (s *GameplayScene) drawTerritoryIcons(screen *ebiten.Image) {
	if s.mapData == nil || s.territories == nil {
		return
	}

	grid := s.mapData["grid"].([]interface{})

	// Build a map of stockpile territories for quick lookup
	stockpileTerritories := make(map[string]string) // territory ID -> player ID
	if s.players != nil {
		for playerID, playerData := range s.players {
			player := playerData.(map[string]interface{})
			if stockpileTerr, ok := player["stockpileTerritory"]; ok && stockpileTerr != nil && stockpileTerr != "" {
				stockpileTerritories[stockpileTerr.(string)] = playerID
			}
		}
	}

	// Draw icons for each territory
	for terrID, terrData := range s.territories {
		terr := terrData.(map[string]interface{})

		// Find all cells of this territory
		cells := s.findTerritoryCells(terrID, grid)
		if len(cells) == 0 {
			continue
		}

		// Collect what needs to be drawn with priority ordering
		// Priority: stockpile > city > weapon > horse > resource > boats
		type iconInfo struct {
			iconType string
			param    string
		}
		var icons []iconInfo

		// Stockpile (highest priority - most important to see)
		if playerID, hasStockpile := stockpileTerritories[terrID]; hasStockpile {
			icons = append(icons, iconInfo{"stockpile", playerID})
		}

		// City
		if hasCity, ok := terr["hasCity"].(bool); ok && hasCity {
			icons = append(icons, iconInfo{"city", ""})
		}

		// Weapon
		if hasWeapon, ok := terr["hasWeapon"].(bool); ok && hasWeapon {
			icons = append(icons, iconInfo{"weapon", ""})
		}

		// Horse
		if hasHorse, ok := terr["hasHorse"].(bool); ok && hasHorse {
			icons = append(icons, iconInfo{"horse", ""})
		}

		// Resource
		if resource, ok := terr["resource"].(string); ok && resource != "None" && resource != "" {
			icons = append(icons, iconInfo{"resource", resource})
		}

		// Boats are now drawn in water cells, not on territories

		// Draw each icon on a different cell
		for i, icon := range icons {
			if i >= len(cells) {
				break // More icons than cells, skip extras
			}
			cell := cells[i]
			sx, sy := s.gridToScreen(cell[0], cell[1])
			s.drawIconOnCell(screen, icon.iconType, icon.param, float32(sx), float32(sy))
		}
	}
}

// findTerritoryCells returns all cells belonging to a territory, sorted for consistent icon placement
func (s *GameplayScene) findTerritoryCells(territoryID string, grid []interface{}) [][2]int {
	// Extract numeric ID from "t1", "t2", etc.
	if len(territoryID) < 2 || territoryID[0] != 't' {
		return nil
	}

	var numID int
	fmt.Sscanf(territoryID[1:], "%d", &numID)

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	// Find all cells
	var cells [][2]int
	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			if int(row[x].(float64)) == numID {
				cells = append(cells, [2]int{x, y})
			}
		}
	}

	// Sort cells to prioritize interior cells (cells with more neighbors of same territory)
	// This helps place icons on cells that are clearly part of the territory
	if len(cells) > 1 {
		s.sortCellsByInteriorness(cells, numID, grid, width, height)
	}

	return cells
}

// sortCellsByInteriorness sorts cells so interior cells come first
func (s *GameplayScene) sortCellsByInteriorness(cells [][2]int, terrID int, grid []interface{}, width, height int) {
	// Calculate "interiorness" score for each cell (count of same-territory neighbors)
	scores := make([]int, len(cells))
	for i, cell := range cells {
		x, y := cell[0], cell[1]
		score := 0
		// Check 4 cardinal directions
		dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				row := grid[ny].([]interface{})
				if int(row[nx].(float64)) == terrID {
					score++
				}
			}
		}
		scores[i] = score
	}

	// Simple bubble sort (territories typically have few cells)
	for i := 0; i < len(cells)-1; i++ {
		for j := i + 1; j < len(cells); j++ {
			if scores[j] > scores[i] {
				cells[i], cells[j] = cells[j], cells[i]
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
}

// drawBoatsInWater draws boat icons in water cells adjacent to territories that own them
func (s *GameplayScene) drawBoatsInWater(screen *ebiten.Image) {
	if s.mapData == nil || s.territories == nil {
		return
	}

	// Get water grid and water bodies info
	_, hasWaterGrid := s.mapData["waterGrid"].([]interface{})
	waterBodies, hasWaterBodies := s.mapData["waterBodies"].(map[string]interface{})
	if !hasWaterGrid || !hasWaterBodies {
		return
	}

	grid := s.mapData["grid"].([]interface{})
	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	// For each territory, find its boats and draw them in adjacent water cells
	for terrID, terrData := range s.territories {
		terr, ok := terrData.(map[string]interface{})
		if !ok {
			continue
		}

		// Get boats map (water body ID -> count)
		boatsData, hasBoats := terr["boats"].(map[string]interface{})
		if !hasBoats || len(boatsData) == 0 {
			continue
		}

		// Get territory owner color
		var boatColor color.RGBA = color.RGBA{139, 90, 43, 255} // Brown default
		if owner, ok := terr["owner"].(string); ok && owner != "" {
			if player, ok := s.players[owner].(map[string]interface{}); ok {
				if playerColorName, ok := player["color"].(string); ok {
					if pc, ok := PlayerColors[playerColorName]; ok {
						boatColor = pc
					}
				}
			}
		}

		// Extract numeric territory ID
		var numTerritoryID int
		if len(terrID) > 1 && terrID[0] == 't' {
			fmt.Sscanf(terrID[1:], "%d", &numTerritoryID)
		}

		// For each water body with boats
		for waterBodyID, countVal := range boatsData {
			count := 0
			switch v := countVal.(type) {
			case float64:
				count = int(v)
			case int:
				count = v
			}
			if count <= 0 {
				continue
			}

			// Get water body info to find cells
			wbData, ok := waterBodies[waterBodyID].(map[string]interface{})
			if !ok {
				continue
			}

			wbCells, ok := wbData["cells"].([]interface{})
			if !ok {
				continue
			}

			// Find water cells in this water body that are adjacent to this territory
			adjacentWaterCells := make([][2]int, 0)
			for _, cellData := range wbCells {
				cell, ok := cellData.([]interface{})
				if !ok || len(cell) < 2 {
					continue
				}
				wx := int(cell[0].(float64))
				wy := int(cell[1].(float64))

				// Check if this water cell is adjacent to the territory
				dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
				for _, d := range dirs {
					nx, ny := wx+d[0], wy+d[1]
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						row := grid[ny].([]interface{})
						if int(row[nx].(float64)) == numTerritoryID {
							adjacentWaterCells = append(adjacentWaterCells, [2]int{wx, wy})
							break
						}
					}
				}
			}

			// Draw boats on adjacent water cells
			boatsDrawn := 0
			for _, waterCell := range adjacentWaterCells {
				if boatsDrawn >= count {
					break
				}
				sx, sy := s.gridToScreen(waterCell[0], waterCell[1])
				s.drawBoatInWaterCell(screen, float32(sx), float32(sy), boatColor)
				boatsDrawn++
			}

			// If not enough adjacent cells, show remaining count as number
			if boatsDrawn < count && len(adjacentWaterCells) > 0 {
				// Draw on the first cell with a count
				cell := adjacentWaterCells[0]
				sx, sy := s.gridToScreen(cell[0], cell[1])
				DrawText(screen, fmt.Sprintf("%d", count), sx+s.cellSize/2, sy+2, color.RGBA{255, 255, 255, 255})
			}
		}
	}
}

// drawBoatInWaterCell draws a single boat icon in a water cell with the owner's color
func (s *GameplayScene) drawBoatInWaterCell(screen *ebiten.Image, cellX, cellY float32, boatColor color.RGBA) {
	cellSize := float32(s.cellSize)
	iconSize := cellSize * 0.65

	// Center the icon in the cell
	offsetX := (cellSize - iconSize) / 2
	offsetY := (cellSize - iconSize) / 2
	x := cellX + offsetX
	y := cellY + offsetY

	// Try to use PNG icon first
	if iconImg := GetIcon("boat"); iconImg != nil {
		op := &ebiten.DrawImageOptions{}
		imgW := float32(iconImg.Bounds().Dx())
		imgH := float32(iconImg.Bounds().Dy())
		scaleX := iconSize / imgW
		scaleY := iconSize / imgH
		op.GeoM.Scale(float64(scaleX), float64(scaleY))
		op.GeoM.Translate(float64(x), float64(y))
		// Tint with owner color
		op.ColorScale.Scale(
			float32(boatColor.R)/255,
			float32(boatColor.G)/255,
			float32(boatColor.B)/255,
			1.0,
		)
		screen.DrawImage(iconImg, op)
	} else {
		// Fallback to drawing with color
		s.drawBoatIconFallbackColored(screen, x, y, iconSize, boatColor)
	}
}

// drawBoatIconFallbackColored draws a boat with specific color
func (s *GameplayScene) drawBoatIconFallbackColored(screen *ebiten.Image, x, y, size float32, boatColor color.RGBA) {
	// Simple boat shape
	cx := x + size/2
	cy := y + size/2

	// Hull
	hullW := size * 0.7
	hullH := size * 0.25
	hullY := cy + size*0.1

	vector.DrawFilledRect(screen, cx-hullW/2, hullY, hullW, hullH, boatColor, false)

	// Mast
	mastX := cx
	mastTop := cy - size*0.3

	vector.StrokeLine(screen, mastX, mastTop, mastX, hullY, 2, boatColor, false)

	// Sail triangle
	sailColor := color.RGBA{min(boatColor.R+50, 255), min(boatColor.G+50, 255), min(boatColor.B+50, 255), 255}
	vector.StrokeLine(screen, mastX, mastTop, mastX+size*0.25, cy, 1, sailColor, false)
	vector.StrokeLine(screen, mastX, mastTop, mastX, cy, 1, sailColor, false)
}
