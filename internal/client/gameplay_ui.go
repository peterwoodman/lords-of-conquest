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

	// Player identity panel (with resources)
	myPlayer, ok := s.players[s.game.config.PlayerID]
	youPanelH := 130 // Taller panel to fit everything
	if ok {
		player := myPlayer.(map[string]interface{})
		playerName := player["name"].(string)
		playerColor := player["color"].(string)

		DrawFancyPanel(screen, sidebarX, sidebarY, sidebarW, youPanelH, "You")

		// Player name - below title bar (title bar is ~30px)
		DrawLargeText(screen, playerName, sidebarX+15, sidebarY+40, ColorText)

		// Color indicator (clickable)
		if pc, ok := PlayerColors[playerColor]; ok {
			colorSize := float32(24)
			colorX := float32(sidebarX + sidebarW - 40)
			colorY := float32(sidebarY + 38)
			vector.DrawFilledRect(screen, colorX, colorY, colorSize, colorSize, pc, false)
			vector.StrokeRect(screen, colorX, colorY, colorSize, colorSize, 2, ColorBorder, false)
			// Store bounds for click detection
			s.myColorBlockBounds = [4]int{int(colorX), int(colorY), int(colorSize), int(colorSize)}
		}

		// Resources in 2 columns below name (with icons)
		coal, gold, iron, timber := 0, 0, 0, 0
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

		resY := sidebarY + 75
		col1X := sidebarX + 12
		col2X := sidebarX + sidebarW/2 + 5
		iconSize := 16

		// Helper to draw resource with inverted (white) icon
		drawResource := func(iconKey string, count int, x, y int) {
			if icon := GetIcon(iconKey); icon != nil {
				DrawIconInverted(screen, icon, x, y, iconSize)
			}
			DrawText(screen, fmt.Sprintf("%d", count), x+iconSize+6, y, ColorText)
		}

		// Row 1: Coal and Gold
		drawResource("coal", coal, col1X, resY)
		drawResource("gold", gold, col2X, resY)

		// Row 2: Iron and Wood
		resY += 24
		drawResource("iron", iron, col1X, resY)
		drawResource("timber", timber, col2X, resY)
	}

	// Players list - compact height based on player count
	playersY := sidebarY + youPanelH + 5 // Below the "You" panel with some spacing
	playerCount := len(s.playerOrder)
	playersH := 40 + playerCount*26 + 40 // Header + per-player height + space for Set Ally button
	if playersH > 240 {
		playersH = 240 // Cap max height
	}

	if playerCount > 0 {
		DrawFancyPanel(screen, sidebarX, playersY, sidebarW, playersH, "Players")

		// Count cities per player
		cityCounts := make(map[string]int)
		for _, terrData := range s.territories {
			terr := terrData.(map[string]interface{})
			owner := terr["owner"].(string)
			if owner != "" {
				if hasCity, ok := terr["hasCity"].(bool); ok && hasCity {
					cityCounts[owner]++
				}
			}
		}

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

				// Player name with city count if > 0
				nameText := playerName
				if cityCount := cityCounts[playerID]; cityCount > 0 {
					nameText += fmt.Sprintf(" (%d)", cityCount)
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
		}
	}

	// History panel - directly below Players (no separate Resources panel)
	historyY := playersY + playersH + 15
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

			// Draw phase header - Year 0 is Territory Selection (setup), Year 1+ are normal years
			var phaseText string
			if event.Round == 0 {
				phaseText = event.Phase // Just show "Territory Selection" without year
			} else {
				phaseText = fmt.Sprintf("Year %d %s", event.Round, event.Phase)
			}
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

	// Calculate base cell size to fit the map in available space (before zoom)
	cellW := availableWidth / width
	cellH := availableHeight / height
	baseCellSize := cellW
	if cellH < cellW {
		baseCellSize = cellH
	}
	// Clamp base cell size to reasonable bounds
	if baseCellSize < 8 {
		baseCellSize = 8
	}
	if baseCellSize > 40 {
		baseCellSize = 40
	}

	// Apply zoom to cell size
	s.cellSize = int(float64(baseCellSize) * s.zoom)
	if s.cellSize < 4 {
		s.cellSize = 4
	}

	// Map dimensions with zoomed cell size
	mapW := width * s.cellSize
	mapH := height * s.cellSize

	// Center the map in available space, then apply pan offset
	baseOffsetX := sidebarWidth + (availableWidth-mapW)/2
	baseOffsetY := 10 + (availableHeight-mapH)/2
	s.offsetX = baseOffsetX + s.panX
	s.offsetY = baseOffsetY + s.panY

	// Draw the map
	s.drawMap(screen)

	// Show zoom indicator if not at default zoom/pan
	if s.zoom != 1.0 || s.panX != 0 || s.panY != 0 {
		zoomText := fmt.Sprintf("Zoom: %.0f%% (Home to reset)", s.zoom*100)
		DrawText(screen, zoomText, sidebarWidth+10, ScreenHeight-bottomBarHeight-25, ColorTextMuted)
	}
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

			if phase == s.currentPhase {
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

	// Check for stockpile placement first (happens during Production phase)
	// During stockpile placement, ALL players place simultaneously (not turn-based)
	stockpilePlacementPending := false
	needsStockpile := false
	if s.currentPhase == "Production" {
		if s.gameState != nil {
			if pending, ok := s.gameState["stockpilePlacementPending"].(bool); ok {
				stockpilePlacementPending = pending
			}
		}
		if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
			player := myPlayer.(map[string]interface{})
			stockpileTerr, hasStockpile := player["stockpileTerritory"]
			needsStockpile = !hasStockpile || stockpileTerr == nil || stockpileTerr == ""
		}
	}

	// If stockpile placement is pending, show special UI instead of turn indicator
	if stockpilePlacementPending {
		if needsStockpile {
			// Draw a prominent instruction box for stockpile placement
			boxX := rightX - 10
			boxY := barY + 10
			boxW := 420
			boxH := 70
			vector.DrawFilledRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), color.RGBA{60, 40, 80, 255}, false)
			vector.StrokeRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), 2, ColorWarning, false)

			DrawLargeText(screen, "PLACE YOUR STOCKPILE", rightX, barY+18, ColorWarning)
			DrawText(screen, "Click one of YOUR territories to place your stockpile", rightX, barY+45, ColorText)
			if s.round == 1 {
				DrawText(screen, "", rightX, barY+62, ColorTextMuted)
			} else {
				DrawText(screen, "Your stockpile was captured - place a new one!", rightX, barY+62, ColorTextMuted)
			}
		} else {
			DrawText(screen, "Waiting for other players to place stockpiles...", rightX, barY+25, ColorTextMuted)
		}
		// End Turn button not shown during stockpile placement
		return
	}

	// Check if current player is eliminated (surrendered)
	amEliminated := false
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if eliminated, ok := player["eliminated"].(bool); ok && eliminated {
			amEliminated = true
		}
	}

	// If player has surrendered, show special message
	if amEliminated {
		// Draw a muted message
		boxX := rightX - 10
		boxY := barY + 10
		boxW := 350
		boxH := 60
		vector.DrawFilledRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), color.RGBA{60, 40, 40, 255}, false)
		vector.StrokeRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), 2, ColorTextMuted, false)

		DrawLargeText(screen, "YOU HAVE SURRENDERED", rightX, barY+18, ColorTextMuted)
		DrawText(screen, "You may continue watching the game.", rightX, barY+45, ColorTextDim)
		return
	}

	// Normal turn indicator (not during stockpile placement)
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

	// Pending horse selection mode (takes priority over phase instructions)
	if s.pendingHorseSelection != "" {
		var horseInstruction string
		var selectedCount int
		switch s.pendingHorseSelection {
		case "offer":
			// Proposer selecting territories to give horses FROM
			selectedCount = len(s.tradeOfferHorseTerrs)
			horseInstruction = fmt.Sprintf("Click %d territory(s) WITH horses to trade (%d/%d selected)",
				s.pendingHorseCount, selectedCount, s.pendingHorseCount)
		case "request":
			// Proposer selecting territories to RECEIVE requested horses ON
			selectedCount = len(s.tradeRequestHorseDestTerrs)
			horseInstruction = fmt.Sprintf("Click %d territory(s) WITHOUT horses to receive them (%d/%d selected)",
				s.pendingHorseCount, selectedCount, s.pendingHorseCount)
		case "receive":
			// Accepter selecting territories to place offered horses ON
			selectedCount = len(s.tradeHorseDestTerrs)
			horseInstruction = fmt.Sprintf("Click %d territory(s) WITHOUT horses to receive them (%d/%d selected)",
				s.pendingHorseCount, selectedCount, s.pendingHorseCount)
		case "give":
			// Accepter selecting territories to give horses FROM
			selectedCount = len(s.tradeHorseSourceTerrs)
			horseInstruction = fmt.Sprintf("Click %d territory(s) WITH horses to give (%d/%d selected)",
				s.pendingHorseCount, selectedCount, s.pendingHorseCount)
		}
		DrawText(screen, horseInstruction, rightX, barY+35, ColorWarning)

		// Draw buttons
		btnY := barY + 55
		s.horseCancelBtn.X = rightX
		s.horseCancelBtn.Y = btnY
		s.horseCancelBtn.W = 100
		s.horseCancelBtn.H = 35
		s.horseCancelBtn.Draw(screen)

		// Only show Confirm button when selection is complete
		if selectedCount >= s.pendingHorseCount {
			s.horseConfirmBtn.X = rightX + 110
			s.horseConfirmBtn.Y = btnY
			s.horseConfirmBtn.W = 100
			s.horseConfirmBtn.H = 35
			s.horseConfirmBtn.Draw(screen)
		}
		return
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
		// Normal production (stockpile already placed)
		instruction = "Resources are being produced automatically"

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
			s.drawDevelopmentControls(screen, rightX, barY)
			// Don't show generic instructions - controls are drawn instead
			return
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

	// Propose Trade button - to the right of the text
	s.proposeTradeBtn.X = startX + 280
	s.proposeTradeBtn.Y = barY + 35
	s.proposeTradeBtn.Draw(screen)

	// End Turn button
	s.endPhaseBtn.X = endX - 170
	s.endPhaseBtn.Y = barY + 30
	s.endPhaseBtn.Draw(screen)
}

// drawDevelopmentControls draws the development phase controls in the status bar.
func (s *GameplayScene) drawDevelopmentControls(screen *ebiten.Image, startX, barY int) {
	barW := ScreenWidth - 20
	barX := 10

	// Get player resources
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

	// Calculate affordability based on gold toggle
	var canAffordCity, canAffordWeapon, canAffordBoat bool

	if s.buildUseGold {
		canAffordCity = gold >= 4
		canAffordWeapon = gold >= 2
		canAffordBoat = gold >= 3
	} else {
		canAffordCity = coal >= 1 && gold >= 1 && iron >= 1 && timber >= 1
		canAffordWeapon = coal >= 1 && iron >= 1
		canAffordBoat = timber >= 3
	}

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
	DrawLargeText(screen, "YOUR TURN - BUILD", startX+20, barY+12, ColorSuccess)

	// Show current selection status
	statusText := "Select what to build, then click a territory"
	if s.selectedBuildType != "" {
		statusText = "Click one of your territories to build " + s.selectedBuildType
	}
	DrawText(screen, statusText, startX, barY+40, ColorTextMuted)

	// Build option buttons
	btnW := 80
	btnH := 30
	btnY := barY + 58
	btnX := startX

	// City button
	s.devCityBtn.X = btnX
	s.devCityBtn.Y = btnY
	s.devCityBtn.W = btnW
	s.devCityBtn.H = btnH
	s.devCityBtn.Primary = s.selectedBuildType == "city"
	s.devCityBtn.Disabled = !canAffordCity
	s.devCityBtn.Tooltip = "" // Costs shown below instead
	s.devCityBtn.Draw(screen)
	cityBtnX := btnX
	btnX += btnW + 10

	// Weapon button
	s.devWeaponBtn.X = btnX
	s.devWeaponBtn.Y = btnY
	s.devWeaponBtn.W = btnW
	s.devWeaponBtn.H = btnH
	s.devWeaponBtn.Primary = s.selectedBuildType == "weapon"
	s.devWeaponBtn.Disabled = !canAffordWeapon
	s.devWeaponBtn.Tooltip = "" // Costs shown below instead
	s.devWeaponBtn.Draw(screen)
	weaponBtnX := btnX
	btnX += btnW + 10

	// Boat button
	s.devBoatBtn.X = btnX
	s.devBoatBtn.Y = btnY
	s.devBoatBtn.W = btnW
	s.devBoatBtn.H = btnH
	s.devBoatBtn.Primary = s.selectedBuildType == "boat"
	s.devBoatBtn.Disabled = !canAffordBoat
	s.devBoatBtn.Tooltip = "" // Costs shown below instead
	s.devBoatBtn.Draw(screen)
	boatBtnX := btnX
	btnX += btnW + 20

	// Cost labels below buttons - show both normal and gold costs
	// Highlight active cost based on Use Gold toggle
	costY := btnY + btnH + 3
	normalColor := ColorTextMuted
	goldColor := ColorTextDim
	if s.buildUseGold {
		normalColor = ColorTextDim
		goldColor = ColorTextMuted
	}

	// City: 1 of each resource OR 4 gold
	DrawText(screen, "1 each", cityBtnX+14, costY, normalColor)
	DrawText(screen, "/4G", cityBtnX+14+36, costY, goldColor)

	// Weapon: 1 Coal + 1 Iron OR 2 gold
	DrawText(screen, "1C+1I", weaponBtnX+15, costY, normalColor)
	DrawText(screen, "/2G", weaponBtnX+15+30, costY, goldColor)

	// Boat: 3 Timber OR 3 gold
	DrawText(screen, "3T", boatBtnX+24, costY, normalColor)
	DrawText(screen, "/3G", boatBtnX+24+12, costY, goldColor)

	// Use Gold toggle
	if s.buildUseGold {
		s.devUseGoldBtn.Text = "[X] Use Gold"
	} else {
		s.devUseGoldBtn.Text = "[ ] Use Gold"
	}
	s.devUseGoldBtn.X = btnX
	s.devUseGoldBtn.Y = btnY
	s.devUseGoldBtn.W = 100
	s.devUseGoldBtn.H = btnH
	s.devUseGoldBtn.Primary = s.buildUseGold
	s.devUseGoldBtn.Draw(screen)

	// End Turn button (right side)
	s.endPhaseBtn.X = barX + barW - 170
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
		isOwnTerritory := owner == s.game.config.PlayerID

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
		strengthPreviewHeight := 65 // Always show strength info
		boxH := baseHeight + contentHeight + strengthPreviewHeight
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

		// Combat strength preview - always shown
		attackStr, defenseStr := s.calculateCombatStrength(tid)

		// Separator line
		vector.StrokeLine(screen, float32(boxX+10), float32(contentY+2), float32(boxX+boxW-10), float32(contentY+2), 1, ColorBorder, false)

		if isOwnTerritory {
			// For own territories, show defense strength
			DrawText(screen, "⚔ COMBAT STRENGTH", boxX+10, contentY+10, ColorWarning)

			defenseText := fmt.Sprintf("Defense: %d", defenseStr)
			DrawText(screen, defenseText, boxX+10, contentY+27, ColorText)
		} else {
			// For enemy/unclaimed territories, show attack vs defense
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

// isClickInBounds checks if a click is within the given bounds [x, y, w, h].
func (s *GameplayScene) isClickInBounds(mx, my int, bounds [4]int) bool {
	return mx >= bounds[0] && mx < bounds[0]+bounds[2] &&
		my >= bounds[1] && my < bounds[1]+bounds[3]
}

// openColorPicker opens the color picker dialog.
func (s *GameplayScene) openColorPicker() {
	// Build the list of used colors (by other players)
	s.usedColors = make(map[string]bool)
	myColor := ""

	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		myColor = player["color"].(string)
	}

	for playerID, playerData := range s.players {
		if playerID == s.game.config.PlayerID {
			continue // Skip self
		}
		player := playerData.(map[string]interface{})
		if colorVal, ok := player["color"].(string); ok {
			s.usedColors[colorVal] = true
		}
	}

	// Create buttons for each color
	s.colorPickerBtns = make([]*Button, len(PlayerColorOrder))
	for i, colorName := range PlayerColorOrder {
		cn := colorName // Capture for closure
		isUsed := s.usedColors[colorName]
		isMine := colorName == myColor

		s.colorPickerBtns[i] = &Button{
			Text:     colorName,
			Disabled: isUsed,
			Primary:  isMine,
			OnClick: func() {
				if !s.usedColors[cn] {
					s.selectColor(cn)
				}
			},
		}
	}

	s.showColorPicker = true
}

// selectColor sends the color change request to the server.
func (s *GameplayScene) selectColor(colorName string) {
	s.game.ChangeColor(colorName)
	s.showColorPicker = false
}

// drawColorPicker draws the color picker dialog.
func (s *GameplayScene) drawColorPicker(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 180}, false)

	// Dialog panel
	dialogW := 400
	dialogH := 380
	dialogX := (ScreenWidth - dialogW) / 2
	dialogY := (ScreenHeight - dialogH) / 2

	DrawFancyPanel(screen, dialogX, dialogY, dialogW, dialogH, "Choose Your Color")

	// Description
	DrawText(screen, "Select a color. Colors in use are disabled.", dialogX+20, dialogY+50, ColorTextMuted)

	// Color grid (4 columns x 3 rows)
	btnW := 80
	btnH := 55
	startX := dialogX + 25
	startY := dialogY + 80
	spacing := 10
	colsPerRow := 4

	for i, btn := range s.colorPickerBtns {
		col := i % colsPerRow
		row := i / colsPerRow

		btn.X = startX + col*(btnW+spacing)
		btn.Y = startY + row*(btnH+spacing)
		btn.W = btnW
		btn.H = btnH

		// Draw colored background for the button
		colorName := PlayerColorOrder[i]
		if pc, ok := PlayerColors[colorName]; ok {
			bgX := float32(btn.X)
			bgY := float32(btn.Y)
			bgW := float32(btn.W)
			bgH := float32(btn.H)

			// Draw color swatch
			if btn.Disabled {
				// Dimmed color for disabled (used by others)
				dimColor := color.RGBA{pc.R / 2, pc.G / 2, pc.B / 2, 200}
				vector.DrawFilledRect(screen, bgX, bgY, bgW, bgH, dimColor, false)
				// Draw X over it
				vector.StrokeLine(screen, bgX+5, bgY+5, bgX+bgW-5, bgY+bgH-5, 2, color.RGBA{150, 150, 150, 255}, false)
				vector.StrokeLine(screen, bgX+bgW-5, bgY+5, bgX+5, bgY+bgH-5, 2, color.RGBA{150, 150, 150, 255}, false)
			} else {
				vector.DrawFilledRect(screen, bgX, bgY, bgW, bgH, pc, false)
			}

			// Border - brighter for current/hovered
			borderColor := ColorBorder
			if btn.Primary {
				borderColor = ColorSuccess
				vector.StrokeRect(screen, bgX, bgY, bgW, bgH, 3, borderColor, false)
				// Draw checkmark
				DrawText(screen, "CURRENT", btn.X+10, btn.Y+btn.H-18, ColorText)
			} else {
				vector.StrokeRect(screen, bgX, bgY, bgW, bgH, 2, borderColor, false)
			}

			// Color name
			if !btn.Disabled {
				DrawText(screen, colorName, btn.X+8, btn.Y+8, ColorText)
			}
		}

		// Handle clicks (button logic is simpler here - just detect clicks)
		if !btn.Disabled {
			btn.Update()
		}
	}

	// Cancel button
	s.cancelColorBtn.X = dialogX + dialogW/2 - 50
	s.cancelColorBtn.Y = dialogY + dialogH - 50
	s.cancelColorBtn.W = 100
	s.cancelColorBtn.H = 35
	s.cancelColorBtn.Draw(screen)
}

// drawTurnToast draws the "YOUR TURN!" notification banner.
func (s *GameplayScene) drawTurnToast(screen *ebiten.Image) {
	if !s.showTurnToast {
		return
	}

	// Toast dimensions
	toastW := 400
	toastH := 60
	toastX := ScreenWidth/2 - toastW/2

	// Calculate Y position based on animation phase
	// Starts at -60 (off-screen), slides to 20, then slides back up
	var toastY int
	switch s.turnToastPhase {
	case "slide-in":
		// Animate from -60 to 20
		progress := float64(s.turnToastTimer) / float64(ToastSlideInFrames)
		toastY = int(-60 + progress*80)
	case "hold":
		toastY = 20
	case "slide-out":
		// Animate from 20 to -60
		progress := float64(s.turnToastTimer) / float64(ToastSlideOutFrames)
		toastY = int(20 - progress*80)
	default:
		toastY = 20
	}

	// Get player's color for accent
	var accentColor color.RGBA
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if playerColor, ok := player["color"].(string); ok {
			if pc, ok := PlayerColors[playerColor]; ok {
				accentColor = pc
			}
		}
	}
	if accentColor.A == 0 {
		accentColor = ColorPrimary // Fallback to cyan
	}

	// Draw toast background
	vector.DrawFilledRect(screen, float32(toastX), float32(toastY), float32(toastW), float32(toastH),
		color.RGBA{30, 30, 70, 240}, false)

	// Draw accent border (using player color)
	vector.StrokeRect(screen, float32(toastX), float32(toastY), float32(toastW), float32(toastH),
		3, accentColor, false)

	// Draw inner glow line at top
	vector.DrawFilledRect(screen, float32(toastX+2), float32(toastY+2), float32(toastW-4), 3,
		accentColor, false)

	// Draw "YOUR TURN!" text - centered, large
	titleText := "YOUR TURN!"
	titleX := toastX + toastW/2 - len(titleText)*6 // Approximate centering for large text
	DrawLargeText(screen, titleText, titleX, toastY+18, ColorText)

	// Draw phase name below - smaller, muted
	phaseText := s.currentPhase
	phaseX := toastX + toastW/2 - len(phaseText)*3 // Approximate centering for normal text
	DrawText(screen, phaseText, phaseX, toastY+42, ColorTextMuted)
}
