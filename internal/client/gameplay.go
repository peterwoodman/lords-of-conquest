package client

import (
	"fmt"
	"image/color"

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
	if s.gameState == nil {
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
	if s.gameState == nil {
		DrawTextCentered(screen, "Loading game...", ScreenWidth/2, ScreenHeight/2, ColorText)
		return
	}

	// Draw map
	s.drawMap(screen)

	// Draw info panel
	s.drawInfoPanel(screen)

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
}

func (s *GameplayScene) drawInfoPanel(screen *ebiten.Image) {
	panelX := 20
	panelY := 20
	panelW := 400
	panelH := 50

	DrawPanel(screen, panelX, panelY, panelW, panelH)

	// Phase and round
	phaseText := fmt.Sprintf("Round %d - %s", s.round, s.currentPhase)
	DrawText(screen, phaseText, panelX+10, panelY+10, ColorText)

	// Current turn
	if s.currentTurn != "" {
		if player, ok := s.players[s.currentTurn].(map[string]interface{}); ok {
			playerName := player["name"].(string)
			playerColor := player["color"].(string)
			
			turnText := fmt.Sprintf("Current Turn: %s", playerName)
			DrawText(screen, turnText, panelX+10, panelY+28, ColorText)

			// Color indicator
			if pc, ok := PlayerColors[playerColor]; ok {
				vector.DrawFilledRect(screen, float32(panelX+120), float32(panelY+30), 
					12, 12, pc, false)
			}

			// Indicate if it's your turn
			if s.currentTurn == s.game.config.PlayerID {
				DrawText(screen, "(YOUR TURN)", panelX+240, panelY+28, ColorSuccess)
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
	if s.mapData == nil || s.currentPhase != "Territory Selection" {
		return
	}

	// Check if it's our turn
	if s.currentTurn != s.game.config.PlayerID {
		return
	}

	grid := s.mapData["grid"].([]interface{})
	row := grid[y].([]interface{})
	territoryID := int(row[x].(float64))

	if territoryID == 0 {
		return // Water
	}

	tid := fmt.Sprintf("t%d", territoryID)
	if terr, ok := s.territories[tid].(map[string]interface{}); ok {
		owner := terr["owner"].(string)
		if owner == "" {
			// Unclaimed, we can select it
			s.game.SelectTerritory(tid)
		}
	}
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
	s.gameState = state
	
	if mapData, ok := state["map"].(map[string]interface{}); ok {
		s.mapData = mapData
	}
	
	if territories, ok := state["territories"].(map[string]interface{}); ok {
		s.territories = territories
	}
	
	if players, ok := state["players"].(map[string]interface{}); ok {
		s.players = players
	}
	
	if phase, ok := state["phase"].(string); ok {
		s.currentPhase = phase
	}
	
	if turn, ok := state["currentPlayerId"].(string); ok {
		s.currentTurn = turn
	}
	
	if round, ok := state["round"].(float64); ok {
		s.round = int(round)
	}
}

func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

