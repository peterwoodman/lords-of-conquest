package client

import (
	"fmt"
	"image/color"

	"lords-of-conquest/internal/game"

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

	// Border inset - matches arc inner edge (radius - lineWidth/2 = 6 - 2 = 4)
	// to ensure cell color doesn't show through rounded corners
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
				// Territory not found - this indicates a bug in map generation
				// Log once per missing territory
				if s.missingTerritories == nil {
					s.missingTerritories = make(map[string]bool)
				}
				if !s.missingTerritories[tid] {
					s.missingTerritories[tid] = true
					fmt.Printf("WARNING: Territory %s (grid ID %d) not found in territories map. Available: %v\n",
						tid, territoryID, getTerritoriesKeys(s.territories))
				}
				// Use a distinct error color (magenta) to make it obvious
				cellColor = color.RGBA{180, 50, 180, 255}
			}
		}

			// Highlight selected territory (for shipment phase)
			if s.selectedTerritory != "" && tid == s.selectedTerritory {
				// Selection highlight
				cellColor.R = min(cellColor.R+60, 255)
				cellColor.G = min(cellColor.G+80, 255)
				cellColor.B = min(cellColor.B+60, 255)
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

			// Draw player drawing pixels on this cell
			if territoryID != 0 && s.cellSize >= game.DrawingSubPixels {
				if terr, ok := s.territories[tid].(map[string]interface{}); ok {
					if drawing, ok := terr["drawing"].(map[string]interface{}); ok && len(drawing) > 0 {
						subSize := float32(s.cellSize) / float32(game.DrawingSubPixels)
						for subY := 0; subY < game.DrawingSubPixels; subY++ {
							for subX := 0; subX < game.DrawingSubPixels; subX++ {
								key := fmt.Sprintf("%d,%d", x*game.DrawingSubPixels+subX, y*game.DrawingSubPixels+subY)
								if colorVal, ok := drawing[key]; ok {
									colorIdx := 0
									switch v := colorVal.(type) {
									case float64:
										colorIdx = int(v)
									case int:
										colorIdx = v
									}
									if dc, ok := DrawingColors[colorIdx]; ok {
										px := float32(sx) + float32(subX)*subSize
										py := float32(sy) + float32(subY)*subSize
										vector.DrawFilledRect(screen, px, py, subSize, subSize, dc, false)
									}
								}
							}
						}
					}
				}
			}

			// Draw diagonal shading for cities
			if territoryID != 0 {
				if terr, ok := s.territories[tid].(map[string]interface{}); ok {
					if hasCity, ok := terr["hasCity"].(bool); ok && hasCity {
						// Draw diagonal lines across the cell
						shadeColor := color.RGBA{
							uint8(max(0, int(cellColor.R)-40)),
							uint8(max(0, int(cellColor.G)-40)),
							uint8(max(0, int(cellColor.B)-40)),
							255,
						}
						lineSpacing := float32(6)
						fullX := float32(sx)
						fullY := float32(sy)
						fullSize := float32(s.cellSize)

						// Calculate phase offset based on global position to align lines across cells
						// For 45-degree lines, lines align when (x - y) is constant
						// We need to find where lines cross the top edge of this cell
						globalOffset := fullX - fullY
						// Adjust to get the first line position within this cell
						phase := float32(int(globalOffset) % int(lineSpacing))
						if phase < 0 {
							phase += lineSpacing
						}

						// Draw lines from top-left to bottom-right direction
						for offset := phase - fullSize; offset < fullSize*2; offset += lineSpacing {
							x1 := fullX + offset
							y1 := fullY
							x2 := fullX + offset + fullSize
							y2 := fullY + fullSize

							// Clip to full cell bounds
							if x1 < fullX {
								y1 += fullX - x1
								x1 = fullX
							}
							if x2 > fullX+fullSize {
								y2 -= x2 - (fullX + fullSize)
								x2 = fullX + fullSize
							}
							if y1 < fullY {
								x1 += fullY - y1
								y1 = fullY
							}
							if y2 > fullY+fullSize {
								x2 -= y2 - (fullY + fullSize)
								y2 = fullY + fullSize
							}

							// Only draw if line is within bounds
							if x1 <= fullX+fullSize && x2 >= fullX && y1 <= fullY+fullSize && y2 >= fullY {
								vector.StrokeLine(screen, x1, y1, x2, y2, 1, shadeColor, false)
							}
						}
					}
				}
			}
		}
	}

	// Draw territory boundaries
	s.drawTerritoryBoundaries(screen, width, height, grid)

	// Draw territory icons (resources, buildings, units - but not boats)
	s.drawTerritoryIcons(screen)

	// Draw boats in water cells
	s.drawBoatsInWater(screen)

	// Draw pulsing highlights on selected territories
	s.drawTerritoryHighlights(screen, width, height, grid)
}

// drawTerritoryBoundaries draws lines between different territories with rounded corners
func (s *GameplayScene) drawTerritoryBoundaries(screen *ebiten.Image, width, height int, grid []interface{}) {
	borderColor := color.RGBA{0, 0, 0, 220}
	cornerRadius := float32(4) // Radius for rounded corners
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

		// City is shown via diagonal shading on the territory, not as an icon

		// Weapon
		if hasWeapon, ok := terr["hasWeapon"].(bool); ok && hasWeapon {
			icons = append(icons, iconInfo{"weapon", ""})
		}

		// Horse
		if hasHorse, ok := terr["hasHorse"].(bool); ok && hasHorse {
			icons = append(icons, iconInfo{"horse", ""})
		}

		// Resource - show two icons if there's city influence (doubles production)
		if resource, ok := terr["resource"].(string); ok && resource != "None" && resource != "" && resource != "Grassland" {
			icons = append(icons, iconInfo{"resource", resource})
			// Check for city influence (has city or adjacent to city)
			if s.hasCityInfluence(terrID, terr) {
				icons = append(icons, iconInfo{"resource", resource})
			}
		} else if resource == "Grassland" {
			// Grassland doesn't get doubled by cities
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

	// Try to use PNG icon (should be white for proper tinting)
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
	// Simple boat shape with prominent player color
	cx := x + size/2
	cy := y + size/2

	// Hull - main player color
	hullW := size * 0.8
	hullH := size * 0.3
	hullY := cy + size*0.15

	vector.DrawFilledRect(screen, cx-hullW/2, hullY, hullW, hullH, boatColor, false)
	// Hull outline for visibility
	vector.StrokeRect(screen, cx-hullW/2, hullY, hullW, hullH, 1, color.RGBA{0, 0, 0, 200}, false)

	// Mast
	mastX := cx
	mastTop := cy - size*0.25

	vector.StrokeLine(screen, mastX, mastTop, mastX, hullY, 2, color.RGBA{60, 40, 20, 255}, false)

	// Sail - filled triangle in player color (lighter)
	sailColor := color.RGBA{
		min(boatColor.R+40, 255),
		min(boatColor.G+40, 255),
		min(boatColor.B+40, 255),
		255,
	}
	// Draw sail as filled triangle using lines
	sailBottom := hullY - 2
	sailHeight := sailBottom - mastTop
	sailWidth := size * 0.3
	for i := float32(0); i < sailHeight; i += 1.5 {
		progress := i / sailHeight
		lineY := mastTop + i
		lineRight := mastX + sailWidth*progress
		vector.StrokeLine(screen, mastX, lineY, lineRight, lineY, 1.5, sailColor, false)
	}
}

// hasCityInfluence checks if a territory has a city or is adjacent to a city owned by the same player.
// This indicates the territory gets doubled production.
func (s *GameplayScene) hasCityInfluence(terrID string, terr map[string]interface{}) bool {
	// Check if territory itself has a city
	if hasCity, ok := terr["hasCity"].(bool); ok && hasCity {
		return true
	}

	// Check adjacent territories
	owner, _ := terr["owner"].(string)
	if owner == "" {
		return false
	}

	if adjacent, ok := terr["adjacent"].([]interface{}); ok {
		for _, adjID := range adjacent {
			adjIDStr := adjID.(string)
			if adjTerr, ok := s.territories[adjIDStr].(map[string]interface{}); ok {
				adjOwner, _ := adjTerr["owner"].(string)
				adjHasCity, _ := adjTerr["hasCity"].(bool)
				// Must be owned by same player and have a city
				if adjOwner == owner && adjHasCity {
					return true
				}
			}
		}
	}

	return false
}

// drawTerritoryHighlights draws pulsing borders around highlighted territories.
func (s *GameplayScene) drawTerritoryHighlights(screen *ebiten.Image, width, height int, grid []interface{}) {
	if len(s.highlightedTerritories) == 0 {
		return
	}

	// Calculate pulse alpha using a sine wave (0.3 to 1.0 range)
	// highlightPulseTimer increments each frame at 60fps
	pulsePhase := float64(s.highlightPulseTimer) * 0.08 // ~0.08 radians per frame = ~0.75Hz
	pulseAlpha := 0.3 + 0.7*(0.5+0.5*sinApprox(pulsePhase))

	getTerritoryAt := func(x, y int) int {
		if x < 0 || x >= width || y < 0 || y >= height {
			return -1
		}
		row := grid[y].([]interface{})
		return int(row[x].(float64))
	}

	for _, highlight := range s.highlightedTerritories {
		// Extract numeric ID from "t1", "t2", etc.
		var numID int
		if len(highlight.TerritoryID) < 2 || highlight.TerritoryID[0] != 't' {
			continue
		}
		fmt.Sscanf(highlight.TerritoryID[1:], "%d", &numID)

		// Apply pulse to highlight color
		hlColor := color.RGBA{
			highlight.Color.R,
			highlight.Color.G,
			highlight.Color.B,
			uint8(float64(highlight.Color.A) * pulseAlpha),
		}

		lineWidth := float32(3)

		// Draw border segments on the outer edges of this territory
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if getTerritoryAt(x, y) != numID {
					continue
				}

				sx, sy := s.gridToScreen(x, y)
				cs := float32(s.cellSize)
				fx := float32(sx)
				fy := float32(sy)

				// Draw edge on each side that borders a different territory
				if getTerritoryAt(x-1, y) != numID {
					vector.StrokeLine(screen, fx, fy, fx, fy+cs, lineWidth, hlColor, false)
				}
				if getTerritoryAt(x+1, y) != numID {
					vector.StrokeLine(screen, fx+cs, fy, fx+cs, fy+cs, lineWidth, hlColor, false)
				}
				if getTerritoryAt(x, y-1) != numID {
					vector.StrokeLine(screen, fx, fy, fx+cs, fy, lineWidth, hlColor, false)
				}
				if getTerritoryAt(x, y+1) != numID {
					vector.StrokeLine(screen, fx, fy+cs, fx+cs, fy+cs, lineWidth, hlColor, false)
				}
			}
		}
	}
}

// SetHighlightedTerritories sets the list of territories to highlight on the map.
func (s *GameplayScene) SetHighlightedTerritories(highlights []TerritoryHighlight) {
	s.highlightedTerritories = highlights
}

// ClearHighlightedTerritories removes all territory highlights.
func (s *GameplayScene) ClearHighlightedTerritories() {
	s.highlightedTerritories = nil
}

// AutoPanToTerritory smoothly pans the map to center a territory in the visible area.
func (s *GameplayScene) AutoPanToTerritory(territoryID string) {
	if s.mapData == nil {
		return
	}

	grid := s.mapData["grid"].([]interface{})
	cells := s.findTerritoryCells(territoryID, grid)
	if len(cells) == 0 {
		return
	}

	// Find centroid of territory cells
	sumX, sumY := 0, 0
	for _, cell := range cells {
		sumX += cell[0]
		sumY += cell[1]
	}
	centerGridX := sumX / len(cells)
	centerGridY := sumY / len(cells)

	// Calculate where this grid position would be in screen space with no pan
	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	sidebarWidth := 300
	bottomBarHeight := 120
	availableWidth := ScreenWidth - sidebarWidth - 20
	availableHeight := ScreenHeight - bottomBarHeight - 20

	cellW := availableWidth / width
	cellH := availableHeight / height
	baseCellSize := cellW
	if cellH < cellW {
		baseCellSize = cellH
	}
	if baseCellSize < 8 {
		baseCellSize = 8
	}
	if baseCellSize > 40 {
		baseCellSize = 40
	}
	zoomedCellSize := int(float64(baseCellSize) * s.zoom)
	if zoomedCellSize < 4 {
		zoomedCellSize = 4
	}

	mapW := width * zoomedCellSize
	mapH := height * zoomedCellSize
	baseOffsetX := sidebarWidth + (availableWidth-mapW)/2
	baseOffsetY := 10 + (availableHeight-mapH)/2

	// Target screen position (center of visible map area)
	targetScreenX := sidebarWidth + availableWidth/2
	targetScreenY := 10 + availableHeight/2

	// Current screen position of territory center (with no pan)
	terrScreenX := baseOffsetX + centerGridX*zoomedCellSize + zoomedCellSize/2
	terrScreenY := baseOffsetY + centerGridY*zoomedCellSize + zoomedCellSize/2

	// Calculate needed pan to center the territory
	s.panX = targetScreenX - terrScreenX
	s.panY = targetScreenY - terrScreenY
}

// getTerritoriesKeys returns the keys of the territories map for debugging
func getTerritoriesKeys(territories map[string]interface{}) []string {
	if territories == nil {
		return nil
	}
	keys := make([]string, 0, len(territories))
	for k := range territories {
		keys = append(keys, k)
	}
	return keys
}
