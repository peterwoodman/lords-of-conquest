package client

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"strings"

	"lords-of-conquest/internal/protocol"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawCombatResult draws the combat result popup
func (s *GameplayScene) drawCombatResult(screen *ebiten.Image) {
	if s.combatResult == nil {
		return
	}

	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 200}, false)

	// Result panel - increased height to prevent overlap
	panelW := 340
	panelH := 200
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	// Panel color based on result
	if s.combatResult.AttackerWins {
		DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "VICTORY!")
	} else {
		DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "DEFEAT")
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
	DrawLargeTextCentered(screen, resultText, ScreenWidth/2, panelY+60, resultColor)

	// Territory name
	DrawTextCentered(screen, s.combatResult.TargetName, ScreenWidth/2, panelY+90, ColorText)

	// Outcome description
	var outcomeText string
	if s.combatResult.AttackerWins {
		outcomeText = "Territory captured!"
	} else {
		outcomeText = "Your forces were repelled."
	}
	DrawTextCentered(screen, outcomeText, ScreenWidth/2, panelY+115, ColorTextMuted)

	// OK button - positioned with proper spacing
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

	// Button layout: space them evenly across the panel
	// [Attack Without] [With Unit] [Cancel]
	btnWidth := 130
	btnGap := 15
	totalBtnsWidth := btnWidth*3 + btnGap*2
	btnStartX := panelX + (panelW-totalBtnsWidth)/2

	// Attack button - only show if base attack strength > 0
	// (if strength is 0, player must bring reinforcements to attack)
	if s.attackPreview.AttackStrength > 0 {
		if reinforceCount == 0 {
			s.attackNoReinfBtn.Text = "Attack"
		} else {
			s.attackNoReinfBtn.Text = "Attack Without"
		}
		s.attackNoReinfBtn.W = btnWidth
		s.attackNoReinfBtn.X = btnStartX
		s.attackNoReinfBtn.Y = btnY
		s.attackNoReinfBtn.Draw(screen)
	} else if s.selectedReinforcement == nil {
		// Show message that reinforcement is required
		DrawText(screen, "Bring forces to attack", btnStartX, btnY+10, ColorWarning)
	}

	// Attack with selected reinforcement (only if one is selected)
	if s.selectedReinforcement != nil {
		s.attackWithReinfBtn.W = btnWidth
		s.attackWithReinfBtn.X = btnStartX + btnWidth + btnGap
		s.attackWithReinfBtn.Y = btnY
		s.attackWithReinfBtn.Text = "With " + s.selectedReinforcement.UnitType
		s.attackWithReinfBtn.Draw(screen)
	}

	// Cancel button (always on the right)
	s.cancelAttackBtn.W = btnWidth
	s.cancelAttackBtn.X = btnStartX + (btnWidth+btnGap)*2
	s.cancelAttackBtn.Y = btnY
	s.cancelAttackBtn.Draw(screen)
}

// updateAttackPlanInput handles input for the attack planning dialog (called from Update)
func (s *GameplayScene) updateAttackPlanInput() {
	if s.attackPreview == nil {
		return
	}

	// Calculate panel dimensions (must match drawAttackPlan)
	reinforceCount := len(s.attackPreview.Reinforcements)
	panelW := 450
	panelH := 160
	if reinforceCount > 0 {
		panelH = 200 + reinforceCount*60
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
	yPos := panelY + 100

	// Handle reinforcement selection clicks
	if reinforceCount > 0 && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		yPos += 25 // After header text

		for i, reinf := range s.attackPreview.Reinforcements {
			optY := yPos + i*60
			if mx >= panelX+15 && mx <= panelX+panelW-15 &&
				my >= optY && my <= optY+55 {
				s.selectedReinforcement = &ReinforcementData{
					UnitType:       reinf.UnitType,
					FromTerritory:  reinf.FromTerritory,
					WaterBodyID:    reinf.WaterBodyID,
					StrengthBonus:  reinf.StrengthBonus,
					CanCarryWeapon: reinf.CanCarryWeapon,
					CanCarryHorse:  reinf.CanCarryHorse,
				}
				s.loadHorseCheckbox = false
				s.loadWeaponCheckbox = false
				break
			}
		}

		// Handle checkbox clicks
		if s.selectedReinforcement != nil {
			checkboxY := yPos + reinforceCount*60
			checkboxX := panelX + 20
			boxSize := 16

			if s.selectedReinforcement.UnitType == "boat" {
				if s.selectedReinforcement.CanCarryHorse {
					if mx >= checkboxX && mx <= checkboxX+boxSize+150 &&
						my >= checkboxY+10 && my <= checkboxY+10+boxSize {
						s.loadHorseCheckbox = !s.loadHorseCheckbox
					}
					checkboxY += 25
				}
				if s.selectedReinforcement.CanCarryWeapon {
					if mx >= checkboxX && mx <= checkboxX+boxSize+150 &&
						my >= checkboxY+10 && my <= checkboxY+10+boxSize {
						s.loadWeaponCheckbox = !s.loadWeaponCheckbox
					}
				}
			} else if s.selectedReinforcement.UnitType == "horse" && s.selectedReinforcement.CanCarryWeapon {
				if mx >= checkboxX && mx <= checkboxX+boxSize+150 &&
					my >= checkboxY+10 && my <= checkboxY+10+boxSize {
					s.loadWeaponCheckbox = !s.loadWeaponCheckbox
				}
			}
		}
	}
}

// drawCheckbox draws a simple checkbox with label (no click handling - done in updateAttackPlanInput)
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
}

// ShowCombatResult starts the combat animation before displaying the result
func (s *GameplayScene) ShowCombatResult(result *CombatResultData) {
	// If animation or result dialog is already showing, queue this result
	if s.showCombatAnimation || s.showCombatResult {
		s.combatResultQueue = append(s.combatResultQueue, result)
		return
	}

	// Start the combat animation
	s.startCombatAnimation(result)
}

// startCombatAnimation begins the animation for a combat result
func (s *GameplayScene) startCombatAnimation(result *CombatResultData) {
	s.combatPendingResult = result
	s.combatAnimTerritory = result.TargetTerritory
	s.combatAnimExplosions = make([]CombatExplosion, 0)

	// Calculate animation duration based on combat strength
	// Base: 60 frames (1 second), plus more for higher strength battles
	totalStrength := result.AttackStrength + result.DefenseStrength
	s.combatAnimMaxDuration = 60 + (totalStrength * 10) // More intense = longer animation
	if s.combatAnimMaxDuration > 300 {                  // Cap at 5 seconds
		s.combatAnimMaxDuration = 300
	}
	s.combatAnimTimer = s.combatAnimMaxDuration
	s.showCombatAnimation = true
}

// dismissCombatResult dismisses the current combat result and shows the next queued one
func (s *GameplayScene) dismissCombatResult() {
	// Send acknowledgment for this combat result
	if s.combatResult != nil && s.combatResult.EventID != "" {
		s.game.SendClientReady(s.combatResult.EventID, protocol.EventCombat)
	}

	s.showCombatResult = false
	s.combatResult = nil

	// Check if there are more combat results queued
	if len(s.combatResultQueue) > 0 {
		// Pop the first result from the queue
		nextResult := s.combatResultQueue[0]
		s.combatResultQueue = s.combatResultQueue[1:]
		// Start animation for the next result
		s.startCombatAnimation(nextResult)
	} else {
		// No more combat results - apply any pending game state
		if s.combatPendingState != nil {
			s.applyGameState(s.combatPendingState)
			s.combatPendingState = nil
		}
	}
}

// updateCombatAnimation updates the combat animation state each frame
func (s *GameplayScene) updateCombatAnimation() {
	s.combatAnimTimer--

	// Update existing explosions
	for i := len(s.combatAnimExplosions) - 1; i >= 0; i-- {
		s.combatAnimExplosions[i].Frame++
		// Remove finished explosions
		if s.combatAnimExplosions[i].Frame >= s.combatAnimExplosions[i].MaxFrames {
			s.combatAnimExplosions = append(s.combatAnimExplosions[:i], s.combatAnimExplosions[i+1:]...)
		}
	}

	// Spawn new explosions based on combat intensity
	// Higher strength = more explosions
	totalStrength := s.combatPendingResult.AttackStrength + s.combatPendingResult.DefenseStrength
	spawnRate := 8 - totalStrength // Faster spawning for stronger battles
	if spawnRate < 2 {
		spawnRate = 2
	}

	// Get territory cells
	if s.combatAnimTimer > 0 && s.combatAnimTimer%spawnRate == 0 {
		cells := s.getCombatTerritoryCells()
		if len(cells) > 0 {
			// Pick random cell
			cell := cells[rand.Intn(len(cells))]
			explosion := CombatExplosion{
				X:         cell[0],
				Y:         cell[1],
				OffsetX:   rand.Float32()*float32(s.cellSize-8) + 4,
				OffsetY:   rand.Float32()*float32(s.cellSize-8) + 4,
				Frame:     0,
				MaxFrames: 15 + rand.Intn(10), // 15-25 frames per explosion
			}
			s.combatAnimExplosions = append(s.combatAnimExplosions, explosion)
		}
	}

	// End animation
	if s.combatAnimTimer <= 0 {
		s.showCombatAnimation = false
		s.combatAnimExplosions = nil

		// Apply any queued game state update now that animation is done
		if s.combatPendingState != nil {
			s.applyGameState(s.combatPendingState)
			s.combatPendingState = nil
		}

		// Check if we captured a stockpile and it's our attack - animate the resource transfer
		isMyAttack := s.combatPendingResult.AttackerID == s.game.config.PlayerID
		log.Printf("Combat animation ended: AttackerID=%s, MyPlayerID=%s, isMyAttack=%v",
			s.combatPendingResult.AttackerID, s.game.config.PlayerID, isMyAttack)

		if isMyAttack && s.combatPendingResult.StockpileCaptured {
			// Start stockpile capture animation
			s.startStockpileCaptureAnimation(s.combatPendingResult)
			return
		}

		if isMyAttack {
			// Show dialog - acknowledgment will be sent when dialog is dismissed
			s.combatResult = s.combatPendingResult
			s.showCombatResult = true
		} else {
			// For other players' attacks, send acknowledgment immediately
			s.game.SendClientReady(s.combatPendingResult.EventID, protocol.EventCombat)

			// Apply pending state since we're not showing a dialog
			if s.combatPendingState != nil {
				s.applyGameState(s.combatPendingState)
				s.combatPendingState = nil
			}

			if len(s.combatResultQueue) > 0 {
				nextResult := s.combatResultQueue[0]
				s.combatResultQueue = s.combatResultQueue[1:]
				s.startCombatAnimation(nextResult)
			}
		}
	}
}

// getCombatTerritoryCells returns all cells of the combat target territory
func (s *GameplayScene) getCombatTerritoryCells() [][2]int {
	if s.mapData == nil || s.combatAnimTerritory == "" {
		return nil
	}

	grid := s.mapData["grid"].([]interface{})
	return s.findTerritoryCells(s.combatAnimTerritory, grid)
}

// drawCombatAnimation draws the combat animation effects
func (s *GameplayScene) drawCombatAnimation(screen *ebiten.Image) {
	// Draw explosions
	for _, exp := range s.combatAnimExplosions {
		s.drawExplosion(screen, exp)
	}
}

// drawExplosion draws a single explosion effect
func (s *GameplayScene) drawExplosion(screen *ebiten.Image, exp CombatExplosion) {
	sx, sy := s.gridToScreen(exp.X, exp.Y)
	centerX := float32(sx) + exp.OffsetX
	centerY := float32(sy) + exp.OffsetY

	// Animation progress (0 to 1)
	progress := float32(exp.Frame) / float32(exp.MaxFrames)

	// Explosion phases:
	// 0-0.3: Expanding bright flash
	// 0.3-0.7: Debris/sparks
	// 0.7-1.0: Fade out

	if progress < 0.3 {
		// Expanding flash
		expandProgress := progress / 0.3
		radius := 3 + expandProgress*6
		alpha := uint8(255 - expandProgress*100)

		// Bright center
		vector.DrawFilledCircle(screen, centerX, centerY, radius,
			color.RGBA{255, 255, 200, alpha}, false)
		// Orange glow
		vector.DrawFilledCircle(screen, centerX, centerY, radius*0.7,
			color.RGBA{255, 150, 50, alpha}, false)
	} else if progress < 0.7 {
		// Debris phase - draw scattered pixels
		debrisProgress := (progress - 0.3) / 0.4
		alpha := uint8(255 - debrisProgress*150)

		// Multiple debris particles
		for i := 0; i < 6; i++ {
			angle := float64(i) * (3.14159 * 2 / 6)
			dist := float64(4 + debrisProgress*8)
			px := centerX + float32(cosApprox(angle)*dist)
			py := centerY + float32(sinApprox(angle)*dist)

			// Debris colors: orange/red/yellow
			colors := []color.RGBA{
				{255, 100, 50, alpha},
				{255, 200, 50, alpha},
				{255, 50, 50, alpha},
			}
			c := colors[i%3]
			vector.DrawFilledRect(screen, px-1, py-1, 3, 3, c, false)
		}
	} else {
		// Fade out - small smoke puffs
		fadeProgress := (progress - 0.7) / 0.3
		alpha := uint8(100 - fadeProgress*100)

		// Gray smoke
		vector.DrawFilledCircle(screen, centerX, centerY, 4,
			color.RGBA{100, 100, 100, alpha}, false)
	}
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
func (s *GameplayScene) ShowPhaseSkipped(eventID, phase, reason string) {
	// Add to queue
	s.phaseSkipQueue = append(s.phaseSkipQueue, PhaseSkipData{
		EventID: eventID,
		Phase:   phase,
		Reason:  reason,
	})

	// If not currently showing a skip, start showing the first one
	if !s.showPhaseSkip {
		s.showNextPhaseSkip()
	}
}

// showNextPhaseSkip displays the next queued phase skip.
func (s *GameplayScene) showNextPhaseSkip() {
	// Send acknowledgment for the current skip before moving to next
	if s.phaseSkipEventID != "" {
		s.game.SendClientReady(s.phaseSkipEventID, protocol.EventPhaseSkip)
		s.phaseSkipEventID = ""
	}

	if len(s.phaseSkipQueue) == 0 {
		s.showPhaseSkip = false
		return
	}

	// Pop from queue
	skip := s.phaseSkipQueue[0]
	s.phaseSkipQueue = s.phaseSkipQueue[1:]

	s.phaseSkipEventID = skip.EventID
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
			reasonText = "by building cities"
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

// drawTradePropose draws the popup for proposing a trade.
func (s *GameplayScene) drawTradePropose(screen *ebiten.Image) {
	panelW, panelH := 560, 450
	// Make dialog taller when offering horses (need space for territory selection)
	if s.tradeOfferHorses > 0 {
		panelH = 530
	}
	centerX, centerY := ScreenWidth/2, ScreenHeight/2
	panelX, panelY := centerX-panelW/2, centerY-panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Propose Trade")

	y := panelY + 50

	// Get online players
	onlinePlayers := s.getOnlinePlayers()
	if len(onlinePlayers) == 0 {
		DrawTextCentered(screen, "No players available to trade with", centerX, y+100, ColorTextMuted)
		s.tradeCancelBtn.X = centerX - 50
		s.tradeCancelBtn.Y = panelY + panelH - 60
		s.tradeCancelBtn.Draw(screen)
		return
	}

	// Target player selection
	DrawText(screen, "Trade with:", panelX+20, y, ColorText)
	y += 25

	// Draw player buttons
	for i, playerID := range onlinePlayers {
		pData := s.players[playerID].(map[string]interface{})
		playerName := pData["name"].(string)

		btnX := panelX + 20 + (i%3)*150
		btnY := y + (i/3)*35

		isSelected := s.tradeTargetPlayer == playerID
		btnColor := ColorPanel
		if isSelected {
			btnColor = ColorSuccess
		}

		// Draw player button
		vector.DrawFilledRect(screen, float32(btnX), float32(btnY), 140, 30, btnColor, false)
		vector.StrokeRect(screen, float32(btnX), float32(btnY), 140, 30, 1, ColorBorder, false)
		DrawTextCentered(screen, playerName, btnX+70, btnY+8, ColorText)
	}

	// Calculate rows for players
	playerRows := (len(onlinePlayers) + 2) / 3
	y += playerRows*35 + 20

	// My resources section
	DrawText(screen, "I OFFER:", panelX+20, y, ColorSuccess)
	myCoal, myGold, myIron, myTimber := s.getMyStockpile()
	myHorses := s.countPlayerHorses(s.game.config.PlayerID)

	y += 25
	s.drawResourceAdjuster(screen, panelX+20, y, "Coal", &s.tradeOfferCoal, 0, myCoal)
	s.drawResourceAdjuster(screen, panelX+120, y, "Gold", &s.tradeOfferGold, 0, myGold)
	s.drawResourceAdjuster(screen, panelX+220, y, "Iron", &s.tradeOfferIron, 0, myIron)
	s.drawResourceAdjuster(screen, panelX+320, y, "Timber", &s.tradeOfferTimber, 0, myTimber)
	s.drawResourceAdjuster(screen, panelX+420, y, "Horses", &s.tradeOfferHorses, 0, myHorses)

	// I want section
	y += 70
	DrawText(screen, "I WANT:", panelX+20, y, ColorWarning)

	if s.tradeTargetPlayer != "" {
		targetCoal, targetGold, targetIron, targetTimber := s.getPlayerStockpile(s.tradeTargetPlayer)
		targetHorses := s.countPlayerHorses(s.tradeTargetPlayer)

		y += 25
		s.drawResourceAdjuster(screen, panelX+20, y, "Coal", &s.tradeRequestCoal, 0, targetCoal)
		s.drawResourceAdjuster(screen, panelX+120, y, "Gold", &s.tradeRequestGold, 0, targetGold)
		s.drawResourceAdjuster(screen, panelX+220, y, "Iron", &s.tradeRequestIron, 0, targetIron)
		s.drawResourceAdjuster(screen, panelX+320, y, "Timber", &s.tradeRequestTimber, 0, targetTimber)
		s.drawResourceAdjuster(screen, panelX+420, y, "Horses", &s.tradeRequestHorses, 0, targetHorses)
	} else {
		y += 25
		DrawText(screen, "Select a player first", panelX+20, y, ColorTextMuted)
	}

	// Horse territory selection (if offering horses)
	if s.tradeOfferHorses > 0 {
		y += 60
		DrawText(screen, "Select territories for horses:", panelX+20, y, ColorText)
		horseTerrs := s.getPlayerHorseTerritories()
		y += 20
		// Show up to 6 territories
		for i, terrID := range horseTerrs {
			if i >= 6 {
				break
			}
			tData := s.territories[terrID].(map[string]interface{})
			terrName := tData["name"].(string)

			btnX := panelX + 20 + (i%3)*150
			btnY := y + (i/3)*25

			// Check if selected
			isSelected := false
			for _, t := range s.tradeOfferHorseTerrs {
				if t == terrID {
					isSelected = true
					break
				}
			}

			btnColor := ColorPanel
			if isSelected {
				btnColor = ColorSuccess
			}

			vector.DrawFilledRect(screen, float32(btnX), float32(btnY), 140, 22, btnColor, false)
			DrawText(screen, terrName, btnX+5, btnY+4, ColorText)
		}
	}

	// Buttons
	canSend := s.tradeTargetPlayer != "" &&
		(s.tradeOfferCoal > 0 || s.tradeOfferGold > 0 || s.tradeOfferIron > 0 ||
			s.tradeOfferTimber > 0 || s.tradeOfferHorses > 0) &&
		(s.tradeRequestCoal > 0 || s.tradeRequestGold > 0 || s.tradeRequestIron > 0 ||
			s.tradeRequestTimber > 0 || s.tradeRequestHorses > 0)

	// Validate horse territories selected
	if s.tradeOfferHorses > 0 && len(s.tradeOfferHorseTerrs) < s.tradeOfferHorses {
		canSend = false
	}

	s.tradeSendBtn.X = centerX - 130
	s.tradeSendBtn.Y = panelY + panelH - 60
	s.tradeSendBtn.Disabled = !canSend
	s.tradeSendBtn.Draw(screen)

	s.tradeCancelBtn.X = centerX + 30
	s.tradeCancelBtn.Y = panelY + panelH - 60
	s.tradeCancelBtn.Draw(screen)
}

// drawResourceAdjuster draws a resource adjuster (+/- buttons with value).
// Note: Click handling is done in Update() via handleResourceAdjusterClick().
func (s *GameplayScene) drawResourceAdjuster(screen *ebiten.Image, x, y int, label string, value *int, min, max int) {
	DrawText(screen, label, x, y, ColorTextMuted)
	y += 18

	// Minus button
	minusBtnX, minusBtnY := x, y
	vector.DrawFilledRect(screen, float32(minusBtnX), float32(minusBtnY), 20, 20, ColorPanel, false)
	vector.StrokeRect(screen, float32(minusBtnX), float32(minusBtnY), 20, 20, 1, ColorBorder, false)
	DrawTextCentered(screen, "-", minusBtnX+10, minusBtnY+3, ColorText)

	// Value
	DrawTextCentered(screen, fmt.Sprintf("%d", *value), x+40, y+3, ColorText)

	// Plus button
	plusBtnX := x + 60
	vector.DrawFilledRect(screen, float32(plusBtnX), float32(minusBtnY), 20, 20, ColorPanel, false)
	vector.StrokeRect(screen, float32(plusBtnX), float32(minusBtnY), 20, 20, 1, ColorBorder, false)
	DrawTextCentered(screen, "+", plusBtnX+10, minusBtnY+3, ColorText)
}

// drawTradeIncoming draws the popup for an incoming trade proposal.
func (s *GameplayScene) drawTradeIncoming(screen *ebiten.Image) {
	if s.tradeProposal == nil {
		return
	}

	// Calculate panel height based on whether horses need destinations
	needsHorseDest := s.tradeProposal.OfferHorses > 0
	panelH := 300
	if needsHorseDest {
		panelH = 400
	}

	panelW := 400
	centerX, centerY := ScreenWidth/2, ScreenHeight/2
	panelX, panelY := centerX-panelW/2, centerY-panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Trade Proposal")

	y := panelY + 50
	DrawTextCentered(screen, fmt.Sprintf("%s wants to trade with you", s.tradeProposal.FromPlayerName), centerX, y, ColorText)

	y += 40
	DrawText(screen, "They Offer:", panelX+20, y, ColorSuccess)
	y += 20
	offerText := s.formatTradeResources(s.tradeProposal.OfferCoal, s.tradeProposal.OfferGold,
		s.tradeProposal.OfferIron, s.tradeProposal.OfferTimber, s.tradeProposal.OfferHorses)
	DrawText(screen, offerText, panelX+30, y, ColorText)

	y += 40
	DrawText(screen, "They Want:", panelX+20, y, ColorWarning)
	y += 20
	requestText := s.formatTradeResources(s.tradeProposal.RequestCoal, s.tradeProposal.RequestGold,
		s.tradeProposal.RequestIron, s.tradeProposal.RequestTimber, s.tradeProposal.RequestHorses)
	DrawText(screen, requestText, panelX+30, y, ColorText)

	// Horse destination selection if receiving horses
	if needsHorseDest {
		y += 40
		DrawText(screen, fmt.Sprintf("Select territories for %d horse(s):", s.tradeProposal.OfferHorses), panelX+20, y, ColorText)
		y += 20

		// Get our territories without horses
		availableTerrs := s.getTerritoriesWithoutHorses()
		for i, terrID := range availableTerrs {
			if i >= 6 {
				break
			}
			tData := s.territories[terrID].(map[string]interface{})
			terrName := tData["name"].(string)

			btnX := panelX + 20 + (i%3)*125
			btnY := y + (i/3)*25

			// Check if selected
			isSelected := false
			for _, t := range s.tradeHorseDestTerrs {
				if t == terrID {
					isSelected = true
					break
				}
			}

			btnColor := ColorPanel
			if isSelected {
				btnColor = ColorSuccess
			}

			vector.DrawFilledRect(screen, float32(btnX), float32(btnY), 120, 22, btnColor, false)
			DrawText(screen, terrName, btnX+5, btnY+4, ColorText)
		}
	}

	// Buttons
	canAccept := !needsHorseDest || len(s.tradeHorseDestTerrs) >= s.tradeProposal.OfferHorses

	s.tradeAcceptBtn.X = centerX - 110
	s.tradeAcceptBtn.Y = panelY + panelH - 60
	s.tradeAcceptBtn.Disabled = !canAccept
	s.tradeAcceptBtn.Draw(screen)

	s.tradeRejectBtn.X = centerX + 10
	s.tradeRejectBtn.Y = panelY + panelH - 60
	s.tradeRejectBtn.Draw(screen)
}

// formatTradeResources formats trade resources for display.
func (s *GameplayScene) formatTradeResources(coal, gold, iron, timber, horses int) string {
	parts := make([]string, 0)
	if coal > 0 {
		parts = append(parts, fmt.Sprintf("%d Coal", coal))
	}
	if gold > 0 {
		parts = append(parts, fmt.Sprintf("%d Gold", gold))
	}
	if iron > 0 {
		parts = append(parts, fmt.Sprintf("%d Iron", iron))
	}
	if timber > 0 {
		parts = append(parts, fmt.Sprintf("%d Timber", timber))
	}
	if horses > 0 {
		parts = append(parts, fmt.Sprintf("%d Horse(s)", horses))
	}
	if len(parts) == 0 {
		return "Nothing"
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

// drawTradeResult draws the trade result popup.
func (s *GameplayScene) drawTradeResult(screen *ebiten.Image) {
	panelW, panelH := 300, 150
	centerX, centerY := ScreenWidth/2, ScreenHeight/2
	panelX, panelY := centerX-panelW/2, centerY-panelH/2

	title := "Trade Result"
	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, title)

	y := panelY + 60
	if s.tradeResultAccepted {
		DrawTextCentered(screen, "Trade Accepted!", centerX, y, ColorSuccess)
	} else {
		DrawTextCentered(screen, "Trade Declined", centerX, y, ColorWarning)
	}

	if s.tradeResultMessage != "" {
		DrawTextCentered(screen, s.tradeResultMessage, centerX, y+25, ColorTextMuted)
	}

	s.tradeResultOkBtn.X = centerX - 50
	s.tradeResultOkBtn.Y = panelY + panelH - 50
	s.tradeResultOkBtn.Draw(screen)
}

// drawTradeWaiting draws the waiting for trade response indicator.
func (s *GameplayScene) drawTradeWaiting(screen *ebiten.Image) {
	panelW, panelH := 300, 100
	centerX, centerY := ScreenWidth/2, ScreenHeight/2
	panelX, panelY := centerX-panelW/2, centerY-panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "")
	DrawTextCentered(screen, "Waiting for response...", centerX, centerY-10, ColorText)
	DrawTextCentered(screen, "(60 second timeout)", centerX, centerY+15, ColorTextMuted)
}

// ==================== Production Animation ====================

// StartProductionAnimation begins the production animation sequence.
func (s *GameplayScene) StartProductionAnimation(payload *protocol.ProductionResultsPayload) {
	// Convert protocol payload to our animation data
	items := make([]ProductionItem, len(payload.Productions))
	for i, p := range payload.Productions {
		items[i] = ProductionItem{
			TerritoryID:     p.TerritoryID,
			TerritoryName:   p.TerritoryName,
			ResourceType:    p.ResourceType,
			Amount:          p.Amount,
			DestinationID:   p.DestinationID,
			DestinationName: p.DestinationName,
		}
	}

	s.productionAnimData = &ProductionAnimData{
		EventID:                payload.EventID,
		Productions:            items,
		StockpileTerritoryID:   payload.StockpileTerritoryID,
		StockpileTerritoryName: payload.StockpileTerritoryName,
	}

	s.productionAnimIndex = 0
	s.productionAnimProgress = 0
	s.productionAnimTimer = 0
	s.showProductionAnim = true

	log.Printf("Starting production animation with %d items", len(items))
}

// updateProductionAnimation updates the production animation state.
func (s *GameplayScene) updateProductionAnimation() {
	if !s.showProductionAnim || s.productionAnimData == nil {
		return
	}

	// Animation speed: 60 frames per item (1 second at 60fps)
	framesPerItem := 60

	s.productionAnimTimer++
	s.productionAnimProgress = float64(s.productionAnimTimer) / float64(framesPerItem)

	// Check if current item animation is complete
	if s.productionAnimProgress >= 1.0 {
		// Apply visual update for the completed item
		if s.productionAnimIndex < len(s.productionAnimData.Productions) {
			s.applyProductionItemVisual(s.productionAnimData.Productions[s.productionAnimIndex])
		}

		s.productionAnimIndex++
		s.productionAnimTimer = 0
		s.productionAnimProgress = 0

		// Check if all items are done
		if s.productionAnimIndex >= len(s.productionAnimData.Productions) {
			s.finishProductionAnimation()
		}
	}
}

// applyProductionItemVisual updates the client's visual state when a production item animation completes.
func (s *GameplayScene) applyProductionItemVisual(item ProductionItem) {
	// For horses (Grassland), show the horse on the destination territory immediately
	if item.ResourceType == "Grassland" && item.DestinationID != "" {
		if terr, ok := s.territories[item.DestinationID].(map[string]interface{}); ok {
			terr["hasHorse"] = true
			log.Printf("Production visual: Horse now visible on %s", item.DestinationName)
		}
	}

	// For resources, update stockpile display
	// The stockpile values are shown in the side panel from player data
	if item.ResourceType != "Grassland" && item.ResourceType != "None" {
		if player, ok := s.players[s.game.config.PlayerID].(map[string]interface{}); ok {
			if stockpile, ok := player["stockpile"].(map[string]interface{}); ok {
				resourceKey := strings.ToLower(item.ResourceType)
				if current, ok := stockpile[resourceKey].(float64); ok {
					stockpile[resourceKey] = current + float64(item.Amount)
					log.Printf("Production visual: Stockpile %s now %v", resourceKey, stockpile[resourceKey])
				}
			}
		}
	}
}

// finishProductionAnimation completes the animation and sends acknowledgment.
func (s *GameplayScene) finishProductionAnimation() {
	log.Printf("Production animation complete, sending acknowledgment")

	// Send acknowledgment to server
	if s.productionAnimData != nil {
		s.game.SendClientReady(s.productionAnimData.EventID, protocol.EventProduction)
	}

	// Clear animation state
	s.showProductionAnim = false
	s.productionAnimData = nil
	s.productionAnimIndex = 0
	s.productionAnimProgress = 0
	s.productionAnimTimer = 0
}

// drawProductionAnimation draws the production animation overlay.
func (s *GameplayScene) drawProductionAnimation(screen *ebiten.Image) {
	if !s.showProductionAnim || s.productionAnimData == nil {
		return
	}

	// Get current production item
	if s.productionAnimIndex >= len(s.productionAnimData.Productions) {
		return
	}
	item := s.productionAnimData.Productions[s.productionAnimIndex]

	// Get source position - where the resource icon is on the territory
	srcX, srcY := s.getResourceIconPosition(item.TerritoryID, item.ResourceType)

	// Destination: stockpile icon for resources, destination territory for horses
	var destX, destY int
	if item.ResourceType == "Grassland" && item.DestinationID != "" {
		// Horse goes to destination territory - find where horse icon will appear
		destX, destY = s.getFirstAvailableIconPosition(item.DestinationID)
	} else {
		// Resource goes to stockpile icon
		destX, destY = s.getStockpileIconPosition(s.productionAnimData.StockpileTerritoryID)
	}

	// Calculate current position (linear interpolation)
	progress := s.productionAnimProgress
	currentX := float32(srcX) + float32(destX-srcX)*float32(progress)
	currentY := float32(srcY) + float32(destY-srcY)*float32(progress)

	// Draw the moving resource icon using actual game icons
	s.drawProductionAnimIcon(screen, currentX, currentY, item.ResourceType)

	// Draw info panel at bottom
	s.drawProductionInfoPanel(screen, item, int(s.productionAnimIndex)+1, len(s.productionAnimData.Productions))
}

// getResourceIconPosition returns the screen position where a resource icon is drawn on a territory.
func (s *GameplayScene) getResourceIconPosition(territoryID, resourceType string) (int, int) {
	if s.mapData == nil {
		return ScreenWidth / 2, ScreenHeight / 2
	}

	grid := s.mapData["grid"].([]interface{})
	cells := s.findTerritoryCells(territoryID, grid)
	if len(cells) == 0 {
		return ScreenWidth / 2, ScreenHeight / 2
	}

	// Count icons that come before the resource in the drawing order
	// Order: stockpile, city, weapon, horse, then resource
	iconIndex := 0
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		if _, hasStockpile := terr["isStockpile"].(bool); hasStockpile {
			iconIndex++
		}
		if cities, ok := terr["cities"].(float64); ok && cities > 0 {
			iconIndex++
		}
		if hasWeapon, ok := terr["hasWeapon"].(bool); ok && hasWeapon {
			iconIndex++
		}
		if hasHorse, ok := terr["hasHorse"].(bool); ok && hasHorse {
			iconIndex++
		}
	}

	// Use the cell at iconIndex (or last available cell)
	cellIdx := iconIndex
	if cellIdx >= len(cells) {
		cellIdx = len(cells) - 1
	}

	cell := cells[cellIdx]
	sx, sy := s.gridToScreen(cell[0], cell[1])

	// Return center of the cell (icon is drawn centered)
	return sx + s.cellSize/2, sy + s.cellSize/2
}

// getStockpileIconPosition returns the screen position where the stockpile icon is drawn.
func (s *GameplayScene) getStockpileIconPosition(territoryID string) (int, int) {
	if s.mapData == nil {
		return ScreenWidth / 2, ScreenHeight / 2
	}

	grid := s.mapData["grid"].([]interface{})
	cells := s.findTerritoryCells(territoryID, grid)
	if len(cells) == 0 {
		return ScreenWidth / 2, ScreenHeight / 2
	}

	// Stockpile is always the first icon (index 0)
	cell := cells[0]
	sx, sy := s.gridToScreen(cell[0], cell[1])

	return sx + s.cellSize/2, sy + s.cellSize/2
}

// getFirstAvailableIconPosition returns the screen position for the first available icon slot on a territory.
func (s *GameplayScene) getFirstAvailableIconPosition(territoryID string) (int, int) {
	if s.mapData == nil {
		return ScreenWidth / 2, ScreenHeight / 2
	}

	grid := s.mapData["grid"].([]interface{})
	cells := s.findTerritoryCells(territoryID, grid)
	if len(cells) == 0 {
		return ScreenWidth / 2, ScreenHeight / 2
	}

	// Count existing icons to find next available slot
	iconIndex := 0
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		if _, hasStockpile := terr["isStockpile"].(bool); hasStockpile {
			iconIndex++
		}
		if cities, ok := terr["cities"].(float64); ok && cities > 0 {
			iconIndex++
		}
		if hasWeapon, ok := terr["hasWeapon"].(bool); ok && hasWeapon {
			iconIndex++
		}
		// Horse slot - this is where a new horse would go
	}

	cellIdx := iconIndex
	if cellIdx >= len(cells) {
		cellIdx = len(cells) - 1
	}

	cell := cells[cellIdx]
	sx, sy := s.gridToScreen(cell[0], cell[1])

	return sx + s.cellSize/2, sy + s.cellSize/2
}

// drawProductionAnimIcon draws a resource/horse icon at the given position for animation.
func (s *GameplayScene) drawProductionAnimIcon(screen *ebiten.Image, x, y float32, resourceType string) {
	// Determine icon size based on cell size (matches map icon sizing)
	// cellSize already includes zoom, so this scales automatically
	iconSize := float32(s.cellSize) * 0.65

	// Get the appropriate icon
	var iconImg *ebiten.Image
	switch resourceType {
	case "Coal":
		iconImg = GetIcon("coal")
	case "Gold":
		iconImg = GetIcon("gold")
	case "Iron":
		iconImg = GetIcon("iron")
	case "Timber":
		iconImg = GetIcon("timber")
	case "Grassland":
		iconImg = GetIcon("horse") // Grassland produces horses
	}

	// Calculate position (center the icon at x, y)
	drawX := x - iconSize/2
	drawY := y - iconSize/2

	if iconImg != nil {
		// Draw the PNG icon scaled to fit
		op := &ebiten.DrawImageOptions{}
		imgW := float32(iconImg.Bounds().Dx())
		imgH := float32(iconImg.Bounds().Dy())
		scaleX := iconSize / imgW
		scaleY := iconSize / imgH
		op.GeoM.Scale(float64(scaleX), float64(scaleY))
		op.GeoM.Translate(float64(drawX), float64(drawY))
		screen.DrawImage(iconImg, op)
	} else {
		// Fallback: draw a colored circle
		var iconColor color.RGBA
		switch resourceType {
		case "Coal":
			iconColor = color.RGBA{50, 50, 50, 255}
		case "Gold":
			iconColor = color.RGBA{255, 215, 0, 255}
		case "Iron":
			iconColor = color.RGBA{150, 150, 170, 255}
		case "Timber":
			iconColor = color.RGBA{139, 90, 43, 255}
		case "Grassland":
			iconColor = color.RGBA{100, 180, 100, 255}
		default:
			iconColor = color.RGBA{200, 200, 200, 255}
		}
		vector.DrawFilledCircle(screen, x, y, iconSize/2, iconColor, false)
		vector.StrokeCircle(screen, x, y, iconSize/2, 2, color.White, false)
	}
}

// drawProductionInfoPanel draws information about the current production.
func (s *GameplayScene) drawProductionInfoPanel(screen *ebiten.Image, item ProductionItem, current, total int) {
	panelW, panelH := 400, 80
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight - 200

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Production")

	// Resource info
	resourceName := item.ResourceType
	if resourceName == "Grassland" {
		resourceName = "Horse"
	}

	fromText := fmt.Sprintf("From: %s", item.TerritoryName)
	toText := ""
	if item.ResourceType == "Grassland" && item.DestinationName != "" {
		toText = fmt.Sprintf("To: %s", item.DestinationName)
	} else {
		toText = fmt.Sprintf("To: Stockpile (%s)", s.productionAnimData.StockpileTerritoryName)
	}

	amountText := fmt.Sprintf("+%d %s", item.Amount, resourceName)

	DrawText(screen, fromText, panelX+15, panelY+35, ColorText)
	DrawText(screen, toText, panelX+15, panelY+50, ColorText)
	DrawText(screen, amountText, panelX+panelW-100, panelY+42, ColorSuccess)

	// Progress indicator
	progressText := fmt.Sprintf("%d / %d", current, total)
	DrawText(screen, progressText, panelX+panelW-60, panelY+60, ColorTextMuted)
}

// ==================== Stockpile Capture Animation ====================

// startStockpileCaptureAnimation starts the animation for transferring captured stockpile resources.
func (s *GameplayScene) startStockpileCaptureAnimation(combatResult *CombatResultData) {
	// Find player's stockpile territory
	playerStockpileTerr := ""
	if myPlayer, ok := s.players[s.game.config.PlayerID].(map[string]interface{}); ok {
		if stockpileTerr, ok := myPlayer["stockpileTerritory"].(string); ok {
			playerStockpileTerr = stockpileTerr
		}
	}

	if playerStockpileTerr == "" {
		// No stockpile, just show combat result
		log.Printf("No stockpile territory found, skipping capture animation")
		s.combatResult = combatResult
		s.showCombatResult = true
		return
	}

	// Build list of resources to animate
	resources := make([]CapturedResource, 0)
	if combatResult.CapturedCoal > 0 {
		resources = append(resources, CapturedResource{"Coal", combatResult.CapturedCoal})
	}
	if combatResult.CapturedGold > 0 {
		resources = append(resources, CapturedResource{"Gold", combatResult.CapturedGold})
	}
	if combatResult.CapturedIron > 0 {
		resources = append(resources, CapturedResource{"Iron", combatResult.CapturedIron})
	}
	if combatResult.CapturedTimber > 0 {
		resources = append(resources, CapturedResource{"Timber", combatResult.CapturedTimber})
	}

	if len(resources) == 0 {
		// No resources to animate, just show combat result
		log.Printf("Stockpile captured but no resources, skipping animation")
		s.combatResult = combatResult
		s.showCombatResult = true
		return
	}

	s.stockpileCaptureData = &StockpileCaptureData{
		FromTerritoryID:   combatResult.CapturedFromTerritory,
		ToTerritoryID:     playerStockpileTerr,
		Resources:         resources,
		PendingEventID:    combatResult.EventID,
		PendingCombatData: combatResult,
	}
	s.stockpileCaptureIndex = 0
	s.stockpileCaptureProgress = 0
	s.stockpileCaptureTimer = 0
	s.showStockpileCapture = true

	log.Printf("Starting stockpile capture animation: %d resources from %s to %s",
		len(resources), combatResult.CapturedFromTerritory, playerStockpileTerr)
}

// updateStockpileCaptureAnimation updates the stockpile capture animation state.
func (s *GameplayScene) updateStockpileCaptureAnimation() {
	if !s.showStockpileCapture || s.stockpileCaptureData == nil {
		return
	}

	// Animation speed: 45 frames per resource (0.75 seconds at 60fps)
	framesPerResource := 45

	s.stockpileCaptureTimer++
	s.stockpileCaptureProgress = float64(s.stockpileCaptureTimer) / float64(framesPerResource)

	// Check if current resource animation is complete
	if s.stockpileCaptureProgress >= 1.0 {
		// Apply visual update for the completed resource
		if s.stockpileCaptureIndex < len(s.stockpileCaptureData.Resources) {
			s.applyStockpileCaptureVisual(s.stockpileCaptureData.Resources[s.stockpileCaptureIndex])
		}

		s.stockpileCaptureIndex++
		s.stockpileCaptureTimer = 0
		s.stockpileCaptureProgress = 0

		// Check if all resources are done
		if s.stockpileCaptureIndex >= len(s.stockpileCaptureData.Resources) {
			s.finishStockpileCaptureAnimation()
		}
	}
}

// applyStockpileCaptureVisual updates the client's visual state when a captured resource animation completes.
func (s *GameplayScene) applyStockpileCaptureVisual(resource CapturedResource) {
	if player, ok := s.players[s.game.config.PlayerID].(map[string]interface{}); ok {
		if stockpile, ok := player["stockpile"].(map[string]interface{}); ok {
			resourceKey := strings.ToLower(resource.ResourceType)
			if current, ok := stockpile[resourceKey].(float64); ok {
				stockpile[resourceKey] = current + float64(resource.Amount)
				log.Printf("Stockpile capture visual: %s now %v", resourceKey, stockpile[resourceKey])
			}
		}
	}
}

// finishStockpileCaptureAnimation completes the stockpile capture animation.
func (s *GameplayScene) finishStockpileCaptureAnimation() {
	log.Printf("Stockpile capture animation complete")

	// Show the combat result dialog
	s.combatResult = s.stockpileCaptureData.PendingCombatData
	s.showCombatResult = true

	// Clear animation state
	s.showStockpileCapture = false
	s.stockpileCaptureData = nil
	s.stockpileCaptureIndex = 0
	s.stockpileCaptureProgress = 0
	s.stockpileCaptureTimer = 0
}

// drawStockpileCaptureAnimation draws the stockpile capture animation overlay.
func (s *GameplayScene) drawStockpileCaptureAnimation(screen *ebiten.Image) {
	if !s.showStockpileCapture || s.stockpileCaptureData == nil {
		return
	}

	// Get current resource
	if s.stockpileCaptureIndex >= len(s.stockpileCaptureData.Resources) {
		return
	}
	resource := s.stockpileCaptureData.Resources[s.stockpileCaptureIndex]

	// Get source position (captured stockpile location)
	srcX, srcY := s.getStockpileIconPosition(s.stockpileCaptureData.FromTerritoryID)

	// Get destination position (player's stockpile)
	destX, destY := s.getStockpileIconPosition(s.stockpileCaptureData.ToTerritoryID)

	// Calculate current position (linear interpolation)
	progress := s.stockpileCaptureProgress
	currentX := float32(srcX) + float32(destX-srcX)*float32(progress)
	currentY := float32(srcY) + float32(destY-srcY)*float32(progress)

	// Draw the moving resource icon
	s.drawProductionAnimIcon(screen, currentX, currentY, resource.ResourceType)

	// Draw info panel
	s.drawStockpileCapturePanel(screen, resource, s.stockpileCaptureIndex+1, len(s.stockpileCaptureData.Resources))
}

// drawStockpileCapturePanel draws information about the stockpile capture.
func (s *GameplayScene) drawStockpileCapturePanel(screen *ebiten.Image, resource CapturedResource, current, total int) {
	panelW, panelH := 350, 80
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight - 200

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Stockpile Captured!")

	// Resource info
	amountText := fmt.Sprintf("+%d %s", resource.Amount, resource.ResourceType)
	DrawText(screen, "Transferring resources to your stockpile", panelX+15, panelY+35, ColorText)
	DrawText(screen, amountText, panelX+panelW/2-30, panelY+52, ColorSuccess)

	// Progress indicator
	progressText := fmt.Sprintf("%d / %d", current, total)
	DrawText(screen, progressText, panelX+panelW-60, panelY+60, ColorTextMuted)
}
