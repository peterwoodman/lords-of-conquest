package client

import (
	"fmt"
	"image/color"

	"lords-of-conquest/internal/protocol"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
	playersH := 40 + playerCount*26 + 40 // Header + per-player height + space for Set Ally button
	if playersH > 240 {
		playersH = 240 // Cap max height
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
				isOnline := true // Default to online
				if onlineVal, ok := player["isOnline"].(bool); ok {
					isOnline = onlineVal
				}

				// Online/offline indicator (small circle)
				indicatorX := float32(sidebarX + 12)
				indicatorY := float32(y + 8)
				if isAI {
					// AI is always "online" - gray indicator
					vector.DrawFilledCircle(screen, indicatorX, indicatorY, 4, color.RGBA{128, 128, 128, 255}, false)
				} else if isOnline {
					// Online - green
					vector.DrawFilledCircle(screen, indicatorX, indicatorY, 4, color.RGBA{100, 200, 100, 255}, false)
				} else {
					// Offline - red
					vector.DrawFilledCircle(screen, indicatorX, indicatorY, 4, color.RGBA{200, 100, 100, 255}, false)
				}

				// Color indicator (moved right to make room for online indicator)
				if pc, ok := PlayerColors[playerColor]; ok {
					vector.DrawFilledRect(screen, float32(sidebarX+22), float32(y+2), 14, 14, pc, false)
					vector.StrokeRect(screen, float32(sidebarX+22), float32(y+2), 14, 14, 1, ColorBorder, false)
				}

				// Player name
				nameText := playerName
				if isAI {
					nameText += " (AI)"
				}
				if playerID == s.game.config.PlayerID {
					nameText += " *"
				}

				DrawText(screen, nameText, sidebarX+42, y, ColorText)
				y += 26

				if y > playersY+playersH-50 {
					break // Don't overflow, leave room for button
				}
			}
		}

		// Set Ally button at bottom of panel (only show if 3+ players)
		if playerCount >= 3 {
			s.setAllyBtn.X = sidebarX + 10
			s.setAllyBtn.Y = playersY + playersH - 38
			s.setAllyBtn.W = sidebarW - 20
			s.setAllyBtn.Draw(screen)
			s.setAllyBtn.Update()
		}
	}

	// Resources panel - below Players
	resourcesY := playersY + playersH + 15
	s.drawResourcesPanel(screen, sidebarX, resourcesY, sidebarW)

	// History panel - below Resources
	historyY := resourcesY + 185 // Resources panel is 170 + 15 margin
	s.drawHistoryPanel(screen, sidebarX, historyY, sidebarW)
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
			name    string
			key     string
			iconKey string
		}{
			{"Coal", "coal", "coal"},
			{"Gold", "gold", "gold"},
			{"Iron", "iron", "iron"},
			{"Wood", "timber", "timber"},
		}

		for _, res := range resources {
			count := 0
			if val, ok := stockpileData[res.key]; ok {
				count = int(val.(float64))
			}

			// Resource icon with light background for visibility
			iconSize := 16
			if icon := GetIcon(res.iconKey); icon != nil {
				// Draw light background behind black icon
				iconX := float32(x + 11)
				iconY := float32(resY)
				vector.DrawFilledRect(screen, iconX-1, iconY-1, float32(iconSize+2), float32(iconSize+2),
					color.RGBA{180, 180, 180, 255}, false)
				vector.StrokeRect(screen, iconX-1, iconY-1, float32(iconSize+2), float32(iconSize+2),
					1, color.RGBA{100, 100, 100, 255}, false)

				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(iconX), float64(iconY))
				screen.DrawImage(icon, op)
			}

			text := fmt.Sprintf("%s: %d", res.name, count)
			DrawText(screen, text, x+11+iconSize+6, resY, ColorText)
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

// drawHistoryPanel draws the game history log.
func (s *GameplayScene) drawHistoryPanel(screen *ebiten.Image, x, y, w int) {
	// Calculate available height (fill remaining sidebar space)
	availableH := ScreenHeight - y - 20
	if availableH < 100 {
		availableH = 100 // Minimum height
	}
	if availableH > 300 {
		availableH = 300 // Maximum height
	}

	DrawFancyPanel(screen, x, y, w, availableH, "History")

	if len(s.history) == 0 {
		DrawText(screen, "No events yet", x+12, y+40, ColorTextMuted)
		return
	}

	// Calculate how many events we can show
	lineHeight := 18
	headerHeight := 35
	maxLines := (availableH - headerHeight - 10) / lineHeight

	// Start from the bottom of the list (most recent), scrolling up
	startIdx := len(s.history) - 1 - s.historyScroll
	if startIdx < 0 {
		startIdx = 0
	}

	// Draw events from bottom to top (most recent at bottom)
	eventY := y + headerHeight
	eventsDrawn := 0
	currentRound := -1
	currentPhase := ""

	// We need to show events in chronological order with most recent at bottom
	// So we iterate from older to newer, but start from an offset that fits our display
	displayStart := len(s.history) - maxLines - s.historyScroll
	if displayStart < 0 {
		displayStart = 0
	}

	for i := displayStart; i < len(s.history) && eventsDrawn < maxLines; i++ {
		event := s.history[i]

		// Check if we need to show a round/phase header
		if event.Round != currentRound || event.Phase != currentPhase {
			currentRound = event.Round
			currentPhase = event.Phase

			// Draw phase header
			phaseText := fmt.Sprintf("R%d %s", event.Round, event.Phase)
			DrawText(screen, phaseText, x+10, eventY, ColorPrimary)
			eventY += lineHeight
			eventsDrawn++

			if eventsDrawn >= maxLines {
				break
			}
		}

		// Draw event message with player color if available
		textColor := ColorText
		if event.PlayerID != "" {
			if player, ok := s.players[event.PlayerID]; ok {
				playerData := player.(map[string]interface{})
				if colorName, ok := playerData["color"].(string); ok {
					if pc, ok := PlayerColors[colorName]; ok {
						textColor = pc
					}
				}
			}
		}

		// Truncate message if too long
		msg := event.Message
		if len(msg) > 24 {
			msg = msg[:21] + "..."
		}

		DrawText(screen, "  "+msg, x+10, eventY, textColor)
		eventY += lineHeight
		eventsDrawn++
	}

	// Draw scroll indicators if needed
	if s.historyScroll > 0 {
		DrawText(screen, "▼", x+w-20, y+availableH-15, ColorTextMuted)
	}
	if displayStart > 0 {
		DrawText(screen, "▲", x+w-20, y+headerHeight, ColorTextMuted)
	}
}

// SetHistory updates the game history from the server.
func (s *GameplayScene) SetHistory(events []protocol.HistoryEvent) {
	s.history = make([]HistoryEntry, len(events))
	for i, e := range events {
		s.history[i] = HistoryEntry{
			ID:         e.ID,
			Round:      e.Round,
			Phase:      e.Phase,
			PlayerID:   e.PlayerID,
			PlayerName: e.PlayerName,
			EventType:  e.EventType,
			Message:    e.Message,
		}
	}
	// Reset scroll to show newest events
	s.historyScroll = 0
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

		// Boats (using totalBoats for display)
		if totalBoats, ok := terr["totalBoats"].(float64); ok && int(totalBoats) > 0 {
			boatCount := int(totalBoats)
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
			if tb, ok := terr["totalBoats"].(float64); ok {
				boats = int(tb)
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
