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
	sidebarW := 250

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
				isOnline := true // Default to online
				if onlineVal, ok := player["isOnline"].(bool); ok {
					isOnline = onlineVal
				}

				// Online/offline indicator (small circle)
				indicatorX := float32(sidebarX + 12)
				indicatorY := float32(y + 8)
				if isOnline {
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

// drawHistoryPanel draws the game history log (newest at top, scrollable).
func (s *GameplayScene) drawHistoryPanel(screen *ebiten.Image, x, y, w int) {
	// Calculate available height (fill remaining sidebar space, stopping before bottom bar)
	bottomBarTop := ScreenHeight - 110  // Bottom bar starts here
	availableH := bottomBarTop - y - 10 // Leave 10px margin above bottom bar
	if availableH < 100 {
		availableH = 100 // Minimum height
	}
	if availableH > 300 {
		availableH = 300 // Maximum height
	}

	// Store bounds for scroll detection
	s.historyPanelBounds = [4]int{x, y, w, availableH}

	DrawFancyPanel(screen, x, y, w, availableH, "History")

	if len(s.history) == 0 {
		DrawText(screen, "No events yet", x+12, y+40, ColorTextMuted)
		return
	}

	// Calculate how many events we can show
	lineHeight := 18
	headerHeight := 35
	maxLines := (availableH - headerHeight - 10) / lineHeight

	// Clamp scroll to valid range
	maxScroll := len(s.history) - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.historyScroll > maxScroll {
		s.historyScroll = maxScroll
	}
	if s.historyScroll < 0 {
		s.historyScroll = 0
	}

	// Draw events from newest to oldest (newest at top)
	eventY := y + headerHeight
	eventsDrawn := 0
	currentRound := -1
	currentPhase := ""

	// Iterate from newest to oldest, starting from scroll offset
	for i := len(s.history) - 1 - s.historyScroll; i >= 0 && eventsDrawn < maxLines; i-- {
		event := s.history[i]

		// Check if we need to show a round/phase header
		if event.Round != currentRound || event.Phase != currentPhase {
			currentRound = event.Round
			currentPhase = event.Phase

			// Draw phase header
			phaseText := fmt.Sprintf("Year %d %s", event.Round, event.Phase)
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

		// Build message with player name
		msg := event.Message
		if event.PlayerName != "" {
			msg = event.PlayerName + " " + msg
		}

		// Truncate message if too long
		if len(msg) > 32 {
			msg = msg[:29] + "..."
		}

		DrawText(screen, "  "+msg, x+10, eventY, textColor)
		eventY += lineHeight
		eventsDrawn++
	}

	// Draw scroll indicators if needed
	if s.historyScroll > 0 {
		DrawText(screen, "^", x+w-15, y+headerHeight, ColorTextMuted)
	}
	if s.historyScroll < maxScroll {
		DrawText(screen, "v", x+w-15, y+availableH-15, ColorTextMuted)
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
	sidebarWidth := 270    // Left sidebar width + margin
	bottomBarHeight := 120 // Bottom bar height + margin
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

// drawBottomBar draws phase/turn information with two sections.
func (s *GameplayScene) drawBottomBar(screen *ebiten.Image) {
	barX := 10
	barY := ScreenHeight - 110
	barW := ScreenWidth - 20
	barH := 100

	DrawFancyPanel(screen, barX, barY, barW, barH, "")

	// === LEFT SECTION: Year and Phase List ===
	leftSectionW := 280

	// Year display - large text on the left
	DrawLargeText(screen, fmt.Sprintf("Year %d", s.round), barX+15, barY+30, ColorText)

	// Phase list - vertical list to the right of Year
	// Order: Development (skipped Year 1) → Production → Trade → Shipment → Conquest
	phases := []string{"Development", "Production", "Trade", "Shipment", "Conquest"}
	phaseX := barX + 100 // Right of "Year X"
	phaseY := barY + 12
	lineHeight := 17

	// Check if we're in Territory Selection (only before year 1)
	if s.currentPhase == "Territory Selection" {
		// During territory selection, show it prominently
		displayText := "> Territory Selection"
		vector.DrawFilledRect(screen, float32(phaseX-2), float32(phaseY+28),
			float32(len(displayText)*7+4), float32(17), color.RGBA{40, 80, 40, 255}, false)
		DrawText(screen, displayText, phaseX, phaseY+30, ColorSuccess)
	} else {
		// Draw phase list vertically
		for _, phase := range phases {
			textColor := ColorTextMuted
			displayText := "  " + phase // Indent for non-current phases

			// Development is skipped in Year 1
			if phase == "Development" && s.round == 1 {
				displayText = "  " + phase + " (skipped)"
				textColor = ColorTextDim
			} else if phase == s.currentPhase {
				// Current phase - highlighted with arrow and background
				textColor = ColorSuccess
				displayText = "> " + phase
				// Draw highlight background
				vector.DrawFilledRect(screen, float32(phaseX-2), float32(phaseY-2),
					float32(len(displayText)*7+4), float32(lineHeight), color.RGBA{40, 80, 40, 255}, false)
			}

			DrawText(screen, displayText, phaseX, phaseY, textColor)
			phaseY += lineHeight
		}
	}

	// Vertical divider between sections
	dividerX := float32(barX + leftSectionW)
	vector.StrokeLine(screen, dividerX, float32(barY+10), dividerX, float32(barY+barH-10), 1, ColorBorder, false)

	// === RIGHT SECTION: Instructions and Controls ===
	rightX := barX + leftSectionW + 20
	isMyTurn := s.currentTurn == s.game.config.PlayerID

	// Turn indicator at top
	if s.currentTurn != "" {
		if player, ok := s.players[s.currentTurn].(map[string]interface{}); ok {
			playerName := player["name"].(string)
			playerColor := player["color"].(string)

			// Color indicator BEFORE turn text
			indicatorX := rightX
			if pc, ok := PlayerColors[playerColor]; ok {
				vector.DrawFilledRect(screen, float32(indicatorX), float32(barY+14), 14, 14, pc, false)
				vector.StrokeRect(screen, float32(indicatorX), float32(barY+14), 14, 14, 1, ColorBorder, false)
			}

			textX := rightX + 20 // After the color indicator
			if isMyTurn {
				DrawLargeText(screen, "YOUR TURN", textX, barY+12, ColorSuccess)
			} else {
				turnText := fmt.Sprintf("%s's turn", playerName)
				DrawText(screen, turnText, textX, barY+15, ColorText)
			}
		}
	}

	// Phase-specific instructions
	instruction := ""
	instruction2 := ""

	switch s.currentPhase {
	case "Territory Selection":
		if isMyTurn {
			instruction = "Click an unclaimed territory to claim it"
		} else {
			instruction = "Waiting for other player to select..."
		}

	case "Production":
		// Check if we need to place a stockpile (at start of round 1 or after losing one)
		needsStockpile := false
		if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
			player := myPlayer.(map[string]interface{})
			stockpileTerr, hasStockpile := player["stockpileTerritory"]
			needsStockpile = !hasStockpile || stockpileTerr == nil || stockpileTerr == ""
		}

		// Check if stockpile placement is pending (from game state)
		stockpilePlacementPending := false
		if s.gameState != nil {
			if pending, ok := s.gameState["stockpilePlacementPending"].(bool); ok {
				stockpilePlacementPending = pending
			}
		}

		if stockpilePlacementPending {
			if needsStockpile {
				instruction = "Click one of YOUR territories to place your stockpile"
				if s.round == 1 {
					instruction2 = "All players place stockpiles simultaneously"
				} else {
					instruction2 = "Your stockpile was captured - place a new one!"
				}
			} else {
				instruction = "Waiting for other players to place stockpiles..."
			}
		} else {
			instruction = "Resources are being produced automatically"
		}

	case "Trade":
		if isMyTurn {
			// Draw trade-specific controls
			s.drawTradeControls(screen, rightX, barY, barX+barW)
			return // Exit early - we draw our own instructions and buttons
		} else {
			instruction = "Waiting for other player to trade..."
		}

	case "Shipment":
		if isMyTurn {
			// Shipment controls are drawn separately
			s.drawShipmentControls(screen, rightX, barY, barX+barW)
			return // Exit early - we draw our own instructions
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
		DrawText(screen, instruction, rightX, barY+45, ColorTextMuted)
	}
	if instruction2 != "" {
		DrawText(screen, instruction2, rightX, barY+65, ColorTextMuted)
	}

	// End Turn button (right side, only during action phases and your turn)
	if isMyTurn && s.isActionPhase() {
		s.endPhaseBtn.X = barX + barW - 170
		s.endPhaseBtn.Y = barY + 30
		s.endPhaseBtn.Draw(screen)
	}
}

// drawShipmentControls draws shipment phase controls in the status bar
func (s *GameplayScene) drawShipmentControls(screen *ebiten.Image, startX, barY, endX int) {
	// Check what units we have available
	hasStockpile := false
	hasHorse := false
	hasBoat := false

	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if stockpileTerr, ok := player["stockpileTerritory"].(string); ok && stockpileTerr != "" {
			hasStockpile = true
		}
	}

	for _, terrData := range s.territories {
		terr := terrData.(map[string]interface{})
		owner := terr["owner"].(string)
		if owner != s.game.config.PlayerID {
			continue
		}
		if h, ok := terr["hasHorse"].(bool); ok && h {
			hasHorse = true
		}
		if boats, ok := terr["totalBoats"].(float64); ok && boats > 0 {
			hasBoat = true
		}
	}

	// Turn indicator at top with color block
	indicatorX := startX
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if playerColor, ok := player["color"].(string); ok {
			if pc, ok := PlayerColors[playerColor]; ok {
				vector.DrawFilledRect(screen, float32(indicatorX), float32(barY+14), 14, 14, pc, false)
				vector.StrokeRect(screen, float32(indicatorX), float32(barY+14), 14, 14, 1, ColorBorder, false)
			}
		}
	}
	DrawLargeText(screen, "YOUR TURN - SHIPMENT", startX+20, barY+12, ColorSuccess)

	btnY := barY + 40 // Moved down to avoid overlap with large text
	btnW := 100
	btnH := 26
	btnSpacing := 8

	if s.shipmentMode == "" {
		// Show mode selection buttons
		DrawText(screen, "Move:", startX, btnY+5, ColorText)

		btnX := startX + 50

		s.moveStockpileBtn.X = btnX
		s.moveStockpileBtn.Y = btnY
		s.moveStockpileBtn.W = btnW
		s.moveStockpileBtn.H = btnH
		s.moveStockpileBtn.Disabled = !hasStockpile
		s.moveStockpileBtn.Draw(screen)
		btnX += btnW + btnSpacing

		s.moveHorseBtn.X = btnX
		s.moveHorseBtn.Y = btnY
		s.moveHorseBtn.W = btnW
		s.moveHorseBtn.H = btnH
		s.moveHorseBtn.Disabled = !hasHorse
		s.moveHorseBtn.Draw(screen)
		btnX += btnW + btnSpacing

		s.moveBoatBtn.X = btnX
		s.moveBoatBtn.Y = btnY
		s.moveBoatBtn.W = btnW
		s.moveBoatBtn.H = btnH
		s.moveBoatBtn.Disabled = !hasBoat
		s.moveBoatBtn.Draw(screen)

		DrawText(screen, "Select what to move, or End Turn to skip", startX, barY+72, ColorTextMuted)
	} else {
		// Show current mode and selection status
		modeText := ""
		switch s.shipmentMode {
		case "stockpile":
			modeText = "Moving Stockpile"
		case "horse":
			modeText = "Moving Horse"
		case "boat":
			modeText = "Moving Boat"
		}
		DrawText(screen, modeText, startX, btnY+5, ColorPrimary)

		// Source info
		infoX := startX + 130
		if s.shipmentMode == "stockpile" {
			// Stockpile source is automatic
			stockpileLoc := ""
			if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
				player := myPlayer.(map[string]interface{})
				if stockpileTerr, ok := player["stockpileTerritory"].(string); ok {
					if terr, ok := s.territories[stockpileTerr].(map[string]interface{}); ok {
						stockpileLoc = terr["name"].(string)
					}
				}
			}
			DrawText(screen, fmt.Sprintf("From: %s", stockpileLoc), infoX, btnY+5, ColorText)
		} else {
			fromName := "(click source)"
			if s.shipmentFromTerritory != "" {
				if terr, ok := s.territories[s.shipmentFromTerritory].(map[string]interface{}); ok {
					fromName = terr["name"].(string)
				}
			}
			DrawText(screen, fmt.Sprintf("From: %s", fromName), infoX, btnY+5, ColorText)
		}

		// Destination info
		destName := "(click destination)"
		if s.selectedTerritory != "" {
			if terr, ok := s.territories[s.selectedTerritory].(map[string]interface{}); ok {
				destName = terr["name"].(string)
			}
		}
		DrawText(screen, fmt.Sprintf("To: %s", destName), infoX+180, btnY+5, ColorText)

		// Cargo checkboxes (for horse and boat) - below mode info
		checkboxY := barY + 60
		if s.shipmentMode == "horse" && s.shipmentFromTerritory != "" {
			if terr, ok := s.territories[s.shipmentFromTerritory].(map[string]interface{}); ok {
				if hasWeapon, _ := terr["hasWeapon"].(bool); hasWeapon {
					s.drawCheckbox(screen, startX, checkboxY, "Carry Weapon", &s.shipmentCarryWeapon)
				}
			}
		} else if s.shipmentMode == "boat" && s.shipmentFromTerritory != "" {
			if terr, ok := s.territories[s.shipmentFromTerritory].(map[string]interface{}); ok {
				hasHorseInTerr, _ := terr["hasHorse"].(bool)
				hasWeaponInTerr, _ := terr["hasWeapon"].(bool)

				cbX := startX
				if hasHorseInTerr {
					s.drawCheckbox(screen, cbX, checkboxY, "Load Horse", &s.shipmentCarryHorse)
					cbX += 120
				}
				if hasWeaponInTerr {
					s.drawCheckbox(screen, cbX, checkboxY, "Load Weapon", &s.shipmentCarryWeapon)
				}
			}
		}

		// Confirm and Cancel buttons - far right, below End Turn
		s.shipmentConfirmBtn.X = startX + 350
		s.shipmentConfirmBtn.Y = barY + 60
		s.shipmentConfirmBtn.W = 90
		s.shipmentConfirmBtn.H = btnH
		s.shipmentConfirmBtn.Disabled = s.selectedTerritory == "" || (s.shipmentMode != "stockpile" && s.shipmentFromTerritory == "")
		s.shipmentConfirmBtn.Draw(screen)

		s.cancelShipmentBtn.X = startX + 448
		s.cancelShipmentBtn.Y = barY + 60
		s.cancelShipmentBtn.W = 70
		s.cancelShipmentBtn.H = btnH
		s.cancelShipmentBtn.Text = "Cancel"
		s.cancelShipmentBtn.Draw(screen)
	}

	// End Turn button (always visible, right side)
	s.endPhaseBtn.X = endX - 170
	s.endPhaseBtn.Y = barY + 30
	s.endPhaseBtn.Draw(screen)
}

// drawTradeControls draws the trade phase controls in the status bar.
func (s *GameplayScene) drawTradeControls(screen *ebiten.Image, startX, barY, endX int) {
	// Turn indicator with color block
	indicatorX := startX
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if playerColor, ok := player["color"].(string); ok {
			if pc, ok := PlayerColors[playerColor]; ok {
				vector.DrawFilledRect(screen, float32(indicatorX), float32(barY+14), 14, 14, pc, false)
				vector.StrokeRect(screen, float32(indicatorX), float32(barY+14), 14, 14, 1, ColorBorder, false)
			}
		}
	}
	DrawLargeText(screen, "YOUR TURN - TRADE", startX+20, barY+12, ColorSuccess)
	DrawText(screen, "Propose trades or End Turn to skip", startX, barY+45, ColorTextMuted)

	// Propose Trade button - below the instruction text
	s.proposeTradeBtn.X = startX
	s.proposeTradeBtn.Y = barY + 65
	s.proposeTradeBtn.Draw(screen)

	// End Turn button
	s.endPhaseBtn.X = endX - 170
	s.endPhaseBtn.Y = barY + 30
	s.endPhaseBtn.Draw(screen)
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
