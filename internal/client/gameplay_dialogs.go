package client

import (
	"fmt"
	"image/color"
	"log"

	"lords-of-conquest/internal/protocol"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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

// drawWaterBodySelect draws the water body selection UI for boat placement
func (s *GameplayScene) drawWaterBodySelect(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 180}, false)

	// Panel
	panelW := 280
	panelH := 120 + len(s.waterBodyOptions)*50
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Place Boat")

	DrawTextCentered(screen, "Select water to place boat:", ScreenWidth/2, panelY+45, ColorText)
	DrawTextCentered(screen, "Click a water cell adjacent to your territory", ScreenWidth/2, panelY+65, ColorTextMuted)

	// Highlight water cells that can be clicked
	if s.mapData != nil {
		s.highlightSelectableWaterCells(screen)
	}
}

// highlightSelectableWaterCells highlights water cells that can be selected for boat placement
func (s *GameplayScene) highlightSelectableWaterCells(screen *ebiten.Image) {
	if s.buildMenuTerritory == "" || len(s.waterBodyOptions) == 0 {
		return
	}

	grid := s.mapData["grid"].([]interface{})
	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	waterBodies, hasWaterBodies := s.mapData["waterBodies"].(map[string]interface{})
	if !hasWaterBodies {
		return
	}

	// Extract numeric territory ID
	var numTerritoryID int
	if len(s.buildMenuTerritory) > 1 && s.buildMenuTerritory[0] == 't' {
		fmt.Sscanf(s.buildMenuTerritory[1:], "%d", &numTerritoryID)
	}

	// For each water body option, find and highlight adjacent cells
	for _, waterBodyID := range s.waterBodyOptions {
		wbData, ok := waterBodies[waterBodyID].(map[string]interface{})
		if !ok {
			continue
		}

		wbCells, ok := wbData["cells"].([]interface{})
		if !ok {
			continue
		}

		// Find water cells adjacent to the territory
		for _, cellData := range wbCells {
			cell, ok := cellData.([]interface{})
			if !ok || len(cell) < 2 {
				continue
			}
			wx := int(cell[0].(float64))
			wy := int(cell[1].(float64))

			// Check if adjacent to territory
			dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
			isAdjacent := false
			for _, d := range dirs {
				nx, ny := wx+d[0], wy+d[1]
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					row := grid[ny].([]interface{})
					if int(row[nx].(float64)) == numTerritoryID {
						isAdjacent = true
						break
					}
				}
			}

			if isAdjacent {
				sx, sy := s.gridToScreen(wx, wy)
				// Draw highlight
				highlightColor := color.RGBA{100, 200, 255, 150}
				vector.DrawFilledRect(screen, float32(sx)+2, float32(sy)+2,
					float32(s.cellSize)-4, float32(s.cellSize)-4, highlightColor, false)
				vector.StrokeRect(screen, float32(sx)+1, float32(sy)+1,
					float32(s.cellSize)-2, float32(s.cellSize)-2, 2, color.RGBA{100, 200, 255, 255}, false)
			}
		}
	}
}

// drawAttackPlan draws the attack planning dialog
func (s *GameplayScene) drawAttackPlan(screen *ebiten.Image) {
	if s.attackPreview == nil {
		return
	}

	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 180}, false)

	// Get target name
	targetName := s.attackPlanTarget
	if terr, ok := s.territories[s.attackPlanTarget].(map[string]interface{}); ok {
		if name, ok := terr["name"].(string); ok {
			targetName = name
		}
	}

	// Panel dimensions based on reinforcement count
	reinforceCount := len(s.attackPreview.Reinforcements)
	panelW := 450 // Wider panel for better button layout
	panelH := 160 // Base height for no reinforcements
	if reinforceCount > 0 {
		panelH = 200 + reinforceCount*60
		// Add extra space for checkboxes when a unit is selected
		if s.selectedReinforcement != nil {
			checkboxCount := 0
			if s.selectedReinforcement.UnitType == "boat" {
				if s.selectedReinforcement.CanCarryHorse {
					checkboxCount++
				}
				if s.selectedReinforcement.CanCarryWeapon {
					checkboxCount++
				}
			} else if s.selectedReinforcement.UnitType == "horse" && s.selectedReinforcement.CanCarryWeapon {
				checkboxCount++
			}
			panelH += checkboxCount * 25
		}
	}
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Plan Attack")

	// Target info
	DrawTextCentered(screen, "Attack: "+targetName, ScreenWidth/2, panelY+45, ColorText)

	// Strength preview with ally info
	attackStr := fmt.Sprintf("%d", s.attackPreview.AttackStrength)
	if s.attackPreview.AttackerAllyStrength > 0 {
		attackStr = fmt.Sprintf("%d (+%d allies)", s.attackPreview.AttackStrength, s.attackPreview.AttackerAllyStrength)
	}
	defenseStr := fmt.Sprintf("%d", s.attackPreview.DefenseStrength)
	if s.attackPreview.DefenderAllyStrength > 0 {
		defenseStr = fmt.Sprintf("%d (+%d allies)", s.attackPreview.DefenseStrength, s.attackPreview.DefenderAllyStrength)
	}
	strengthText := fmt.Sprintf("Attack: %s vs Defense: %s", attackStr, defenseStr)
	DrawTextCentered(screen, strengthText, ScreenWidth/2, panelY+70, ColorTextMuted)

	yPos := panelY + 100

	// Reinforcement options
	if reinforceCount > 0 {
		DrawText(screen, "Available Reinforcements (click to select):", panelX+15, yPos, ColorText)
		yPos += 25

		for i, reinf := range s.attackPreview.Reinforcements {
			// Territory name
			fromName := reinf.FromTerritory
			if terr, ok := s.territories[reinf.FromTerritory].(map[string]interface{}); ok {
				if name, ok := terr["name"].(string); ok {
					fromName = name
				}
			}

			// Check if selected
			isSelected := s.selectedReinforcement != nil &&
				s.selectedReinforcement.FromTerritory == reinf.FromTerritory &&
				s.selectedReinforcement.UnitType == reinf.UnitType

			// Draw option box
			optY := yPos + i*60
			boxColor := color.RGBA{50, 50, 70, 255}
			if isSelected {
				boxColor = color.RGBA{80, 100, 150, 255}
			}
			vector.DrawFilledRect(screen, float32(panelX+15), float32(optY), float32(panelW-30), 55, boxColor, false)
			if isSelected {
				vector.StrokeRect(screen, float32(panelX+15), float32(optY), float32(panelW-30), 55, 2, ColorBorder, false)
			}

			// Unit type and location
			unitLabel := fmt.Sprintf("%s from %s (+%d)", reinf.UnitType, fromName, reinf.StrengthBonus)
			DrawText(screen, unitLabel, panelX+25, optY+15, ColorText)

			// Carry options
			carryText := ""
			if reinf.UnitType == "boat" {
				if reinf.CanCarryHorse {
					carryText += "Can load Horse "
				}
				if reinf.CanCarryWeapon {
					carryText += "Can load Weapon"
				}
			} else if reinf.UnitType == "horse" && reinf.CanCarryWeapon {
				carryText = "Can carry Weapon"
			}
			if carryText != "" {
				DrawText(screen, carryText, panelX+25, optY+35, ColorTextMuted)
			}

			// Handle click on this option
			mx, my := ebiten.CursorPosition()
			if mx >= panelX+15 && mx <= panelX+panelW-15 &&
				my >= optY && my <= optY+55 {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					s.selectedReinforcement = &ReinforcementData{
						UnitType:       reinf.UnitType,
						FromTerritory:  reinf.FromTerritory,
						WaterBodyID:    reinf.WaterBodyID,
						StrengthBonus:  reinf.StrengthBonus,
						CanCarryWeapon: reinf.CanCarryWeapon,
						CanCarryHorse:  reinf.CanCarryHorse,
					}
					// Reset checkboxes when changing selection
					s.loadHorseCheckbox = false
					s.loadWeaponCheckbox = false
				}
			}
		}
		yPos += reinforceCount * 60

		// Cargo checkboxes if boat or horse selected
		if s.selectedReinforcement != nil {
			if s.selectedReinforcement.UnitType == "boat" {
				if s.selectedReinforcement.CanCarryHorse {
					s.drawCheckbox(screen, panelX+20, yPos+10, "Load Horse onto Boat", &s.loadHorseCheckbox)
					yPos += 25
				}
				if s.selectedReinforcement.CanCarryWeapon {
					s.drawCheckbox(screen, panelX+20, yPos+10, "Load Weapon onto Boat", &s.loadWeaponCheckbox)
					yPos += 25
				}
			} else if s.selectedReinforcement.UnitType == "horse" && s.selectedReinforcement.CanCarryWeapon {
				s.drawCheckbox(screen, panelX+20, yPos+10, "Carry Weapon on Horse", &s.loadWeaponCheckbox)
				yPos += 25
			}
		}
	}

	// Buttons at bottom of panel
	btnY := panelY + panelH - 55

	// Attack button (text depends on whether reinforcements are available)
	if reinforceCount == 0 {
		s.attackNoReinfBtn.Text = "Attack"
	} else {
		s.attackNoReinfBtn.Text = "Attack Without"
	}
	s.attackNoReinfBtn.X = panelX + 20
	s.attackNoReinfBtn.Y = btnY
	s.attackNoReinfBtn.Draw(screen)

	// Cancel button (always on the right)
	s.cancelAttackBtn.X = panelX + panelW - 120
	s.cancelAttackBtn.Y = btnY
	s.cancelAttackBtn.Draw(screen)

	// Attack with selected reinforcement (only if one is selected, positioned in middle)
	if s.selectedReinforcement != nil {
		s.attackWithReinfBtn.X = panelX + (panelW-160)/2 // Center the button
		s.attackWithReinfBtn.Y = btnY
		s.attackWithReinfBtn.Text = "With " + s.selectedReinforcement.UnitType
		s.attackWithReinfBtn.Draw(screen)
	}
}

// drawCheckbox draws a simple checkbox with label
func (s *GameplayScene) drawCheckbox(screen *ebiten.Image, x, y int, label string, checked *bool) {
	boxSize := 16

	// Draw box
	boxColor := color.RGBA{60, 60, 80, 255}
	if *checked {
		boxColor = color.RGBA{100, 150, 200, 255}
	}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(boxSize), float32(boxSize), boxColor, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(boxSize), float32(boxSize), 1, ColorText, false)

	// Draw check mark if checked
	if *checked {
		// Simple X mark
		vector.StrokeLine(screen, float32(x+3), float32(y+3), float32(x+boxSize-3), float32(y+boxSize-3), 2, ColorText, false)
		vector.StrokeLine(screen, float32(x+boxSize-3), float32(y+3), float32(x+3), float32(y+boxSize-3), 2, ColorText, false)
	}

	// Draw label
	DrawText(screen, label, x+boxSize+8, y+2, ColorText)

	// Handle click
	mx, my := ebiten.CursorPosition()
	if mx >= x && mx <= x+boxSize+150 && my >= y && my <= y+boxSize {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			*checked = !*checked
		}
	}
}

// ShowCombatResult displays the combat result popup
func (s *GameplayScene) ShowCombatResult(result *CombatResultData) {
	s.combatResult = result
	s.showCombatResult = true
}

// ShowAttackPlan displays the attack planning dialog
func (s *GameplayScene) ShowAttackPlan(preview *AttackPreviewData) {
	s.attackPreview = preview
	s.attackPlanTarget = preview.TargetTerritory
	s.selectedReinforcement = nil
	s.loadHorseCheckbox = false
	s.loadWeaponCheckbox = false
	s.showAttackPlan = true
}

// doAttack executes the attack with or without reinforcement
func (s *GameplayScene) doAttack(withReinforcement bool) {
	if s.attackPlanTarget == "" {
		return
	}

	var reinforcement *ReinforcementInfo
	if withReinforcement && s.selectedReinforcement != nil {
		reinforcement = &ReinforcementInfo{
			UnitType:      s.selectedReinforcement.UnitType,
			FromTerritory: s.selectedReinforcement.FromTerritory,
			WaterBodyID:   s.selectedReinforcement.WaterBodyID,
		}
		// For boats: add cargo
		if s.selectedReinforcement.UnitType == "boat" {
			if s.loadWeaponCheckbox && s.selectedReinforcement.CanCarryWeapon {
				reinforcement.CarryWeapon = true
				reinforcement.WeaponFrom = s.selectedReinforcement.FromTerritory
			}
			if s.loadHorseCheckbox && s.selectedReinforcement.CanCarryHorse {
				reinforcement.CarryHorse = true
				reinforcement.HorseFrom = s.selectedReinforcement.FromTerritory
			}
		}
		// For horses: add weapon cargo
		if s.selectedReinforcement.UnitType == "horse" {
			if s.loadWeaponCheckbox && s.selectedReinforcement.CanCarryWeapon {
				reinforcement.CarryWeapon = true
				reinforcement.WeaponFrom = s.selectedReinforcement.FromTerritory
			}
		}
	}

	log.Printf("Executing attack on %s with reinforcement: %v", s.attackPlanTarget, reinforcement)
	s.game.ExecuteAttackWithReinforcement(s.attackPlanTarget, reinforcement)
	s.cancelAttackPlan()
}

// cancelAttackPlan cancels the attack planning
func (s *GameplayScene) cancelAttackPlan() {
	s.showAttackPlan = false
	s.attackPlanTarget = ""
	s.attackPreview = nil
	s.selectedReinforcement = nil
	s.loadHorseCheckbox = false
	s.loadWeaponCheckbox = false
}

// drawAllyMenu draws the alliance selection menu
func (s *GameplayScene) drawAllyMenu(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 180}, false)

	// Count other players for menu sizing
	otherPlayerCount := 0
	for _, playerIDInterface := range s.playerOrder {
		playerID := playerIDInterface.(string)
		if playerID != s.game.config.PlayerID {
			otherPlayerCount++
		}
	}

	// Menu panel - calculate proper height
	// Base: 60 (header + current) + 3*45 (ask/defender/neutral) + 25 (label) + players*45 + 55 (cancel + padding)
	menuW := 280
	menuH := 60 + 3*45 + 25 + otherPlayerCount*45 + 55
	menuX := ScreenWidth/2 - menuW/2
	menuY := ScreenHeight/2 - menuH/2

	DrawFancyPanel(screen, menuX, menuY, menuW, menuH, "Set Alliance")

	// Current setting display
	currentText := "Current: "
	switch s.myAllianceSetting {
	case "neutral":
		currentText += "Always Neutral"
	case "defender":
		currentText += "Always Defender"
	case "ask":
		currentText += "Ask Each Time"
	default:
		// It's a player ID - find the name
		if playerData, ok := s.players[s.myAllianceSetting]; ok {
			player := playerData.(map[string]interface{})
			currentText += "Ally with " + player["name"].(string)
		} else {
			currentText += s.myAllianceSetting
		}
	}
	DrawText(screen, currentText, menuX+20, menuY+35, ColorTextMuted)

	// Position buttons
	btnX := menuX + 40
	btnY := menuY + 60

	s.allyAskBtn.X = btnX
	s.allyAskBtn.Y = btnY
	s.allyAskBtn.Primary = s.myAllianceSetting == "ask"
	s.allyAskBtn.Draw(screen)
	btnY += 45

	s.allyDefenderBtn.X = btnX
	s.allyDefenderBtn.Y = btnY
	s.allyDefenderBtn.Primary = s.myAllianceSetting == "defender"
	s.allyDefenderBtn.Draw(screen)
	btnY += 45

	s.allyNeutralBtn.X = btnX
	s.allyNeutralBtn.Y = btnY
	s.allyNeutralBtn.Primary = s.myAllianceSetting == "neutral"
	s.allyNeutralBtn.Draw(screen)
	btnY += 45

	// Add buttons for each other player
	if otherPlayerCount > 0 {
		DrawText(screen, "Ally with player:", menuX+20, btnY+5, ColorTextMuted)
		btnY += 25

		// Rebuild player buttons list
		s.allyPlayerBtns = make([]*Button, 0, otherPlayerCount)
		s.allyPlayerIDs = make([]string, 0, otherPlayerCount)

		for _, playerIDInterface := range s.playerOrder {
			playerID := playerIDInterface.(string)
			if playerID == s.game.config.PlayerID {
				continue
			}
			if playerData, ok := s.players[playerID]; ok {
				player := playerData.(map[string]interface{})
				playerName := player["name"].(string)

				btn := &Button{
					X: btnX, Y: btnY, W: 200, H: 35,
					Text:    playerName,
					Primary: s.myAllianceSetting == playerID,
				}
				// Capture playerID for closure
				pid := playerID
				btn.OnClick = func() { s.setAlliance(pid) }

				// Update button to handle clicks (since we create it fresh each frame)
				btn.Update()
				btn.Draw(screen)

				s.allyPlayerBtns = append(s.allyPlayerBtns, btn)
				s.allyPlayerIDs = append(s.allyPlayerIDs, playerID)
				btnY += 45
			}
		}
	}

	// Cancel button after all player buttons
	btnY += 10 // Small gap before cancel
	s.cancelAllyMenuBtn.X = btnX
	s.cancelAllyMenuBtn.Y = btnY
	s.cancelAllyMenuBtn.Draw(screen)
}

// setAlliance sends the alliance setting to the server
func (s *GameplayScene) setAlliance(setting string) {
	log.Printf("Setting alliance to: %s", setting)
	s.game.SetAlliance(setting)
	s.myAllianceSetting = setting
	s.showAllyMenu = false
}

// drawAllyRequest draws the alliance request popup
func (s *GameplayScene) drawAllyRequest(screen *ebiten.Image) {
	if s.allyRequest == nil {
		return
	}

	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 200}, false)

	// Popup panel
	panelW := 400
	panelH := 220
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Alliance Request")

	// Battle info
	y := panelY + 40
	DrawText(screen, fmt.Sprintf("%s is attacking %s", s.allyRequest.AttackerName, s.allyRequest.DefenderName),
		panelX+20, y, ColorText)
	y += 25
	DrawText(screen, fmt.Sprintf("at %s", s.allyRequest.TerritoryName),
		panelX+20, y, ColorTextMuted)
	y += 30
	DrawText(screen, fmt.Sprintf("Your adjacent strength: %d", s.allyRequest.YourStrength),
		panelX+20, y, ColorText)
	y += 30

	// Countdown
	secondsLeft := s.allyRequestCountdown / 60 // Assuming 60fps
	DrawText(screen, fmt.Sprintf("Time remaining: %d seconds", secondsLeft),
		panelX+20, y, ColorTextMuted)

	// Buttons
	btnY := panelY + panelH - 60
	btnW := 120
	spacing := 10

	s.supportAttackerBtn.X = panelX + 20
	s.supportAttackerBtn.Y = btnY
	s.supportAttackerBtn.W = btnW
	s.supportAttackerBtn.Draw(screen)

	s.stayNeutralBtn.X = panelX + 20 + btnW + spacing
	s.stayNeutralBtn.Y = btnY
	s.stayNeutralBtn.W = btnW
	s.stayNeutralBtn.Draw(screen)

	s.supportDefenderBtn.X = panelX + 20 + 2*(btnW+spacing)
	s.supportDefenderBtn.Y = btnY
	s.supportDefenderBtn.W = btnW
	s.supportDefenderBtn.Draw(screen)
}

// ShowAllianceRequest displays the alliance request popup
func (s *GameplayScene) ShowAllianceRequest(payload *protocol.AllianceRequestPayload) {
	s.allyRequest = &AllianceRequestData{
		BattleID:      payload.BattleID,
		AttackerID:    payload.AttackerID,
		AttackerName:  payload.AttackerName,
		DefenderID:    payload.DefenderID,
		DefenderName:  payload.DefenderName,
		TerritoryID:   payload.TerritoryID,
		TerritoryName: payload.TerritoryName,
		YourStrength:  payload.YourStrength,
		TimeLimit:     payload.TimeLimit,
		ExpiresAt:     payload.ExpiresAt,
	}
	// Set countdown to 60 seconds (60fps * 60 seconds)
	s.allyRequestCountdown = 60 * 60
	s.showAllyRequest = true
}

// voteAlliance sends the alliance vote to the server
func (s *GameplayScene) voteAlliance(side string) {
	if s.allyRequest != nil {
		log.Printf("Voting %s for battle %s", side, s.allyRequest.BattleID)
		s.game.AllianceVote(s.allyRequest.BattleID, side)
	}
	s.showAllyRequest = false
	s.allyRequest = nil
}

// ShowPhaseSkipped queues a phase skip popup for display.
func (s *GameplayScene) ShowPhaseSkipped(phase, reason string) {
	// Add to queue
	s.phaseSkipQueue = append(s.phaseSkipQueue, PhaseSkipData{Phase: phase, Reason: reason})

	// If not currently showing a skip, start showing the first one
	if !s.showPhaseSkip {
		s.showNextPhaseSkip()
	}
}

// showNextPhaseSkip displays the next queued phase skip.
func (s *GameplayScene) showNextPhaseSkip() {
	if len(s.phaseSkipQueue) == 0 {
		s.showPhaseSkip = false
		return
	}

	// Pop from queue
	skip := s.phaseSkipQueue[0]
	s.phaseSkipQueue = s.phaseSkipQueue[1:]

	s.phaseSkipPhase = skip.Phase
	s.phaseSkipReason = skip.Reason
	s.phaseSkipCountdown = 30 * 60 // 30 seconds at 60fps
	s.showPhaseSkip = true
}

// drawPhaseSkip draws the phase skip popup.
func (s *GameplayScene) drawPhaseSkip(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 180}, false)

	// Popup panel
	panelW := 450
	panelH := 180
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, s.phaseSkipPhase+" Skipped!")

	// Reason text (may need to wrap)
	reason := s.phaseSkipReason
	y := panelY + 50

	// Word wrap the reason if too long
	maxCharsPerLine := 50
	if len(reason) > maxCharsPerLine {
		// Simple word wrap
		words := []string{}
		start := 0
		for i := 0; i < len(reason); i++ {
			if reason[i] == ' ' {
				words = append(words, reason[start:i])
				start = i + 1
			}
		}
		if start < len(reason) {
			words = append(words, reason[start:])
		}

		line := ""
		for _, word := range words {
			if len(line)+len(word)+1 > maxCharsPerLine {
				DrawText(screen, line, panelX+20, y, ColorText)
				y += 20
				line = word
			} else {
				if line != "" {
					line += " "
				}
				line += word
			}
		}
		if line != "" {
			DrawText(screen, line, panelX+20, y, ColorText)
			y += 20
		}
	} else {
		DrawText(screen, reason, panelX+20, y, ColorText)
		y += 20
	}

	// Countdown
	y += 10
	secondsLeft := s.phaseSkipCountdown / 60
	DrawText(screen, fmt.Sprintf("(continuing in %d seconds...)", secondsLeft),
		panelX+20, y, ColorTextMuted)

	// OK button
	s.dismissSkipBtn.X = panelX + panelW/2 - 50
	s.dismissSkipBtn.Y = panelY + panelH - 55
	s.dismissSkipBtn.Draw(screen)
}

// ShowVictory displays the victory screen.
func (s *GameplayScene) ShowVictory(winnerID, winnerName, reason string) {
	s.victoryWinnerID = winnerID
	s.victoryWinnerName = winnerName
	s.victoryReason = reason
	s.victoryTimer = 0
	s.showVictory = true

	// Start playing the victory music
	PlayWinnerMusic()
}

// drawVictoryScreen draws the victory celebration screen.
func (s *GameplayScene) drawVictoryScreen(screen *ebiten.Image) {
	// Full screen dark overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 230}, false)

	// Calculate center
	centerX := ScreenWidth / 2
	centerY := ScreenHeight / 2

	// Transition messages after 5 seconds (300 frames)
	showMusicMessage := s.victoryTimer > 300

	// Draw decorative border/frame
	frameW := 600
	frameH := 350
	frameX := centerX - frameW/2
	frameY := centerY - frameH/2

	// Fancy gold border
	borderColor := color.RGBA{218, 165, 32, 255} // Gold
	vector.StrokeRect(screen, float32(frameX-4), float32(frameY-4),
		float32(frameW+8), float32(frameH+8), 4, borderColor, false)
	vector.StrokeRect(screen, float32(frameX-8), float32(frameY-8),
		float32(frameW+16), float32(frameH+16), 2, borderColor, false)

	// Dark panel background
	DrawFancyPanel(screen, frameX, frameY, frameW, frameH, "")

	if showMusicMessage {
		// "A Musical Tribute to the Winner"
		DrawLargeTextCentered(screen, "A Musical Tribute", centerX, frameY+60, borderColor)
		DrawLargeTextCentered(screen, "to the Winner", centerX, frameY+100, borderColor)

		// Winner name
		DrawLargeTextCentered(screen, s.victoryWinnerName, centerX, frameY+170, ColorText)

		// Music credit
		DrawTextCentered(screen, "Prelude and Fugue No. 1 in C major, BWV 846", centerX, frameY+220, ColorTextMuted)
		DrawTextCentered(screen, "by Johann Sebastian Bach", centerX, frameY+240, ColorTextMuted)
	} else {
		// "A Lord Of Conquest Is Proclaimed!!"
		DrawLargeTextCentered(screen, "A Lord Of Conquest", centerX, frameY+60, borderColor)
		DrawLargeTextCentered(screen, "Is Proclaimed!!", centerX, frameY+100, borderColor)

		// Winner name in large text
		DrawLargeTextCentered(screen, s.victoryWinnerName, centerX, frameY+170, ColorSuccess)

		// Victory reason
		reasonText := "by conquest"
		if s.victoryReason == "cities" {
			reasonText = fmt.Sprintf("with %d cities", s.game.lobbyState.Settings.VictoryCities)
		} else if s.victoryReason == "elimination" {
			reasonText = "by eliminating all rivals"
		}
		DrawTextCentered(screen, reasonText, centerX, frameY+210, ColorTextMuted)
	}

	// Return to lobby button
	s.returnToLobbyBtn.X = centerX - 100
	s.returnToLobbyBtn.Y = frameY + frameH - 70
	s.returnToLobbyBtn.Draw(screen)

	// Check if this player is the winner
	if s.victoryWinnerID == s.game.config.PlayerID {
		DrawTextCentered(screen, "Congratulations! You are victorious!", centerX, frameY+frameH+20, ColorSuccess)
	}
}
