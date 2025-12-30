package client

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// emptyImage is used for drawing triangles with solid colors
var emptyImage = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

// GameplayScene handles the main game display and interaction.
type GameplayScene struct {
	game *Game

	// Game state data
	gameState    map[string]interface{}
	mapData      map[string]interface{}
	territories  map[string]interface{}
	players      map[string]interface{}
	playerOrder  []interface{}
	currentPhase string
	currentTurn  string
	round        int

	// Rendering
	cellSize    int
	offsetX     int
	offsetY     int
	hoveredCell [2]int

	// UI
	infoPanel         *Panel
	actionPanel       *Panel
	endPhaseBtn       *Button
	selectedTerritory string // For multi-step actions like moving stockpile

	// Build menu (Development phase)
	showBuildMenu      bool
	buildMenuTerritory string
	buildCityBtn       *Button
	buildWeaponBtn     *Button
	buildBoatBtn       *Button
	cancelBuildBtn     *Button

	// Combat result display
	showCombatResult bool
	combatResult     *CombatResultData
	dismissResultBtn *Button
}

// CombatResultData holds the result of a combat for display
type CombatResultData struct {
	AttackerWins    bool
	AttackStrength  int
	DefenseStrength int
	TargetTerritory string
	TargetName      string
}

// Panel is a UI panel.
type Panel struct {
	X, Y, W, H int
}

// NewGameplayScene creates a new gameplay scene.
func NewGameplayScene(game *Game) *GameplayScene {
	s := &GameplayScene{
		game:        game,
		cellSize:    28,  // Slightly smaller to fit better
		offsetX:     260, // Leave room for left sidebar
		offsetY:     30,  // Top margin
		hoveredCell: [2]int{-1, -1},
	}

	// End phase button (positioned in drawBottomBar)
	s.endPhaseBtn = &Button{
		X: 0, Y: 0, W: 150, H: 45,
		Text:    "End Turn",
		Primary: true,
		OnClick: func() { s.game.EndPhase() },
	}

	// Build menu buttons
	s.buildCityBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Build City",
		OnClick: func() { s.doBuild("city") },
	}
	s.buildWeaponBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Build Weapon",
		OnClick: func() { s.doBuild("weapon") },
	}
	s.buildBoatBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Build Boat",
		OnClick: func() { s.doBuild("boat") },
	}
	s.cancelBuildBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Cancel",
		OnClick: func() { s.showBuildMenu = false },
	}

	// Combat result dismiss button
	s.dismissResultBtn = &Button{
		X: 0, Y: 0, W: 120, H: 40,
		Text:    "OK",
		Primary: true,
		OnClick: func() { s.showCombatResult = false },
	}

	return s
}

func (s *GameplayScene) OnEnter() {
	s.gameState = nil
}

func (s *GameplayScene) OnExit() {}

func (s *GameplayScene) Update() error {
	// Only process input if we have map data
	if s.mapData == nil {
		return nil
	}

	// Handle combat result dialog
	if s.showCombatResult {
		s.dismissResultBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			s.showCombatResult = false
		}
		return nil // Block other input while showing result
	}

	// Handle build menu
	if s.showBuildMenu {
		s.buildCityBtn.Update()
		s.buildWeaponBtn.Update()
		s.buildBoatBtn.Update()
		s.cancelBuildBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showBuildMenu = false
		}
		return nil // Block other input while showing menu
	}

	// Update hovered cell
	mx, my := ebiten.CursorPosition()
	s.hoveredCell = s.screenToGrid(mx, my)

	// Update buttons
	isMyTurn := s.currentTurn == s.game.config.PlayerID
	showEndButton := isMyTurn && s.isActionPhase()
	s.endPhaseBtn.Disabled = !showEndButton
	if showEndButton {
		s.endPhaseBtn.Update()
	}

	// Handle click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if s.hoveredCell[0] >= 0 {
			s.handleCellClick(s.hoveredCell[0], s.hoveredCell[1])
		}
	}

	// ESC to cancel selection
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.selectedTerritory = ""
	}

	return nil
}

// isActionPhase returns true if we're in a phase where the player can take actions and end their turn.
func (s *GameplayScene) isActionPhase() bool {
	return s.currentPhase == "Trade" || s.currentPhase == "Shipment" || s.currentPhase == "Conquest" || s.currentPhase == "Development"
}

func (s *GameplayScene) Draw(screen *ebiten.Image) {
	if s.mapData == nil {
		DrawLargeTextCentered(screen, "Loading game...", ScreenWidth/2, ScreenHeight/2, ColorText)
		return
	}

	// Background stars
	for i := 0; i < 40; i++ {
		x := float32((i * 167) % ScreenWidth)
		y := float32((i * 113) % ScreenHeight)
		alpha := uint8(30 + (i % 80))
		starColor := color.RGBA{100, 150, 255, alpha}
		vector.DrawFilledCircle(screen, x, y, 1, starColor, false)
	}

	// Left sidebar (You, Players, Resources)
	s.drawLeftSidebar(screen)

	// Map area with frame (fills remaining space)
	s.drawMapArea(screen)

	// Bottom info bar
	s.drawBottomBar(screen)

	// Draw hover info (includes attack preview during conquest)
	if s.hoveredCell[0] >= 0 {
		s.drawHoverInfo(screen)
	}

	// Draw build menu overlay
	if s.showBuildMenu {
		s.drawBuildMenu(screen)
	}

	// Draw combat result overlay
	if s.showCombatResult {
		s.drawCombatResult(screen)
	}
}

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

	// Draw territory icons (resources, buildings, units)
	s.drawTerritoryIcons(screen)
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
	import_math_used := 3.14159265358979323846 / 180.0 // degrees to radians

	for i := 0; i < segments; i++ {
		a1 := (startAngle + (endAngle-startAngle)*float64(i)/float64(segments)) * import_math_used
		a2 := (startAngle + (endAngle-startAngle)*float64(i+1)/float64(segments)) * import_math_used

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

		// Find the center of this territory
		centerX, centerY := s.findTerritoryCenter(terrID, grid)
		if centerX < 0 {
			continue
		}

		sx, sy := s.gridToScreen(centerX, centerY)
		cellCenterX := float32(sx) + float32(s.cellSize)/2
		cellCenterY := float32(sy) + float32(s.cellSize)/2

		// Collect what needs to be drawn
		var icons []string

		// Resource (always show if territory has one)
		if resource, ok := terr["resource"].(string); ok && resource != "None" && resource != "" {
			icons = append(icons, "resource:"+resource)
		}

		// City
		if hasCity, ok := terr["hasCity"].(bool); ok && hasCity {
			icons = append(icons, "city")
		}

		// Weapon
		if hasWeapon, ok := terr["hasWeapon"].(bool); ok && hasWeapon {
			icons = append(icons, "weapon")
		}

		// Horse
		if hasHorse, ok := terr["hasHorse"].(bool); ok && hasHorse {
			icons = append(icons, "horse")
		}

		// Boats
		if boats, ok := terr["boats"].(float64); ok && int(boats) > 0 {
			icons = append(icons, fmt.Sprintf("boats:%d", int(boats)))
		}

		// Stockpile
		if playerID, hasStockpile := stockpileTerritories[terrID]; hasStockpile {
			icons = append(icons, "stockpile:"+playerID)
		}

		// Draw icons in a grid around the center
		s.drawIconsAtPosition(screen, icons, cellCenterX, cellCenterY)
	}
}

// drawIconsAtPosition draws a set of icons arranged around a center point
func (s *GameplayScene) drawIconsAtPosition(screen *ebiten.Image, icons []string, cx, cy float32) {
	if len(icons) == 0 {
		return
	}

	iconSize := float32(8) // Base icon size
	spacing := float32(10) // Space between icons

	// Calculate positions based on icon count
	// For 1-2 icons: horizontal layout
	// For 3-4 icons: 2x2 grid
	// For 5-6 icons: 2x3 grid
	positions := s.calculateIconPositions(len(icons), cx, cy, iconSize, spacing)

	for i, icon := range icons {
		if i >= len(positions) {
			break
		}
		px, py := positions[i][0], positions[i][1]
		s.drawIcon(screen, icon, px, py, iconSize)
	}
}

// calculateIconPositions returns screen positions for arranging icons
func (s *GameplayScene) calculateIconPositions(count int, cx, cy, iconSize, spacing float32) [][2]float32 {
	positions := make([][2]float32, count)
	halfIcon := iconSize / 2

	switch count {
	case 1:
		// Center
		positions[0] = [2]float32{cx - halfIcon, cy - halfIcon}
	case 2:
		// Side by side
		positions[0] = [2]float32{cx - spacing/2 - halfIcon, cy - halfIcon}
		positions[1] = [2]float32{cx + spacing/2 - halfIcon, cy - halfIcon}
	case 3:
		// Triangle: 2 on top, 1 below
		positions[0] = [2]float32{cx - spacing/2 - halfIcon, cy - spacing/2 - halfIcon}
		positions[1] = [2]float32{cx + spacing/2 - halfIcon, cy - spacing/2 - halfIcon}
		positions[2] = [2]float32{cx - halfIcon, cy + spacing/2 - halfIcon}
	case 4:
		// 2x2 grid
		positions[0] = [2]float32{cx - spacing/2 - halfIcon, cy - spacing/2 - halfIcon}
		positions[1] = [2]float32{cx + spacing/2 - halfIcon, cy - spacing/2 - halfIcon}
		positions[2] = [2]float32{cx - spacing/2 - halfIcon, cy + spacing/2 - halfIcon}
		positions[3] = [2]float32{cx + spacing/2 - halfIcon, cy + spacing/2 - halfIcon}
	case 5:
		// 2 on top, 3 on bottom
		positions[0] = [2]float32{cx - spacing/2 - halfIcon, cy - spacing/2 - halfIcon}
		positions[1] = [2]float32{cx + spacing/2 - halfIcon, cy - spacing/2 - halfIcon}
		positions[2] = [2]float32{cx - spacing - halfIcon, cy + spacing/2 - halfIcon}
		positions[3] = [2]float32{cx - halfIcon, cy + spacing/2 - halfIcon}
		positions[4] = [2]float32{cx + spacing - halfIcon, cy + spacing/2 - halfIcon}
	default:
		// 3x2 grid for 6+
		for i := 0; i < count && i < 6; i++ {
			row := i / 3
			col := i % 3
			offsetX := (float32(col) - 1) * spacing
			offsetY := (float32(row) - 0.5) * spacing
			positions[i] = [2]float32{cx + offsetX - halfIcon, cy + offsetY - halfIcon}
		}
	}

	return positions
}

// drawIcon draws a single icon at the specified position
func (s *GameplayScene) drawIcon(screen *ebiten.Image, iconType string, x, y, size float32) {
	// Parse icon type
	parts := splitIconType(iconType)
	baseType := parts[0]
	param := ""
	if len(parts) > 1 {
		param = parts[1]
	}

	switch baseType {
	case "resource":
		s.drawResourceIcon(screen, param, x, y, size)
	case "city":
		s.drawCityIcon(screen, x, y, size)
	case "weapon":
		s.drawWeaponIcon(screen, x, y, size)
	case "horse":
		s.drawHorseIcon(screen, x, y, size)
	case "boats":
		count := 1
		fmt.Sscanf(param, "%d", &count)
		s.drawBoatIcon(screen, x, y, size, count)
	case "stockpile":
		s.drawStockpileIcon(screen, param, x, y, size)
	}
}

// splitIconType splits "type:param" into ["type", "param"]
func splitIconType(s string) []string {
	for i, c := range s {
		if c == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

// drawResourceIcon draws a resource indicator
func (s *GameplayScene) drawResourceIcon(screen *ebiten.Image, resource string, x, y, size float32) {
	var iconColor color.RGBA
	var symbol string

	switch resource {
	case "Coal":
		iconColor = color.RGBA{40, 40, 40, 255}
		symbol = "C"
	case "Gold":
		iconColor = color.RGBA{255, 200, 50, 255}
		symbol = "G"
	case "Iron":
		iconColor = color.RGBA{140, 140, 160, 255}
		symbol = "I"
	case "Timber":
		iconColor = color.RGBA{100, 70, 40, 255}
		symbol = "W"
	case "Horses":
		iconColor = color.RGBA{160, 100, 60, 255}
		symbol = "H"
	default:
		return
	}

	// Draw diamond shape for resources
	cx, cy := x+size/2, y+size/2
	halfSize := size * 0.6

	// Diamond points
	path := vector.Path{}
	path.MoveTo(cx, cy-halfSize) // Top
	path.LineTo(cx+halfSize, cy) // Right
	path.LineTo(cx, cy+halfSize) // Bottom
	path.LineTo(cx-halfSize, cy) // Left
	path.Close()

	// Fill
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(iconColor.R) / 255
		vs[i].ColorG = float32(iconColor.G) / 255
		vs[i].ColorB = float32(iconColor.B) / 255
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)

	// Border
	vector.StrokeLine(screen, cx, cy-halfSize, cx+halfSize, cy, 1, color.RGBA{0, 0, 0, 200}, false)
	vector.StrokeLine(screen, cx+halfSize, cy, cx, cy+halfSize, 1, color.RGBA{0, 0, 0, 200}, false)
	vector.StrokeLine(screen, cx, cy+halfSize, cx-halfSize, cy, 1, color.RGBA{0, 0, 0, 200}, false)
	vector.StrokeLine(screen, cx-halfSize, cy, cx, cy-halfSize, 1, color.RGBA{0, 0, 0, 200}, false)

	_ = symbol // Could draw letter if needed
}

// drawCityIcon draws a city building icon
func (s *GameplayScene) drawCityIcon(screen *ebiten.Image, x, y, size float32) {
	// Draw a small house/castle shape
	buildingColor := color.RGBA{220, 200, 180, 255}
	roofColor := color.RGBA{150, 80, 60, 255}

	// Building body (rectangle)
	bodyH := size * 0.6
	bodyY := y + size - bodyH
	vector.DrawFilledRect(screen, x+1, bodyY, size-2, bodyH, buildingColor, false)
	vector.StrokeRect(screen, x+1, bodyY, size-2, bodyH, 1, color.RGBA{100, 80, 60, 255}, false)

	// Roof (triangle)
	cx := x + size/2
	path := vector.Path{}
	path.MoveTo(cx, y)         // Top point
	path.LineTo(x+size, bodyY) // Right
	path.LineTo(x, bodyY)      // Left
	path.Close()

	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(roofColor.R) / 255
		vs[i].ColorG = float32(roofColor.G) / 255
		vs[i].ColorB = float32(roofColor.B) / 255
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)
}

// drawWeaponIcon draws a sword/weapon icon
func (s *GameplayScene) drawWeaponIcon(screen *ebiten.Image, x, y, size float32) {
	bladeColor := color.RGBA{180, 180, 200, 255}
	hiltColor := color.RGBA{120, 80, 40, 255}

	cx := x + size/2
	cy := y + size/2

	// Blade (vertical line)
	bladeLen := size * 0.7
	vector.StrokeLine(screen, cx, cy-bladeLen/2, cx, cy+bladeLen/3, 2, bladeColor, false)

	// Crossguard (horizontal line)
	guardLen := size * 0.5
	guardY := cy + bladeLen/6
	vector.StrokeLine(screen, cx-guardLen/2, guardY, cx+guardLen/2, guardY, 2, hiltColor, false)

	// Handle
	vector.StrokeLine(screen, cx, guardY, cx, cy+bladeLen/2, 2, hiltColor, false)
}

// drawHorseIcon draws a horse icon
func (s *GameplayScene) drawHorseIcon(screen *ebiten.Image, x, y, size float32) {
	horseColor := color.RGBA{140, 100, 60, 255}

	// Simplified horse shape: body oval + head + legs
	cx := x + size/2
	cy := y + size/2

	// Body (oval)
	bodyW := size * 0.7
	bodyH := size * 0.4
	vector.DrawFilledCircle(screen, cx, cy, bodyW/2, horseColor, false)

	// Head (smaller circle offset to right)
	headX := cx + bodyW/3
	headY := cy - bodyH/2
	vector.DrawFilledCircle(screen, headX, headY, size*0.2, horseColor, false)

	// Legs (simple lines)
	legColor := color.RGBA{100, 70, 40, 255}
	legY := cy + bodyH/2
	vector.StrokeLine(screen, cx-bodyW/4, cy, cx-bodyW/4, legY+size*0.2, 1.5, legColor, false)
	vector.StrokeLine(screen, cx+bodyW/4, cy, cx+bodyW/4, legY+size*0.2, 1.5, legColor, false)
}

// drawBoatIcon draws a boat icon with count
func (s *GameplayScene) drawBoatIcon(screen *ebiten.Image, x, y, size float32, count int) {
	boatColor := color.RGBA{100, 80, 60, 255}
	sailColor := color.RGBA{240, 240, 230, 255}

	cx := x + size/2
	cy := y + size/2

	// Hull (curved bottom)
	hullW := size * 0.8
	hullH := size * 0.3
	hullY := cy + size*0.2

	// Simple hull rectangle
	vector.DrawFilledRect(screen, cx-hullW/2, hullY, hullW, hullH, boatColor, false)

	// Sail (triangle)
	mastX := cx
	mastTop := cy - size*0.3
	path := vector.Path{}
	path.MoveTo(mastX, mastTop)
	path.LineTo(mastX+size*0.3, hullY)
	path.LineTo(mastX, hullY)
	path.Close()

	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(sailColor.R) / 255
		vs[i].ColorG = float32(sailColor.G) / 255
		vs[i].ColorB = float32(sailColor.B) / 255
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)

	// Mast
	vector.StrokeLine(screen, mastX, mastTop, mastX, hullY, 1, boatColor, false)

	// Count indicator if more than 1
	if count > 1 {
		countText := fmt.Sprintf("%d", count)
		DrawTextCentered(screen, countText, int(x+size+2), int(cy-2), color.RGBA{255, 255, 255, 255})
	}
}

// drawStockpileIcon draws a stockpile crate icon
func (s *GameplayScene) drawStockpileIcon(screen *ebiten.Image, playerID string, x, y, size float32) {
	// Stockpile color (golden brown)
	stockpileColor := color.RGBA{200, 160, 80, 255}
	borderColor := color.RGBA{150, 100, 30, 255}

	// Draw crate
	vector.DrawFilledRect(screen, x, y, size, size, stockpileColor, false)
	vector.StrokeRect(screen, x, y, size, size, 1.5, borderColor, false)

	// Cross-hatching for crate appearance
	vector.StrokeLine(screen, x, y, x+size, y+size, 1, borderColor, false)
	vector.StrokeLine(screen, x+size, y, x, y+size, 1, borderColor, false)
}

// findTerritoryCenter finds the center cell of a territory
func (s *GameplayScene) findTerritoryCenter(territoryID string, grid []interface{}) (int, int) {
	// Extract numeric ID from "t1", "t2", etc.
	if len(territoryID) < 2 || territoryID[0] != 't' {
		return -1, -1
	}

	var numID int
	fmt.Sscanf(territoryID[1:], "%d", &numID)

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	// Find all cells of this territory and compute center
	sumX, sumY, count := 0, 0, 0
	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			if int(row[x].(float64)) == numID {
				sumX += x
				sumY += y
				count++
			}
		}
	}

	if count == 0 {
		return -1, -1
	}

	return sumX / count, sumY / count
}

func (s *GameplayScene) drawPlayerIdentityPanel(screen *ebiten.Image) {
	// Deprecated - now part of drawLeftSidebar
}

func (s *GameplayScene) drawInfoPanel(screen *ebiten.Image) {
	// Deprecated - now part of drawBottomBar
}

// drawLeftSidebar draws player identity and players list.
func (s *GameplayScene) drawLeftSidebar(screen *ebiten.Image) {
	sidebarX := 10
	sidebarY := 10
	sidebarW := 200

	// Player identity panel
	myPlayer, ok := s.players[s.game.config.PlayerID]
	if ok {
		player := myPlayer.(map[string]interface{})
		playerName := player["name"].(string)
		playerColor := player["color"].(string)

		DrawFancyPanel(screen, sidebarX, sidebarY, sidebarW, 80, "You")

		DrawLargeText(screen, playerName, sidebarX+15, sidebarY+30, ColorText)

		// Color indicator
		if pc, ok := PlayerColors[playerColor]; ok {
			colorSize := float32(32)
			colorX := float32(sidebarX + sidebarW - 48)
			colorY := float32(sidebarY + 28)
			vector.DrawFilledRect(screen, colorX, colorY, colorSize, colorSize, pc, false)
			vector.StrokeRect(screen, colorX, colorY, colorSize, colorSize, 2, ColorBorder, false)
		}

		DrawText(screen, playerColor, sidebarX+15, sidebarY+58, ColorTextMuted)
	}

	// Players list - compact height based on player count
	playersY := sidebarY + 95
	playerCount := len(s.playerOrder)
	playersH := 40 + playerCount*26 // Header + per-player height
	if playersH > 200 {
		playersH = 200 // Cap max height
	}

	if playerCount > 0 {
		DrawFancyPanel(screen, sidebarX, playersY, sidebarW, playersH, "Players")

		y := playersY + 38
		for _, playerIDInterface := range s.playerOrder {
			playerID := playerIDInterface.(string)
			if playerData, ok := s.players[playerID]; ok {
				player := playerData.(map[string]interface{})
				playerName := player["name"].(string)
				playerColor := player["color"].(string)
				isAI := player["isAI"].(bool)

				// Color indicator
				if pc, ok := PlayerColors[playerColor]; ok {
					vector.DrawFilledRect(screen, float32(sidebarX+12), float32(y+2), 14, 14, pc, false)
					vector.StrokeRect(screen, float32(sidebarX+12), float32(y+2), 14, 14, 1, ColorBorder, false)
				}

				// Player name
				nameText := playerName
				if isAI {
					nameText += " (AI)"
				}
				if playerID == s.game.config.PlayerID {
					nameText += " *"
				}

				DrawText(screen, nameText, sidebarX+32, y, ColorText)
				y += 26

				if y > playersY+playersH-20 {
					break // Don't overflow
				}
			}
		}
	}

	// Resources panel - below Players
	resourcesY := playersY + playersH + 15
	s.drawResourcesPanel(screen, sidebarX, resourcesY, sidebarW)
}

// drawResourcesPanel draws the player's resources
func (s *GameplayScene) drawResourcesPanel(screen *ebiten.Image, x, y, w int) {
	myPlayer, ok := s.players[s.game.config.PlayerID]
	if !ok {
		return
	}

	player := myPlayer.(map[string]interface{})
	panelH := 170

	DrawFancyPanel(screen, x, y, w, panelH, "Resources")

	// Get stockpile data
	stockpile, hasStockpile := player["stockpile"]
	if hasStockpile {
		stockpileData := stockpile.(map[string]interface{})

		resY := y + 40
		resources := []struct {
			name  string
			key   string
			color color.RGBA
		}{
			{"Coal", "coal", color.RGBA{60, 60, 60, 255}},
			{"Gold", "gold", color.RGBA{255, 215, 0, 255}},
			{"Iron", "iron", color.RGBA{160, 160, 180, 255}},
			{"Wood", "timber", color.RGBA{139, 90, 43, 255}},
		}

		for _, res := range resources {
			count := 0
			if val, ok := stockpileData[res.key]; ok {
				count = int(val.(float64))
			}

			// Resource icon (colored square)
			vector.DrawFilledRect(screen, float32(x+12), float32(resY+2), 14, 14, res.color, false)
			vector.StrokeRect(screen, float32(x+12), float32(resY+2), 14, 14, 1, ColorBorder, false)

			text := fmt.Sprintf("%s: %d", res.name, count)
			DrawText(screen, text, x+32, resY, ColorText)
			resY += 24
		}

		// Stockpile location
		if stockpileTerr, ok := player["stockpileTerritory"]; ok && stockpileTerr != nil && stockpileTerr != "" {
			terrID := stockpileTerr.(string)
			if terr, ok := s.territories[terrID]; ok {
				terrData := terr.(map[string]interface{})
				terrName := terrData["name"].(string)
				DrawText(screen, "At: "+terrName, x+12, resY+5, ColorTextMuted)
			}
		}
	} else {
		DrawText(screen, "No stockpile yet", x+12, y+45, ColorTextMuted)
	}
}

// drawMapArea draws the map with a decorative frame.
func (s *GameplayScene) drawMapArea(screen *ebiten.Image) {
	if s.mapData == nil {
		return
	}

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	// Calculate available space for the map
	sidebarWidth := 220    // Left sidebar width + margin
	bottomBarHeight := 100 // Bottom bar height + margin
	availableWidth := ScreenWidth - sidebarWidth - 20
	availableHeight := ScreenHeight - bottomBarHeight - 20

	// Calculate cell size to fit the map in available space
	cellW := availableWidth / width
	cellH := availableHeight / height
	s.cellSize = cellW
	if cellH < cellW {
		s.cellSize = cellH
	}
	// Clamp cell size to reasonable bounds
	if s.cellSize < 16 {
		s.cellSize = 16
	}
	if s.cellSize > 40 {
		s.cellSize = 40
	}

	// Map dimensions with calculated cell size
	mapW := width * s.cellSize
	mapH := height * s.cellSize

	// Center the map in available space
	s.offsetX = sidebarWidth + (availableWidth-mapW)/2
	s.offsetY = 10 + (availableHeight-mapH)/2

	// Frame around map
	frameX := s.offsetX - 10
	frameY := s.offsetY - 10
	frameW := mapW + 20
	frameH := mapH + 20

	// Draw fancy frame
	DrawFancyPanel(screen, frameX, frameY, frameW, frameH, "")

	// Draw the map
	s.drawMap(screen)
}

// drawBottomBar draws phase/turn information.
func (s *GameplayScene) drawBottomBar(screen *ebiten.Image) {
	barX := 10
	barY := ScreenHeight - 90
	barW := ScreenWidth - 20
	barH := 80

	DrawFancyPanel(screen, barX, barY, barW, barH, "")

	// Phase and round - larger text
	phaseText := fmt.Sprintf("Round %d - %s", s.round, s.currentPhase)
	DrawLargeText(screen, phaseText, barX+15, barY+15, ColorText)

	// Phase-specific instructions
	instruction := ""
	instruction2 := ""
	showTurnIndicator := true
	isMyTurn := s.currentTurn == s.game.config.PlayerID

	switch s.currentPhase {
	case "Territory Selection":
		if isMyTurn {
			instruction = "Click an unclaimed territory to claim it"
		} else {
			instruction = "Waiting for other player to select..."
		}

	case "Production":
		if s.round == 1 {
			// Check if we've already placed our stockpile
			myStockpilePlaced := false
			if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
				player := myPlayer.(map[string]interface{})
				if stockpileTerr, ok := player["stockpileTerritory"]; ok && stockpileTerr != nil && stockpileTerr != "" {
					myStockpilePlaced = true
				}
			}

			if myStockpilePlaced {
				instruction = "Waiting for other players to place stockpiles..."
				instruction2 = ""
			} else {
				instruction = "Click one of YOUR territories to place your stockpile"
				instruction2 = "All players place stockpiles simultaneously"
			}
			showTurnIndicator = false
		} else {
			instruction = "Resources are being produced automatically"
		}

	case "Trade":
		instruction = "Trade phase - negotiate trades with other players"
		instruction2 = "Click 'End Turn' to skip trading"
		// Trade is available to all players simultaneously, but we show turn-based for simplicity

	case "Shipment":
		if isMyTurn {
			if s.selectedTerritory != "" {
				instruction = "Click a connected territory to move stockpile there"
				instruction2 = "Press ESC to cancel, or click 'End Turn' to skip"
			} else {
				instruction = "Click your stockpile territory, then click destination"
				instruction2 = "Or click 'End Turn' to skip this phase"
			}
		} else {
			instruction = "Waiting for other player to move units..."
		}

	case "Conquest":
		if isMyTurn {
			instruction = "Click an enemy territory adjacent to yours to attack"
			instruction2 = "Or click 'End Turn' when done attacking"
		} else {
			instruction = "Waiting for other player to attack..."
		}

	case "Development":
		if isMyTurn {
			instruction = "Build cities, weapons, or boats on your territories"
			instruction2 = "Click 'End Turn' when done building"
		} else {
			instruction = "Waiting for other player to build..."
		}
	}

	if instruction != "" {
		DrawText(screen, instruction, barX+15, barY+45, ColorTextMuted)
	}
	if instruction2 != "" {
		DrawText(screen, instruction2, barX+15, barY+60, ColorTextMuted)
	}

	// Current turn indicator (left side)
	if showTurnIndicator && s.currentTurn != "" {
		if player, ok := s.players[s.currentTurn].(map[string]interface{}); ok {
			playerName := player["name"].(string)
			playerColor := player["color"].(string)

			turnX := barX + 450

			turnText := fmt.Sprintf("Turn: %s", playerName)
			DrawText(screen, turnText, turnX, barY+15, ColorText)

			// Color indicator
			if pc, ok := PlayerColors[playerColor]; ok {
				textWidth := len(turnText) * 8 // Approximate
				vector.DrawFilledRect(screen, float32(turnX+textWidth+10), float32(barY+12),
					16, 16, pc, false)
				vector.StrokeRect(screen, float32(turnX+textWidth+10), float32(barY+12),
					16, 16, 2, ColorBorder, false)
			}

			// Indicate if it's your turn
			if isMyTurn {
				DrawLargeText(screen, "YOUR TURN!", turnX, barY+35, ColorSuccess)
			}
		}
	}

	// End Turn button (right side, only during action phases and your turn)
	if isMyTurn && s.isActionPhase() {
		s.endPhaseBtn.X = barX + barW - 170
		s.endPhaseBtn.Y = barY + 18
		s.endPhaseBtn.Draw(screen)
	}
}

func (s *GameplayScene) drawPlayersPanel(screen *ebiten.Image) {
	// Deprecated - now part of drawLeftSidebar
}

func (s *GameplayScene) drawResourcePanel(screen *ebiten.Image) {
	// Deprecated - now part of drawRightSidebar
}

func (s *GameplayScene) drawHoverInfo(screen *ebiten.Image) {
	if s.mapData == nil {
		return
	}

	x, y := s.hoveredCell[0], s.hoveredCell[1]
	grid := s.mapData["grid"].([]interface{})
	row := grid[y].([]interface{})
	territoryID := int(row[x].(float64))

	if territoryID == 0 {
		return // Water, no info
	}

	tid := fmt.Sprintf("t%d", territoryID)
	if terr, ok := s.territories[tid].(map[string]interface{}); ok {
		mx, my := ebiten.CursorPosition()

		owner := terr["owner"].(string)
		isMyTurn := s.currentTurn == s.game.config.PlayerID
		isEnemy := owner != "" && owner != s.game.config.PlayerID
		showAttackPreview := s.currentPhase == "Conquest" && isMyTurn && isEnemy

		// Collect territory contents for display
		var contents []string

		// Resource
		resource := terr["resource"].(string)
		if resource != "None" && resource != "" {
			contents = append(contents, "Resource: "+resource)
		}

		// City
		if hasCity, ok := terr["hasCity"].(bool); ok && hasCity {
			contents = append(contents, "🏠 City (+2 strength)")
		}

		// Weapon
		if hasWeapon, ok := terr["hasWeapon"].(bool); ok && hasWeapon {
			contents = append(contents, "⚔ Weapon (+3 strength)")
		}

		// Horse
		if hasHorse, ok := terr["hasHorse"].(bool); ok && hasHorse {
			contents = append(contents, "🐎 Horse (+1 strength)")
		}

		// Boats
		if boats, ok := terr["boats"].(float64); ok && int(boats) > 0 {
			boatCount := int(boats)
			contents = append(contents, fmt.Sprintf("⛵ Boats: %d (+%d strength)", boatCount, boatCount*2))
		}

		// Check for stockpile
		for _, playerData := range s.players {
			player := playerData.(map[string]interface{})
			if stockpileTerr, ok := player["stockpileTerritory"]; ok && stockpileTerr == tid {
				playerName := player["name"].(string)
				contents = append(contents, "📦 Stockpile ("+playerName+")")
				break
			}
		}

		// Coastal info
		if coastalTiles, ok := terr["coastalTiles"].(float64); ok && int(coastalTiles) > 0 {
			boats := 0
			if b, ok := terr["boats"].(float64); ok {
				boats = int(b)
			}
			contents = append(contents, fmt.Sprintf("🌊 Coastal (%d/%d boat slots)", boats, int(coastalTiles)))
		}

		// Determine box height based on content
		baseHeight := 48 // Name + Owner/Unclaimed
		contentHeight := len(contents) * 16
		attackPreviewHeight := 0
		if showAttackPreview {
			attackPreviewHeight = 65
		}
		boxH := baseHeight + contentHeight + attackPreviewHeight
		if boxH < 60 {
			boxH = 60
		}

		// Draw info box near cursor
		boxX := mx + 15
		boxY := my + 15
		boxW := 220

		// Keep on screen
		if boxX+boxW > ScreenWidth {
			boxX = mx - boxW - 15
		}
		if boxY+boxH > ScreenHeight {
			boxY = my - boxH - 15
		}

		DrawPanel(screen, boxX, boxY, boxW, boxH)

		name := terr["name"].(string)
		DrawText(screen, name, boxX+10, boxY+10, ColorText)

		if owner != "" {
			if player, ok := s.players[owner].(map[string]interface{}); ok {
				playerName := player["name"].(string)
				DrawText(screen, "Owner: "+playerName, boxX+10, boxY+28, ColorTextMuted)
			}
		} else {
			DrawText(screen, "Unclaimed", boxX+10, boxY+28, ColorTextMuted)
		}

		// Draw territory contents
		contentY := boxY + 44
		for _, content := range contents {
			DrawText(screen, content, boxX+10, contentY, ColorTextMuted)
			contentY += 16
		}

		// Attack preview during conquest phase
		if showAttackPreview {
			attackStr, defenseStr := s.calculateCombatStrength(tid)

			// Separator line
			vector.StrokeLine(screen, float32(boxX+10), float32(contentY+2), float32(boxX+boxW-10), float32(contentY+2), 1, ColorBorder, false)

			DrawText(screen, "⚔ ATTACK PREVIEW", boxX+10, contentY+10, ColorWarning)

			// Attack strength (green)
			attackText := fmt.Sprintf("Your Attack: %d", attackStr)
			DrawText(screen, attackText, boxX+10, contentY+27, ColorSuccess)

			// Defense strength (red)
			defenseText := fmt.Sprintf("Defense: %d", defenseStr)
			DrawText(screen, defenseText, boxX+10, contentY+41, ColorDanger)

			// Odds indicator
			var oddsText string
			var oddsColor color.RGBA
			if attackStr > defenseStr {
				oddsText = "Favorable odds!"
				oddsColor = ColorSuccess
			} else if attackStr == defenseStr {
				oddsText = "Even odds"
				oddsColor = ColorWarning
			} else {
				oddsText = "Risky attack!"
				oddsColor = ColorDanger
			}
			DrawText(screen, oddsText, boxX+120, contentY+27, oddsColor)
		}
	}
}

func (s *GameplayScene) handleCellClick(x, y int) {
	if s.mapData == nil {
		return
	}

	grid := s.mapData["grid"].([]interface{})
	row := grid[y].([]interface{})
	territoryID := int(row[x].(float64))

	if territoryID == 0 {
		return // Water
	}

	tid := fmt.Sprintf("t%d", territoryID)

	// Handle based on current phase
	switch s.currentPhase {
	case "Territory Selection":
		s.handleTerritorySelection(tid)
	case "Production":
		s.handleStockpilePlacement(tid)
	case "Trade":
		s.handleTrade(tid)
	case "Shipment":
		s.handleShipment(tid)
	case "Conquest":
		s.handleConquest(tid)
	case "Development":
		s.handleDevelopment(tid)
	default:
		log.Printf("No handler for phase: %s", s.currentPhase)
	}
}

func (s *GameplayScene) handleTerritorySelection(territoryID string) {
	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner == "" {
			// Unclaimed, we can select it
			s.game.SelectTerritory(territoryID)
		}
	}
}

func (s *GameplayScene) handleStockpilePlacement(territoryID string) {
	// Can only place on your own territories
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner == s.game.config.PlayerID {
			// This is our territory, place stockpile here
			s.game.PlaceStockpile(territoryID)
			log.Printf("Placing stockpile at %s", territoryID)
		}
	}
}

func (s *GameplayScene) handleTrade(territoryID string) {
	// Trade phase - clicking on territories doesn't do anything special
	// Trade would require a separate UI dialog to propose/accept trades
	// For now, players can just click "End Turn" to skip
	log.Printf("Trade phase - trade UI not yet implemented. Click 'End Turn' to skip.")
}

func (s *GameplayScene) handleShipment(territoryID string) {
	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	// Get our player data
	myPlayer, ok := s.players[s.game.config.PlayerID]
	if !ok {
		return
	}
	player := myPlayer.(map[string]interface{})

	// Get stockpile territory
	stockpileTerr, hasStockpile := player["stockpileTerritory"]
	if !hasStockpile || stockpileTerr == nil || stockpileTerr == "" {
		log.Printf("No stockpile to move")
		return
	}
	stockpileID := stockpileTerr.(string)

	// If no selection yet, check if clicking on stockpile
	if s.selectedTerritory == "" {
		if territoryID == stockpileID {
			s.selectedTerritory = territoryID
			log.Printf("Selected stockpile at %s - click destination to move", territoryID)
		} else {
			// Check if clicked on own territory - could be destination
			if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
				owner := terr["owner"].(string)
				if owner == s.game.config.PlayerID {
					// Move directly to this territory
					log.Printf("Moving stockpile to %s", territoryID)
					s.game.MoveStockpile(territoryID)
					s.selectedTerritory = ""
				}
			}
		}
	} else {
		// Already selected stockpile, this click is destination
		if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
			owner := terr["owner"].(string)
			if owner == s.game.config.PlayerID {
				log.Printf("Moving stockpile from %s to %s", s.selectedTerritory, territoryID)
				s.game.MoveStockpile(territoryID)
				s.selectedTerritory = ""
			} else {
				log.Printf("Cannot move stockpile to enemy territory")
			}
		}
	}
}

func (s *GameplayScene) handleConquest(territoryID string) {
	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	// Check if the territory belongs to an enemy
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner != "" && owner != s.game.config.PlayerID {
			// Enemy territory - try to attack
			log.Printf("Attacking territory %s", territoryID)
			s.game.ExecuteAttack(territoryID)
		} else if owner == s.game.config.PlayerID {
			log.Printf("Cannot attack your own territory")
		} else {
			log.Printf("Cannot attack unclaimed territory")
		}
	}
}

func (s *GameplayScene) handleDevelopment(territoryID string) {
	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	// Check if the territory belongs to us
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner == s.game.config.PlayerID {
			// Our territory - show build menu
			s.buildMenuTerritory = territoryID
			s.showBuildMenu = true
			log.Printf("Opened build menu for %s", territoryID)
		} else {
			log.Printf("Cannot build on enemy territory")
		}
	}
}

// calculateCombatStrength estimates attack and defense strength for a territory
func (s *GameplayScene) calculateCombatStrength(targetTID string) (attack, defense int) {
	target, ok := s.territories[targetTID].(map[string]interface{})
	if !ok {
		return 0, 0
	}

	targetOwner := target["owner"].(string)
	myID := s.game.config.PlayerID

	// Defense: 1 for the territory itself
	defense = 1

	// Add target's buildings and units
	defense += s.getTerritoryStrength(target)

	// Attack: count our adjacent territories
	for _, terrData := range s.territories {
		terr := terrData.(map[string]interface{})
		if terr["owner"].(string) != myID {
			continue
		}

		// Check if this territory is adjacent to target
		terrID := terr["id"].(string)
		if s.isAdjacent(terrID, targetTID) {
			attack++ // Territory contribution
			attack += s.getTerritoryStrength(terr)
		}
	}

	// Count defender's adjacent territories
	for _, terrData := range s.territories {
		terr := terrData.(map[string]interface{})
		if terr["owner"].(string) != targetOwner {
			continue
		}
		terrID := terr["id"].(string)
		if terrID != targetTID && s.isAdjacent(terrID, targetTID) {
			defense++ // Adjacent territory contribution
			defense += s.getTerritoryStrength(terr)
		}
	}

	return attack, defense
}

// getTerritoryStrength returns the combat strength bonus from a territory's buildings and units
func (s *GameplayScene) getTerritoryStrength(terr map[string]interface{}) int {
	strength := 0

	// City: +2
	if hasCity, ok := terr["hasCity"].(bool); ok && hasCity {
		strength += 2
	}

	// Weapon: +3
	if hasWeapon, ok := terr["hasWeapon"].(bool); ok && hasWeapon {
		strength += 3
	}

	// Horse: +1
	if hasHorse, ok := terr["hasHorse"].(bool); ok && hasHorse {
		strength += 1
	}

	// Boats: +2 each
	if boats, ok := terr["boats"].(float64); ok && int(boats) > 0 {
		strength += int(boats) * 2
	}

	return strength
}

// isAdjacent checks if two territories are adjacent (simplified check based on grid proximity)
func (s *GameplayScene) isAdjacent(tid1, tid2 string) bool {
	// Get centers of both territories to verify they exist
	grid := s.mapData["grid"].([]interface{})
	c1x, _ := s.findTerritoryCenter(tid1, grid)
	c2x, _ := s.findTerritoryCenter(tid2, grid)

	if c1x < 0 || c2x < 0 {
		return false
	}

	// Check if any cell of territory 1 is adjacent to any cell of territory 2
	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	// Extract numeric IDs
	var num1, num2 int
	fmt.Sscanf(tid1[1:], "%d", &num1)
	fmt.Sscanf(tid2[1:], "%d", &num2)

	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			if int(row[x].(float64)) != num1 {
				continue
			}
			// Check neighbors
			neighbors := [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}}
			for _, n := range neighbors {
				nx, ny := n[0], n[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				neighborRow := grid[ny].([]interface{})
				if int(neighborRow[nx].(float64)) == num2 {
					return true
				}
			}
		}
	}
	return false
}

// drawBuildMenu draws the build selection menu
func (s *GameplayScene) drawBuildMenu(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 180}, false)

	// Menu panel
	menuW := 280
	menuH := 280
	menuX := ScreenWidth/2 - menuW/2
	menuY := ScreenHeight/2 - menuH/2

	DrawFancyPanel(screen, menuX, menuY, menuW, menuH, "Build")

	// Get player resources to show affordability
	coal, gold, iron, timber := 0, 0, 0, 0
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if stockpile, ok := player["stockpile"].(map[string]interface{}); ok {
			if v, ok := stockpile["coal"].(float64); ok {
				coal = int(v)
			}
			if v, ok := stockpile["gold"].(float64); ok {
				gold = int(v)
			}
			if v, ok := stockpile["iron"].(float64); ok {
				iron = int(v)
			}
			if v, ok := stockpile["timber"].(float64); ok {
				timber = int(v)
			}
		}
	}

	// Get territory info
	terrName := "territory"
	hasCity, hasWeapon := false, false
	isCoastal := false
	if terr, ok := s.territories[s.buildMenuTerritory].(map[string]interface{}); ok {
		terrName = terr["name"].(string)
		if hc, ok := terr["hasCity"].(bool); ok {
			hasCity = hc
		}
		if hw, ok := terr["hasWeapon"].(bool); ok {
			hasWeapon = hw
		}
		// Check coastal (simplified - would need server data)
		isCoastal = true // Assume true for now, server will reject if not
	}

	DrawText(screen, "Building at: "+terrName, menuX+15, menuY+40, ColorTextMuted)

	btnX := menuX + 40
	btnY := menuY + 70

	// City button - costs 1 coal, 1 gold, 1 iron, 1 timber (or 4 gold)
	canAffordCity := (coal >= 1 && gold >= 1 && iron >= 1 && timber >= 1) || gold >= 4
	s.buildCityBtn.X = btnX
	s.buildCityBtn.Y = btnY
	s.buildCityBtn.Disabled = !canAffordCity || hasCity
	s.buildCityBtn.Text = "City (C+G+I+W or 4G)"
	if hasCity {
		s.buildCityBtn.Text = "City (already built)"
	}
	s.buildCityBtn.Draw(screen)

	// Weapon button - costs 1 coal, 1 iron (or 2 gold)
	canAffordWeapon := (coal >= 1 && iron >= 1) || gold >= 2
	s.buildWeaponBtn.X = btnX
	s.buildWeaponBtn.Y = btnY + 50
	s.buildWeaponBtn.Disabled = !canAffordWeapon || hasWeapon
	s.buildWeaponBtn.Text = "Weapon (C+I or 2G)"
	if hasWeapon {
		s.buildWeaponBtn.Text = "Weapon (already built)"
	}
	s.buildWeaponBtn.Draw(screen)

	// Boat button - costs 3 timber (or 3 gold), coastal only
	canAffordBoat := timber >= 3 || gold >= 3
	s.buildBoatBtn.X = btnX
	s.buildBoatBtn.Y = btnY + 100
	s.buildBoatBtn.Disabled = !canAffordBoat || !isCoastal
	s.buildBoatBtn.Text = "Boat (3W or 3G)"
	if !isCoastal {
		s.buildBoatBtn.Text = "Boat (coastal only)"
	}
	s.buildBoatBtn.Draw(screen)

	// Cancel button
	s.cancelBuildBtn.X = btnX
	s.cancelBuildBtn.Y = btnY + 160
	s.cancelBuildBtn.Draw(screen)

	// Resource reminder
	resText := fmt.Sprintf("Resources: C:%d G:%d I:%d W:%d", coal, gold, iron, timber)
	DrawText(screen, resText, menuX+15, menuY+menuH-25, ColorTextMuted)
}

// drawCombatResult draws the combat result popup
func (s *GameplayScene) drawCombatResult(screen *ebiten.Image) {
	if s.combatResult == nil {
		return
	}

	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 200}, false)

	// Result panel
	panelW := 320
	panelH := 160
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	// Panel color based on result
	if s.combatResult.AttackerWins {
		DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "⚔ VICTORY!")
	} else {
		DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "⚔ DEFEAT")
	}

	// Result text
	var resultText string
	var resultColor color.RGBA
	if s.combatResult.AttackerWins {
		resultText = "Attack Successful!"
		resultColor = ColorSuccess
	} else {
		resultText = "Attack Failed!"
		resultColor = ColorDanger
	}
	DrawLargeTextCentered(screen, resultText, ScreenWidth/2, panelY+55, resultColor)

	// Territory name
	DrawTextCentered(screen, s.combatResult.TargetName, ScreenWidth/2, panelY+80, ColorText)

	// Outcome description
	var outcomeText string
	if s.combatResult.AttackerWins {
		outcomeText = "Territory captured!"
	} else {
		outcomeText = "Your forces were repelled."
	}
	DrawTextCentered(screen, outcomeText, ScreenWidth/2, panelY+100, ColorTextMuted)

	// OK button
	s.dismissResultBtn.X = ScreenWidth/2 - 60
	s.dismissResultBtn.Y = panelY + panelH - 55
	s.dismissResultBtn.Draw(screen)
}

// doBuild executes a build action
func (s *GameplayScene) doBuild(buildType string) {
	if s.buildMenuTerritory == "" {
		return
	}

	// Determine if we should use gold (if we can't afford regular cost)
	useGold := false
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if stockpile, ok := player["stockpile"].(map[string]interface{}); ok {
			coal, gold, iron, timber := 0, 0, 0, 0
			if v, ok := stockpile["coal"].(float64); ok {
				coal = int(v)
			}
			if v, ok := stockpile["gold"].(float64); ok {
				gold = int(v)
			}
			if v, ok := stockpile["iron"].(float64); ok {
				iron = int(v)
			}
			if v, ok := stockpile["timber"].(float64); ok {
				timber = int(v)
			}

			switch buildType {
			case "city":
				if !(coal >= 1 && gold >= 1 && iron >= 1 && timber >= 1) {
					useGold = true
				}
			case "weapon":
				if !(coal >= 1 && iron >= 1) {
					useGold = true
				}
			case "boat":
				if timber < 3 {
					useGold = true
				}
			}
			_ = gold // Avoid unused variable warning
		}
	}

	log.Printf("Building %s at %s (useGold: %v)", buildType, s.buildMenuTerritory, useGold)
	s.game.Build(buildType, s.buildMenuTerritory, useGold)
	s.showBuildMenu = false
	s.buildMenuTerritory = ""
}

// ShowCombatResult displays the combat result popup
func (s *GameplayScene) ShowCombatResult(result *CombatResultData) {
	s.combatResult = result
	s.showCombatResult = true
}

func (s *GameplayScene) gridToScreen(gridX, gridY int) (int, int) {
	return s.offsetX + gridX*s.cellSize, s.offsetY + gridY*s.cellSize
}

func (s *GameplayScene) screenToGrid(screenX, screenY int) [2]int {
	if s.mapData == nil {
		return [2]int{-1, -1}
	}

	gridX := (screenX - s.offsetX) / s.cellSize
	gridY := (screenY - s.offsetY) / s.cellSize

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	if gridX < 0 || gridX >= width || gridY < 0 || gridY >= height {
		return [2]int{-1, -1}
	}

	return [2]int{gridX, gridY}
}

// SetGameState updates the game state from the server.
func (s *GameplayScene) SetGameState(state map[string]interface{}) {
	log.Println("GameplayScene.SetGameState called")
	s.gameState = state

	if mapData, ok := state["map"].(map[string]interface{}); ok {
		s.mapData = mapData
		log.Printf("Map data loaded: %dx%d", int(mapData["width"].(float64)), int(mapData["height"].(float64)))
	} else {
		log.Printf("No map data in state, keys: %v", getKeys(state))
	}

	if territories, ok := state["territories"].(map[string]interface{}); ok {
		s.territories = territories
		log.Printf("Loaded %d territories", len(territories))
	} else {
		log.Println("No territories in state")
	}

	if players, ok := state["players"].(map[string]interface{}); ok {
		s.players = players
		log.Printf("Loaded %d players", len(players))
	} else {
		log.Println("No players in state")
	}

	if playerOrder, ok := state["playerOrder"].([]interface{}); ok {
		s.playerOrder = playerOrder
		log.Printf("Loaded player order: %d players", len(playerOrder))
	} else {
		log.Println("No player order in state")
	}

	if phase, ok := state["phase"].(string); ok {
		// Clear selection when phase changes
		if s.currentPhase != phase {
			s.selectedTerritory = ""
		}
		s.currentPhase = phase
		log.Printf("Phase: %s", phase)
	}

	if turn, ok := state["currentPlayerId"].(string); ok {
		s.currentTurn = turn
		log.Printf("Current turn: %s", turn)
	}

	if round, ok := state["round"].(float64); ok {
		s.round = int(round)
		log.Printf("Round: %d", s.round)
	}
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}
