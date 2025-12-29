package client

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
	infoPanel   *Panel
	actionPanel *Panel
}

// Panel is a UI panel.
type Panel struct {
	X, Y, W, H int
}

// NewGameplayScene creates a new gameplay scene.
func NewGameplayScene(game *Game) *GameplayScene {
	return &GameplayScene{
		game:        game,
		cellSize:    32,
		offsetX:     20,
		offsetY:     80,
		hoveredCell: [2]int{-1, -1},
	}
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

	// Update hovered cell
	mx, my := ebiten.CursorPosition()
	s.hoveredCell = s.screenToGrid(mx, my)

	// Handle click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if s.hoveredCell[0] >= 0 {
			s.handleCellClick(s.hoveredCell[0], s.hoveredCell[1])
		}
	}

	return nil
}

func (s *GameplayScene) Draw(screen *ebiten.Image) {
	if s.mapData == nil {
		DrawTextCentered(screen, "Loading game...", ScreenWidth/2, ScreenHeight/2, ColorText)
		return
	}

	// Draw map
	s.drawMap(screen)

	// Draw player identity panel
	s.drawPlayerIdentityPanel(screen)

	// Draw info panel
	s.drawInfoPanel(screen)

	// Draw resource panel
	s.drawResourcePanel(screen)

	// Draw players list panel
	s.drawPlayersPanel(screen)

	// Draw hover info
	if s.hoveredCell[0] >= 0 {
		s.drawHoverInfo(screen)
	}
}

func (s *GameplayScene) drawMap(screen *ebiten.Image) {
	if s.mapData == nil {
		return
	}

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))
	grid := s.mapData["grid"].([]interface{})

	// Draw territories
	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			territoryID := int(row[x].(float64))
			
			sx, sy := s.gridToScreen(x, y)
			
			// Determine color
			var cellColor color.RGBA
			if territoryID == 0 {
				// Water
				cellColor = color.RGBA{20, 60, 120, 255}
			} else {
				// Land - get owner color
				tid := fmt.Sprintf("t%d", territoryID)
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

			// Highlight hovered cell
			if x == s.hoveredCell[0] && y == s.hoveredCell[1] {
				cellColor.R = min(cellColor.R+40, 255)
				cellColor.G = min(cellColor.G+40, 255)
				cellColor.B = min(cellColor.B+40, 255)
			}

			vector.DrawFilledRect(screen, float32(sx), float32(sy), 
				float32(s.cellSize-1), float32(s.cellSize-1), cellColor, false)
		}
	}

	// Draw territory boundaries
	s.drawTerritoryBoundaries(screen, width, height, grid)

	// Draw stockpile indicators
	s.drawStockpileIndicators(screen)
}

// drawTerritoryBoundaries draws lines between different territories
func (s *GameplayScene) drawTerritoryBoundaries(screen *ebiten.Image, width, height int, grid []interface{}) {
	borderColor := color.RGBA{0, 0, 0, 180}

	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			territoryID := int(row[x].(float64))
			sx, sy := s.gridToScreen(x, y)

			// Check right neighbor
			if x < width-1 {
				rightRow := grid[y].([]interface{})
				rightID := int(rightRow[x+1].(float64))
				if rightID != territoryID {
					// Draw vertical line
					x1 := float32(sx + s.cellSize - 1)
					y1 := float32(sy)
					y2 := float32(sy + s.cellSize)
					vector.StrokeLine(screen, x1, y1, x1, y2, 2, borderColor, false)
				}
			}

			// Check bottom neighbor
			if y < height-1 {
				bottomRow := grid[y+1].([]interface{})
				bottomID := int(bottomRow[x].(float64))
				if bottomID != territoryID {
					// Draw horizontal line
					x1 := float32(sx)
					x2 := float32(sx + s.cellSize)
					y1 := float32(sy + s.cellSize - 1)
					vector.StrokeLine(screen, x1, y1, x2, y1, 2, borderColor, false)
				}
			}
		}
	}
}

// drawStockpileIndicators shows where each player's stockpile is located
func (s *GameplayScene) drawStockpileIndicators(screen *ebiten.Image) {
	if s.mapData == nil || s.players == nil {
		return
	}

	grid := s.mapData["grid"].([]interface{})

	for _, playerData := range s.players {
		player := playerData.(map[string]interface{})
		
		// Check if player has stockpile placed
		stockpileTerr, hasStockpile := player["stockpileTerritory"]
		if !hasStockpile || stockpileTerr == nil || stockpileTerr == "" {
			continue
		}

		stockpileID := stockpileTerr.(string)
		
		// Find the center of this territory to place the icon
		centerX, centerY := s.findTerritoryCenter(stockpileID, grid)
		if centerX < 0 {
			continue
		}

		sx, sy := s.gridToScreen(centerX, centerY)
		
		// Draw stockpile icon (a small box/crate)
		iconSize := float32(s.cellSize / 3)
		iconX := float32(sx) + float32(s.cellSize)/2 - iconSize/2
		iconY := float32(sy) + float32(s.cellSize)/2 - iconSize/2
		
		// Stockpile color (golden brown)
		stockpileColor := color.RGBA{200, 160, 80, 255}
		vector.DrawFilledRect(screen, iconX, iconY, iconSize, iconSize, stockpileColor, false)
		vector.StrokeRect(screen, iconX, iconY, iconSize, iconSize, 1.5, 
			color.RGBA{150, 100, 30, 255}, false)
		
		// Add player's first initial on it
		if name, ok := player["name"].(string); ok && len(name) > 0 {
			initial := string(name[0])
			// Note: This uses the debug font, ideally we'd use a better font
			DrawTextCentered(screen, initial, int(iconX+iconSize/2), int(iconY+iconSize/2-4), 
				color.RGBA{50, 30, 10, 255})
		}
	}
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
	// Only show if we have player data
	myPlayer, ok := s.players[s.game.config.PlayerID]
	if !ok {
		return
	}

	player := myPlayer.(map[string]interface{})
	playerName := player["name"].(string)
	playerColor := player["color"].(string)

	// Panel in top-left corner
	panelX := 20
	panelY := 20
	panelW := 200
	panelH := 60

	DrawPanel(screen, panelX, panelY, panelW, panelH)

	// "You" label
	DrawText(screen, "You:", panelX+10, panelY+8, ColorTextMuted)

	// Player name
	DrawText(screen, playerName, panelX+10, panelY+26, ColorText)

	// Color indicator (large square)
	if pc, ok := PlayerColors[playerColor]; ok {
		colorSize := float32(30)
		colorX := float32(panelX + panelW - 40)
		colorY := float32(panelY + 15)
		vector.DrawFilledRect(screen, colorX, colorY, colorSize, colorSize, pc, false)
		vector.StrokeRect(screen, colorX, colorY, colorSize, colorSize, 2, 
			color.RGBA{0, 0, 0, 120}, false)
	}

	// Color name
	DrawText(screen, playerColor, panelX+10, panelY+44, ColorTextMuted)
}

func (s *GameplayScene) drawInfoPanel(screen *ebiten.Image) {
	panelX := 240
	panelY := 20
	panelW := 800
	panelH := 50

	DrawPanel(screen, panelX, panelY, panelW, panelH)

	// Phase and round
	phaseText := fmt.Sprintf("Round %d - %s", s.round, s.currentPhase)
	DrawText(screen, phaseText, panelX+10, panelY+10, ColorText)

	// Phase-specific instructions
	instruction := ""
	showTurnIndicator := true
	
	if s.currentPhase == "Territory Selection" {
		instruction = "Click to claim territories"
	} else if s.currentPhase == "Production" && s.round == 1 {
		instruction = "Click YOUR territory to place stockpile"
		showTurnIndicator = false // Stockpile placement is async, everyone does it
	} else if s.currentPhase == "Shipment" {
		instruction = "Move units between territories"
		// Shipment is turn-based
	}
	
	if instruction != "" {
		DrawText(screen, instruction, panelX+250, panelY+10, ColorTextMuted)
	}

	// Current turn (only for turn-based phases)
	if showTurnIndicator && s.currentTurn != "" {
		if player, ok := s.players[s.currentTurn].(map[string]interface{}); ok {
			playerName := player["name"].(string)
			playerColor := player["color"].(string)
			
			turnText := fmt.Sprintf("Turn: %s", playerName)
			DrawText(screen, turnText, panelX+10, panelY+28, ColorText)

			// Color indicator
			if pc, ok := PlayerColors[playerColor]; ok {
				vector.DrawFilledRect(screen, float32(panelX+60), float32(panelY+30), 
					12, 12, pc, false)
			}

			// Indicate if it's your turn
			if s.currentTurn == s.game.config.PlayerID {
				DrawText(screen, "YOUR TURN!", panelX+120, panelY+28, ColorSuccess)
			}
		}
	}
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
		
		// Draw info box near cursor
		boxX := mx + 15
		boxY := my + 15
		boxW := 200
		boxH := 60

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

		owner := terr["owner"].(string)
		if owner != "" {
			if player, ok := s.players[owner].(map[string]interface{}); ok {
				playerName := player["name"].(string)
				DrawText(screen, "Owner: "+playerName, boxX+10, boxY+28, ColorTextMuted)
			}
		} else {
			DrawText(screen, "Unclaimed", boxX+10, boxY+28, ColorTextMuted)
		}

		resource := terr["resource"].(string)
		if resource != "None" {
			DrawText(screen, "Resource: "+resource, boxX+10, boxY+42, ColorTextMuted)
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
	case "Shipment":
		s.handleShipment(tid)
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

func (s *GameplayScene) handleShipment(territoryID string) {
	// TODO: Implement shipment phase logic
	// For now, just log
	log.Printf("Shipment phase - clicked territory: %s", territoryID)
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
		s.currentPhase = phase
		log.Printf("Phase: %s", phase)
	}
	
	if turn, ok := state["currentPlayerId"].(string); ok {
		s.currentTurn = turn
		log.Printf("Current turn: %s", turn)
	}
	
	if round, ok := state["round"].(float64); ok {
		s.round = int(round)
		log.Printf("Round: %d", round)
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

// drawResourcePanel shows the player's current resources
func (s *GameplayScene) drawResourcePanel(screen *ebiten.Image) {
	// Only show for current player
	myPlayer, ok := s.players[s.game.config.PlayerID]
	if !ok {
		return
	}

	player := myPlayer.(map[string]interface{})
	
	// Panel position (top right)
	panelX := ScreenWidth - 220
	panelY := 20
	panelW := 200
	panelH := 130

	DrawPanel(screen, panelX, panelY, panelW, panelH)

	// Title
	DrawText(screen, "Your Resources", panelX+10, panelY+8, ColorText)

	// Get stockpile data
	stockpile, hasStockpile := player["stockpile"]
	if !hasStockpile {
		DrawText(screen, "No stockpile yet", panelX+10, panelY+28, ColorTextMuted)
		return
	}

	stockpileData := stockpile.(map[string]interface{})
	
	// Draw resources
	y := panelY + 28
	resources := []struct {
		name  string
		key   string
		icon  string
	}{
		{"Coal", "coal", "C"},
		{"Gold", "gold", "G"},
		{"Iron", "iron", "I"},
		{"Wood", "timber", "W"},
	}

	for _, res := range resources {
		count := 0
		if val, ok := stockpileData[res.key]; ok {
			count = int(val.(float64))
		}
		
		text := fmt.Sprintf("%s: %d", res.name, count)
		DrawText(screen, text, panelX+10, y, ColorText)
		y += 20
	}

	// Show stockpile location
	if stockpileTerr, ok := player["stockpileTerritory"]; ok && stockpileTerr != nil && stockpileTerr != "" {
		terrID := stockpileTerr.(string)
		if terr, ok := s.territories[terrID]; ok {
			terrData := terr.(map[string]interface{})
			terrName := terrData["name"].(string)
			DrawText(screen, fmt.Sprintf("At: %s", terrName), panelX+10, y, ColorTextMuted)
		}
	}
}

// drawPlayersPanel shows all players in the game
func (s *GameplayScene) drawPlayersPanel(screen *ebiten.Image) {
	if s.players == nil || len(s.players) == 0 {
		return
	}

	// Panel position (below identity panel)
	panelX := 20
	panelY := 90
	panelW := 200
	panelH := 40 + len(s.players)*24

	DrawPanel(screen, panelX, panelY, panelW, panelH)

	// Title
	DrawText(screen, "Players", panelX+10, panelY+8, ColorText)

	// List players in turn order
	y := panelY + 28
	for _, playerIDInterface := range s.playerOrder {
		playerID := playerIDInterface.(string)
		if playerData, ok := s.players[playerID]; ok {
			player := playerData.(map[string]interface{})
			playerName := player["name"].(string)
			playerColor := player["color"].(string)
			isAI := player["isAI"].(bool)

			// Color indicator
			if pc, ok := PlayerColors[playerColor]; ok {
				vector.DrawFilledRect(screen, float32(panelX+10), float32(y+2), 12, 12, pc, false)
			}

			// Player name
			nameText := playerName
			if isAI {
				nameText += " (AI)"
			}
			if playerID == s.game.config.PlayerID {
				nameText += " ★"
			}
			
			DrawText(screen, nameText, panelX+28, y, ColorText)
			y += 24
		}
	}
}

