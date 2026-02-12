package client

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"strings"

	"lords-of-conquest/internal/game"
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

	// Result panel
	panelW := 380
	panelH := 240
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	// Panel title based on result
	if s.combatResult.AttackerWins {
		DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "VICTORY!")
	} else {
		DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "DEFEAT")
	}

	// Attacker vs Defender line
	attackerLabel := s.combatResult.AttackerName
	if attackerLabel == "" {
		attackerLabel = "Unknown"
	}
	defenderLabel := s.combatResult.DefenderName
	if defenderLabel == "" {
		defenderLabel = "Unclaimed"
	}
	vsText := fmt.Sprintf("%s  vs  %s", attackerLabel, defenderLabel)
	DrawTextCentered(screen, vsText, ScreenWidth/2, panelY+48, ColorText)

	// Territory name
	DrawTextCentered(screen, s.combatResult.TargetName, ScreenWidth/2, panelY+68, ColorTextMuted)

	// Strength comparison
	strengthText := fmt.Sprintf("Attack %d  vs  Defense %d", s.combatResult.AttackStrength, s.combatResult.DefenseStrength)
	DrawTextCentered(screen, strengthText, ScreenWidth/2, panelY+95, ColorText)

	// Result text
	var resultText string
	var resultColor color.RGBA
	if s.combatResult.AttackerWins {
		resultText = "Attack Successful!"
		resultColor = ColorSuccess
	} else {
		resultText = "Attack Repulsed!"
		resultColor = ColorDanger
	}
	DrawLargeTextCentered(screen, resultText, ScreenWidth/2, panelY+125, resultColor)

	// Outcome description
	var outcomeText string
	if s.combatResult.AttackerWins {
		outcomeText = fmt.Sprintf("%s captured %s!", attackerLabel, s.combatResult.TargetName)
	} else {
		outcomeText = fmt.Sprintf("%s defended %s.", defenderLabel, s.combatResult.TargetName)
	}
	DrawTextCentered(screen, outcomeText, ScreenWidth/2, panelY+150, ColorTextMuted)

	// OK button
	s.dismissResultBtn.X = ScreenWidth/2 - 60
	s.dismissResultBtn.Y = panelY + panelH - 55
	s.dismissResultBtn.Draw(screen)
}

// drawWaterBodySelect draws the water body selection UI for boat placement
func (s *GameplayScene) drawWaterBodySelect(screen *ebiten.Image) {
	// Semi-transparent overlay (less opaque so map is visible)
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 120}, false)

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

	barX := 10
	barY := s.currentBarTop
	barW := ScreenWidth - 20

	// Get target name
	targetName := s.attackPlanTarget
	if terr, ok := s.territories[s.attackPlanTarget].(map[string]interface{}); ok {
		if name, ok := terr["name"].(string); ok {
			targetName = name
		}
	}

	reinforceCount := len(s.attackPreview.Reinforcements)

	// === LEFT SECTION: Title + Strength ===
	DrawLargeText(screen, "Plan Attack: "+targetName, barX+20, barY+12, ColorText)

	// Strength preview
	attackStr := fmt.Sprintf("%d", s.attackPreview.AttackStrength)
	if s.attackPreview.AttackerAllyStrength > 0 {
		attackStr = fmt.Sprintf("%d (+%d allies)", s.attackPreview.AttackStrength, s.attackPreview.AttackerAllyStrength)
	}
	defenseStr := fmt.Sprintf("%d", s.attackPreview.DefenseStrength)
	if s.attackPreview.DefenderAllyStrength > 0 {
		defenseStr = fmt.Sprintf("%d (+%d allies)", s.attackPreview.DefenseStrength, s.attackPreview.DefenderAllyStrength)
	}
	strengthText := fmt.Sprintf("Atk: %s  vs  Def: %s", attackStr, defenseStr)
	DrawText(screen, strengthText, barX+20, barY+40, ColorTextMuted)

	// === CENTER SECTION: Reinforcements ===
	reinfX := barX + 420
	if reinforceCount > 0 {
		DrawText(screen, "Reinforcements (click to toggle):", reinfX, barY+12, ColorText)

		for i, reinf := range s.attackPreview.Reinforcements {
			fromName := reinf.FromTerritory
			if terr, ok := s.territories[reinf.FromTerritory].(map[string]interface{}); ok {
				if name, ok := terr["name"].(string); ok {
					fromName = name
				}
			}

			isSelected := s.selectedReinforcement != nil &&
				s.selectedReinforcement.FromTerritory == reinf.FromTerritory &&
				s.selectedReinforcement.UnitType == reinf.UnitType

			optY := barY + 35 + i*50
			boxW := 380
			boxColor := color.RGBA{50, 50, 70, 255}
			if isSelected {
				boxColor = color.RGBA{80, 100, 150, 255}
			}
			vector.DrawFilledRect(screen, float32(reinfX), float32(optY), float32(boxW), 45, boxColor, false)
			if isSelected {
				vector.StrokeRect(screen, float32(reinfX), float32(optY), float32(boxW), 45, 2, ColorBorder, false)
			}

			unitLabel := fmt.Sprintf("%s from %s (+%d)", reinf.UnitType, fromName, reinf.StrengthBonus)
			DrawText(screen, unitLabel, reinfX+10, optY+10, ColorText)

			carryText := ""
			if reinf.UnitType == "boat" {
				if reinf.CanCarryHorse {
					if reinf.HorseStrengthBonus > 0 {
						carryText += fmt.Sprintf("Can load Horse (+%d) ", reinf.HorseStrengthBonus)
					} else {
						carryText += "Can load Horse (in range) "
					}
				}
				if reinf.CanCarryWeapon {
					if reinf.WeaponStrengthBonus > 0 {
						carryText += fmt.Sprintf("Can load Weapon (+%d)", reinf.WeaponStrengthBonus)
					} else {
						carryText += "Can load Weapon (in range)"
					}
				}
			} else if reinf.UnitType == "horse" && reinf.CanCarryWeapon {
				if reinf.WeaponStrengthBonus > 0 {
					carryText = fmt.Sprintf("Can carry Weapon (+%d)", reinf.WeaponStrengthBonus)
				} else {
					carryText = "Can carry Weapon (in range)"
				}
			}
			if carryText != "" {
				DrawText(screen, carryText, reinfX+10, optY+28, ColorTextDim)
			}
		}

		// Cargo checkboxes below reinforcements
		checkboxY := barY + 35 + reinforceCount*50
		if s.selectedReinforcement != nil {
			if s.selectedReinforcement.UnitType == "boat" {
				if s.selectedReinforcement.CanCarryHorse {
					horseLabel := "Load Horse onto Boat"
					if s.selectedReinforcement.HorseStrengthBonus > 0 {
						horseLabel = fmt.Sprintf("Load Horse onto Boat (+%d)", s.selectedReinforcement.HorseStrengthBonus)
					} else {
						horseLabel = "Load Horse onto Boat (already in range)"
					}
					s.drawCheckbox(screen, reinfX, checkboxY+5, horseLabel, &s.loadHorseCheckbox)
					checkboxY += 25
				}
				if s.selectedReinforcement.CanCarryWeapon {
					weaponLabel := "Load Weapon onto Boat"
					if s.selectedReinforcement.WeaponStrengthBonus > 0 {
						weaponLabel = fmt.Sprintf("Load Weapon onto Boat (+%d)", s.selectedReinforcement.WeaponStrengthBonus)
					} else {
						weaponLabel = "Load Weapon onto Boat (already in range)"
					}
					s.drawCheckbox(screen, reinfX, checkboxY+5, weaponLabel, &s.loadWeaponCheckbox)
					checkboxY += 25
				}
			} else if s.selectedReinforcement.UnitType == "horse" && s.selectedReinforcement.CanCarryWeapon {
				weaponLabel := "Carry Weapon on Horse"
				if s.selectedReinforcement.WeaponStrengthBonus > 0 {
					weaponLabel = fmt.Sprintf("Carry Weapon on Horse (+%d)", s.selectedReinforcement.WeaponStrengthBonus)
				} else {
					weaponLabel = "Carry Weapon on Horse (already in range)"
				}
				s.drawCheckbox(screen, reinfX, checkboxY+5, weaponLabel, &s.loadWeaponCheckbox)
			}
		}
	} else {
		DrawText(screen, "No reinforcements available", reinfX, barY+30, ColorTextDim)
	}

	// === RIGHT SECTION: Buttons (stacked vertically) ===
	btnWidth := 150
	btnX := barX + barW - btnWidth - 20

	// Plan Attack button (uses selected reinforcement if any)
	canAttack := s.attackPreview.AttackStrength > 0 || s.selectedReinforcement != nil
	if canAttack {
		s.attackNoReinfBtn.Text = "Plan Attack"
		s.attackNoReinfBtn.W = btnWidth
		s.attackNoReinfBtn.X = btnX
		s.attackNoReinfBtn.Y = barY + 15
		s.attackNoReinfBtn.Draw(screen)
	} else {
		DrawText(screen, "Bring forces", btnX, barY+20, ColorWarning)
		DrawText(screen, "to attack", btnX, barY+38, ColorWarning)
	}

	// Cancel button
	s.cancelAttackBtn.W = btnWidth
	s.cancelAttackBtn.X = btnX
	s.cancelAttackBtn.Y = barY + 60
	s.cancelAttackBtn.Draw(screen)

	// Card selection hint (card combat mode only)
	if s.combatMode == "cards" && len(s.myAttackCards) > 0 {
		selectedCount := len(s.selectedCardIDs)
		if selectedCount > 0 {
			DrawText(screen, fmt.Sprintf("%d attack card(s) selected", selectedCount), barX+20, barY+60, ColorSuccess)
		} else {
			DrawText(screen, "Click attack cards below to select", barX+20, barY+60, ColorTextDim)
		}
		DrawText(screen, "Esc to cancel", barX+20, barY+78, ColorTextDim)
	} else {
		DrawText(screen, "Esc to cancel", barX+20, barY+60, ColorTextDim)
	}
}

// updateAttackPlanInput handles input for the attack planning dialog (called from Update)
func (s *GameplayScene) updateAttackPlanInput() {
	if s.attackPreview == nil {
		return
	}

	barX := 10
	barY := s.currentBarTop
	reinfX := barX + 420
	reinforceCount := len(s.attackPreview.Reinforcements)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()

		// Handle reinforcement selection clicks (click to select, click again to deselect)
		if reinforceCount > 0 {
			for i, reinf := range s.attackPreview.Reinforcements {
				optY := barY + 35 + i*50
				boxW := 380
				if mx >= reinfX && mx <= reinfX+boxW &&
					my >= optY && my <= optY+45 {
					// Toggle: if already selected, deselect it
					if s.selectedReinforcement != nil &&
						s.selectedReinforcement.FromTerritory == reinf.FromTerritory &&
						s.selectedReinforcement.UnitType == reinf.UnitType {
						s.selectedReinforcement = nil
						s.loadHorseCheckbox = false
						s.loadWeaponCheckbox = false
					} else {
						s.selectedReinforcement = &ReinforcementData{
							UnitType:            reinf.UnitType,
							FromTerritory:       reinf.FromTerritory,
							WaterBodyID:         reinf.WaterBodyID,
							StrengthBonus:       reinf.StrengthBonus,
							CanCarryWeapon:      reinf.CanCarryWeapon,
							WeaponStrengthBonus: reinf.WeaponStrengthBonus,
							CanCarryHorse:       reinf.CanCarryHorse,
							HorseStrengthBonus:  reinf.HorseStrengthBonus,
						}
						s.loadHorseCheckbox = false
						s.loadWeaponCheckbox = false
					}
					break
				}
			}

			// Handle checkbox clicks
			if s.selectedReinforcement != nil {
				checkboxY := barY + 35 + reinforceCount*50
				boxSize := 16

				if s.selectedReinforcement.UnitType == "boat" {
					if s.selectedReinforcement.CanCarryHorse {
						if mx >= reinfX && mx <= reinfX+boxSize+300 &&
							my >= checkboxY+5 && my <= checkboxY+5+boxSize {
							s.loadHorseCheckbox = !s.loadHorseCheckbox
						}
						checkboxY += 25
					}
					if s.selectedReinforcement.CanCarryWeapon {
						if mx >= reinfX && mx <= reinfX+boxSize+300 &&
							my >= checkboxY+5 && my <= checkboxY+5+boxSize {
							s.loadWeaponCheckbox = !s.loadWeaponCheckbox
						}
					}
				} else if s.selectedReinforcement.UnitType == "horse" && s.selectedReinforcement.CanCarryWeapon {
					if mx >= reinfX && mx <= reinfX+boxSize+300 &&
						my >= checkboxY+5 && my <= checkboxY+5+boxSize {
						s.loadWeaponCheckbox = !s.loadWeaponCheckbox
					}
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

// ShowCombatResult starts the combat animation before displaying the result.
func (s *GameplayScene) ShowCombatResult(result *CombatResultData) {
	// Highlight the target territory
	s.SetHighlightedTerritories([]TerritoryHighlight{
		{TerritoryID: result.TargetTerritory, Color: color.RGBA{255, 80, 80, 255}},
	})

	// Start the combat animation directly - server handles synchronization
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

// showCombatResultAsNotification shows the combat result in the bottom bar notification.
func (s *GameplayScene) showCombatResultAsNotification() {
	if s.combatResult == nil {
		return
	}
	r := s.combatResult
	var msg string
	if r.AttackerWins {
		msg = fmt.Sprintf("ATTACK SUCCESSFUL! %s captured %s -- Atk: %d vs Def: %d", r.AttackerName, r.TargetName, r.AttackStrength, r.DefenseStrength)
	} else {
		msg = fmt.Sprintf("ATTACK REPULSED! %s defended %s -- Atk: %d vs Def: %d", r.DefenderName, r.TargetName, r.AttackStrength, r.DefenseStrength)
	}
	s.showBottomBarNotification(msg, "OK", func() {
		s.dismissCombatResult()
	})
}

// dismissCombatResult dismisses the combat result and sends ack to the server.
func (s *GameplayScene) dismissCombatResult() {
	// Send acknowledgment for this combat result
	if s.combatResult != nil && s.combatResult.EventID != "" {
		s.game.SendClientReady(s.combatResult.EventID, protocol.EventCombat)
	}

	s.showCombatResult = false
	s.combatResult = nil
	s.ClearHighlightedTerritories()
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

		// Check if we captured a stockpile and it's our attack - animate the resource transfer
		isMyAttack := s.combatPendingResult.AttackerID == s.game.config.PlayerID
		log.Printf("Combat animation ended: AttackerID=%s, MyPlayerID=%s, isMyAttack=%v",
			s.combatPendingResult.AttackerID, s.game.config.PlayerID, isMyAttack)

		if isMyAttack && s.combatPendingResult.StockpileCaptured {
			// Start stockpile capture animation
			s.startStockpileCaptureAnimation(s.combatPendingResult)
			return
		}

		// Show combat result as bottom bar notification
		s.combatResult = s.combatPendingResult
		s.showCombatResultAsNotification()
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
	s.selectedAttackCardIDs = make(map[string]bool)
	s.selectedCardIDs = make(map[string]bool) // For card hand click-to-toggle
	s.showAttackPlan = true

	// Calculate dynamic bar height based on content
	// Right section buttons need: Plan Attack at +15 (h40) + Cancel at +60 (h40) = 100 + padding
	reinforceCount := len(preview.Reinforcements)
	barH := 105 // base: enough for buttons + text with no reinforcements
	if reinforceCount > 0 {
		// Reinforcement boxes start at +35, each is 45px tall with 5px gap (50px per item)
		// Below boxes: up to 2 checkboxes (25px each) + 5px offset = 55px
		reinfH := 35 + reinforceCount*50 + 55 + 10 // +10 bottom padding
		if reinfH > barH {
			barH = reinfH
		}
	}
	s.SetBarHeight(barH)

	// Highlight the target territory and auto-pan to it
	s.SetHighlightedTerritories([]TerritoryHighlight{
		{TerritoryID: preview.TargetTerritory, Color: color.RGBA{255, 80, 80, 255}},
	})
}

// doAttack requests the attack plan (triggers alliance resolution)
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

	log.Printf("Requesting attack plan for %s with reinforcement: %v", s.attackPlanTarget, reinforcement)

	// Store the reinforcement selection for later confirmation
	// Hide the attack plan dialog and show waiting overlay in bottom bar
	s.showAttackPlan = false
	s.ResetBarHeight()
	s.showWaitingForAlliance = true
	s.showBottomBarNotification("Planning attack... Waiting for alliance decisions (up to 60s)", "", nil)

	// Send the request to server to resolve alliances
	s.game.RequestAttackPlan(s.attackPlanTarget, reinforcement)
}

// cancelAttackPlan cancels the attack planning
func (s *GameplayScene) cancelAttackPlan() {
	s.showAttackPlan = false
	s.attackPlanTarget = ""
	s.attackPreview = nil
	s.selectedReinforcement = nil
	s.selectedCardIDs = make(map[string]bool)
	s.ClearHighlightedTerritories()
	s.loadHorseCheckbox = false
	s.loadWeaponCheckbox = false
	s.ResetBarHeight()
}

// drawDiplomacyMenu draws the diplomacy menu (alliance + surrender options)
// Uses two-column layout for player lists to fit 8 players on screen
func (s *GameplayScene) drawDiplomacyMenu(screen *ebiten.Image) {
	// Semi-transparent overlay (less opaque so map is visible)
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 120}, false)

	// Count other non-eliminated players for menu sizing
	otherPlayerCount := 0
	for _, playerIDInterface := range s.playerOrder {
		playerID := playerIDInterface.(string)
		if playerID == s.game.config.PlayerID {
			continue
		}
		if playerData, ok := s.players[playerID]; ok {
			player := playerData.(map[string]interface{})
			if eliminated, ok := player["eliminated"].(bool); ok && eliminated {
				continue
			}
			otherPlayerCount++
		}
	}

	// Check if current player is eliminated (surrendered)
	amEliminated := false
	if myPlayer, ok := s.players[s.game.config.PlayerID]; ok {
		player := myPlayer.(map[string]interface{})
		if eliminated, ok := player["eliminated"].(bool); ok && eliminated {
			amEliminated = true
		}
	}

	// Two-column layout for player lists
	// Top section: Current alliance display + 3 mode buttons (2 rows)
	// Bottom section: Two columns - "Ally with" on left, "Surrender to" on right
	colWidth := 180
	colGap := 20
	menuW := colWidth*2 + colGap + 60 // Two columns + gap + margins

	// Calculate height based on player count (players shown in columns)
	topSectionH := 45 + 90                     // Header/current + mode buttons (2 rows)
	playerSectionH := 25 + otherPlayerCount*40 // Label + player buttons
	cancelSectionH := 55

	// If not eliminated, we show surrender column too, but same height as ally column
	menuH := topSectionH + playerSectionH + cancelSectionH

	menuX := ScreenWidth/2 - menuW/2
	menuY := ScreenHeight/2 - menuH/2

	DrawFancyPanel(screen, menuX, menuY, menuW, menuH, "Diplomacy")

	// Current alliance setting display
	currentText := "Alliance: "
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

	// Mode buttons in a row (or two rows if needed)
	btnY := menuY + 55
	btnW := 130
	btnH := 35
	btnGap := 10

	// First row: Ask and Defender
	s.allyAskBtn.X = menuX + 20
	s.allyAskBtn.Y = btnY
	s.allyAskBtn.W = btnW
	s.allyAskBtn.H = btnH
	s.allyAskBtn.Primary = s.myAllianceSetting == "ask"
	s.allyAskBtn.Draw(screen)

	s.allyDefenderBtn.X = menuX + 20 + btnW + btnGap
	s.allyDefenderBtn.Y = btnY
	s.allyDefenderBtn.W = btnW
	s.allyDefenderBtn.H = btnH
	s.allyDefenderBtn.Primary = s.myAllianceSetting == "defender"
	s.allyDefenderBtn.Draw(screen)

	s.allyNeutralBtn.X = menuX + 20 + (btnW+btnGap)*2
	s.allyNeutralBtn.Y = btnY
	s.allyNeutralBtn.W = btnW
	s.allyNeutralBtn.H = btnH
	s.allyNeutralBtn.Primary = s.myAllianceSetting == "neutral"
	s.allyNeutralBtn.Draw(screen)

	btnY += 50 // Move past mode buttons

	// Two-column player section
	leftColX := menuX + 20
	rightColX := menuX + 20 + colWidth + colGap

	// Left column: Ally with player
	if otherPlayerCount > 0 {
		DrawText(screen, "Ally with:", leftColX, btnY+5, ColorTextMuted)

		// Right column header (only if not eliminated)
		if !amEliminated {
			DrawText(screen, "Surrender to:", rightColX, btnY+5, ColorWarning)
		}

		playerBtnY := btnY + 25

		// Rebuild player buttons lists
		s.allyPlayerBtns = make([]*Button, 0, otherPlayerCount)
		s.allyPlayerIDs = make([]string, 0, otherPlayerCount)
		s.surrenderPlayerBtns = make([]*Button, 0, otherPlayerCount)

		for _, playerIDInterface := range s.playerOrder {
			playerID := playerIDInterface.(string)
			if playerID == s.game.config.PlayerID {
				continue
			}
			if playerData, ok := s.players[playerID]; ok {
				player := playerData.(map[string]interface{})
				// Skip eliminated players
				if eliminated, ok := player["eliminated"].(bool); ok && eliminated {
					continue
				}
				playerName := player["name"].(string)

				// Left column: Ally button
				allyBtn := &Button{
					X: leftColX, Y: playerBtnY, W: colWidth, H: 32,
					Text:    playerName,
					Primary: s.myAllianceSetting == playerID,
				}
				pid := playerID
				allyBtn.OnClick = func() { s.setAlliance(pid) }
				allyBtn.Draw(screen)

				s.allyPlayerBtns = append(s.allyPlayerBtns, allyBtn)
				s.allyPlayerIDs = append(s.allyPlayerIDs, playerID)

				// Right column: Surrender button (only if not eliminated)
				if !amEliminated {
					surrenderBtn := &Button{
						X: rightColX, Y: playerBtnY, W: colWidth, H: 32,
						Text:    playerName,
						Primary: false,
					}
					pname := playerName
					surrenderBtn.OnClick = func() {
						s.surrenderTargetID = pid
						s.surrenderTargetName = pname
						s.showSurrenderConfirm = true
						s.showAllyMenu = false
					}
					surrenderBtn.Draw(screen)
					s.surrenderPlayerBtns = append(s.surrenderPlayerBtns, surrenderBtn)
				}

				playerBtnY += 40
			}
		}
		btnY = playerBtnY
	}

	// Cancel button centered at bottom
	btnY += 10
	s.cancelAllyMenuBtn.X = menuX + menuW/2 - 100
	s.cancelAllyMenuBtn.Y = btnY
	s.cancelAllyMenuBtn.Draw(screen)
}

// drawSurrenderConfirm draws the surrender confirmation dialog
func (s *GameplayScene) drawSurrenderConfirm(screen *ebiten.Image) {
	// Semi-transparent overlay (less opaque so map is visible)
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 120}, false)

	// Panel
	panelW := 450
	panelH := 200
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Confirm Surrender")

	// Warning text
	y := panelY + 50
	DrawTextCentered(screen, fmt.Sprintf("Surrender to %s?", s.surrenderTargetName), ScreenWidth/2, y, ColorWarning)
	y += 30
	DrawTextCentered(screen, "All your territories and resources", ScreenWidth/2, y, ColorText)
	y += 20
	DrawTextCentered(screen, "will be given to them.", ScreenWidth/2, y, ColorText)
	y += 25
	DrawTextCentered(screen, "You may continue watching the game.", ScreenWidth/2, y, ColorTextMuted)

	// Buttons
	btnY := panelY + panelH - 55
	btnWidth := 140
	btnGap := 20
	totalBtnsWidth := btnWidth*2 + btnGap
	btnStartX := panelX + (panelW-totalBtnsWidth)/2

	s.confirmSurrenderBtn.X = btnStartX
	s.confirmSurrenderBtn.Y = btnY
	s.confirmSurrenderBtn.W = btnWidth
	s.confirmSurrenderBtn.Draw(screen)

	s.cancelSurrenderBtn.X = btnStartX + btnWidth + btnGap
	s.cancelSurrenderBtn.Y = btnY
	s.cancelSurrenderBtn.W = btnWidth
	s.cancelSurrenderBtn.Draw(screen)
}

// executeSurrender sends the surrender request to the server
func (s *GameplayScene) executeSurrender() {
	if s.surrenderTargetID == "" {
		return
	}
	log.Printf("Surrendering to player %s", s.surrenderTargetID)
	s.game.Surrender(s.surrenderTargetID)
	s.showSurrenderConfirm = false
	s.surrenderTargetID = ""
	s.surrenderTargetName = ""
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

	// Show as bottom bar medium interaction instead of modal
	text := fmt.Sprintf("%s is attacking %s -- Your adjacent strength: %d",
		payload.AttackerName, payload.TerritoryName, payload.YourStrength)
	subtext := "Join the defense?"

	defendBtn := &Button{
		W: 140, H: 35,
		Text:    "Yes, Defend",
		Primary: true,
		OnClick: func() {
			s.voteAlliance("defender")
			s.clearBottomBarMedium()
		},
	}
	neutralBtn := &Button{
		W: 140, H: 35,
		Text: "Stay Neutral",
		OnClick: func() {
			s.voteAlliance("neutral")
			s.clearBottomBarMedium()
		},
	}
	s.showBottomBarMedium("alliance_request", text, subtext, []*Button{defendBtn, neutralBtn})

	// Highlight the territory under attack
	s.SetHighlightedTerritories([]TerritoryHighlight{
		{TerritoryID: payload.TerritoryID, Color: color.RGBA{255, 200, 50, 255}},
	})
}

// voteAlliance sends the alliance vote to the server
func (s *GameplayScene) voteAlliance(side string) {
	if s.allyRequest != nil {
		log.Printf("Voting %s for battle %s", side, s.allyRequest.BattleID)
		s.game.AllianceVote(s.allyRequest.BattleID, side)
	}
	s.showAllyRequest = false
	s.allyRequest = nil
	s.ClearHighlightedTerritories()
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

// showNextPhaseSkip displays the next queued phase skip as a bottom bar notification.
func (s *GameplayScene) showNextPhaseSkip() {
	// Send acknowledgment for the current skip before moving to next
	if s.phaseSkipEventID != "" {
		s.game.SendClientReady(s.phaseSkipEventID, protocol.EventPhaseSkip)
		s.phaseSkipEventID = ""
	}

	if len(s.phaseSkipQueue) == 0 {
		return
	}

	// Pop from queue
	skip := s.phaseSkipQueue[0]
	s.phaseSkipQueue = s.phaseSkipQueue[1:]

	s.phaseSkipEventID = skip.EventID
	s.phaseSkipPhase = skip.Phase
	s.phaseSkipReason = skip.Reason

	msg := fmt.Sprintf("%s was skipped: %s", skip.Phase, skip.Reason)
	s.showBottomBarNotification(msg, "OK", func() {
		s.showNextPhaseSkip()
	})
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
	panelW, panelH := 560, 420
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

	// Note about horse selection (if offering horses)
	if s.tradeOfferHorses > 0 {
		y += 50
		DrawText(screen, "(Horse territories selected on map after clicking Send)", panelX+20, y, ColorTextMuted)
	}

	// Buttons
	canSend := s.tradeTargetPlayer != "" &&
		(s.tradeOfferCoal > 0 || s.tradeOfferGold > 0 || s.tradeOfferIron > 0 ||
			s.tradeOfferTimber > 0 || s.tradeOfferHorses > 0) &&
		(s.tradeRequestCoal > 0 || s.tradeRequestGold > 0 || s.tradeRequestIron > 0 ||
			s.tradeRequestTimber > 0 || s.tradeRequestHorses > 0)

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

	needsHorseDest := s.tradeProposal.OfferHorses > 0
	panelH := 320
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

	// Note about horse selection (if receiving horses)
	if needsHorseDest {
		y += 40
		DrawText(screen, "(Horse destinations selected on map after clicking Accept)", panelX+20, y, ColorTextMuted)
	}

	// Buttons - always allow clicking Accept, will prompt for horse selection after
	canAccept := true

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
		s.showCombatResultAsNotification()
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
		s.showCombatResultAsNotification()
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

	// Show the combat result as a bottom bar notification
	s.combatResult = s.stockpileCaptureData.PendingCombatData
	s.showCombatResultAsNotification()

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

// ==================== Attack Confirmation ====================

// ShowAttackConfirmation displays the confirmation in the expanded bottom bar with resolved alliance totals.
func (s *GameplayScene) ShowAttackConfirmation(payload *protocol.AttackPlanResolvedPayload) {
	s.showWaitingForAlliance = false
	s.bottomBarNotification = "" // Clear waiting notification
	s.attackPlanResolved = payload
	s.showAttackConfirmation = true
	s.SetBarHeight(200)
	log.Printf("Showing attack confirmation: plan %s", payload.PlanID)

	// Highlight the target territory
	if payload.TargetTerritory != "" {
		s.SetHighlightedTerritories([]TerritoryHighlight{
			{TerritoryID: payload.TargetTerritory, Color: color.RGBA{255, 80, 80, 255}},
		})
	}
}

// drawWaitingForAlliance draws the waiting overlay while alliances are being resolved.
func (s *GameplayScene) drawWaitingForAlliance(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 200}, false)

	// Simple centered message
	panelW, panelH := 350, 120
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Planning Attack")

	DrawTextCentered(screen, "Waiting for alliance decisions...", ScreenWidth/2, panelY+55, ColorText)
	DrawTextCentered(screen, "(This may take up to 60 seconds)", ScreenWidth/2, panelY+80, ColorTextMuted)
}

// drawAttackConfirmation draws the confirmation inside the expanded bottom bar.
func (s *GameplayScene) drawAttackConfirmation(screen *ebiten.Image) {
	if s.attackPlanResolved == nil {
		return
	}

	barX := 10
	barY := s.currentBarTop
	barW := ScreenWidth - 20

	// Get target name
	targetName := s.attackPlanResolved.TargetTerritory
	if terr, ok := s.territories[s.attackPlanResolved.TargetTerritory].(map[string]interface{}); ok {
		if name, ok := terr["name"].(string); ok {
			targetName = name
		}
	}

	// Title
	DrawLargeText(screen, "Confirm Attack: "+targetName, barX+20, barY+12, ColorText)

	// Three columns: Attack | Defense | Buttons
	col1X := barX + 20
	col2X := barX + barW/3 + 20
	col3X := barX + barW - 180
	y := barY + 45

	// Attack side breakdown (left column)
	DrawText(screen, "ATTACK FORCES:", col1X, y, ColorSuccess)
	attackY := y + 22

	totalAttack := s.attackPlanResolved.BaseAttackStrength + s.attackPlanResolved.AttackerAllyStrength
	DrawText(screen, fmt.Sprintf("  Your forces: %d", s.attackPlanResolved.BaseAttackStrength), col1X, attackY, ColorText)
	attackY += 20

	if s.attackPlanResolved.AttackerAllyStrength > 0 {
		allyNames := strings.Join(s.attackPlanResolved.AttackerAllyNames, ", ")
		DrawText(screen, fmt.Sprintf("  Allies: +%d (%s)", s.attackPlanResolved.AttackerAllyStrength, allyNames), col1X, attackY, ColorText)
		attackY += 20
	} else {
		DrawText(screen, "  Allies: none", col1X, attackY, ColorTextMuted)
		attackY += 20
	}

	DrawText(screen, fmt.Sprintf("  Total: %d", totalAttack), col1X, attackY, ColorSuccess)

	// Defense side breakdown (center column)
	DrawText(screen, "DEFENSE FORCES:", col2X, y, ColorDanger)
	defenseY := y + 22

	totalDefense := s.attackPlanResolved.BaseDefenseStrength + s.attackPlanResolved.DefenderAllyStrength
	DrawText(screen, fmt.Sprintf("  Base defense: %d", s.attackPlanResolved.BaseDefenseStrength), col2X, defenseY, ColorText)
	defenseY += 20

	if s.attackPlanResolved.DefenderAllyStrength > 0 {
		allyNames := strings.Join(s.attackPlanResolved.DefenderAllyNames, ", ")
		DrawText(screen, fmt.Sprintf("  Allies: +%d (%s)", s.attackPlanResolved.DefenderAllyStrength, allyNames), col2X, defenseY, ColorText)
		defenseY += 20
	} else {
		DrawText(screen, "  Allies: none", col2X, defenseY, ColorTextMuted)
		defenseY += 20
	}

	DrawText(screen, fmt.Sprintf("  Total: %d", totalDefense), col2X, defenseY, ColorDanger)

	// Buttons (right column, stacked vertically)
	btnWidth := 150
	s.confirmAttackBtn.X = col3X
	s.confirmAttackBtn.Y = barY + 50
	s.confirmAttackBtn.W = btnWidth
	s.confirmAttackBtn.Draw(screen)

	s.cancelConfirmBtn.X = col3X
	s.cancelConfirmBtn.Y = barY + 95
	s.cancelConfirmBtn.W = btnWidth
	s.cancelConfirmBtn.Draw(screen)

	// Card selection info & keyboard hint
	if s.combatMode == "cards" {
		selectedCount := len(s.selectedCardIDs)
		if selectedCount > 0 {
			DrawText(screen, fmt.Sprintf("%d attack card(s) selected", selectedCount), col3X, barY+140, ColorSuccess)
		} else if len(s.myAttackCards) > 0 {
			DrawText(screen, "No attack cards selected", col3X, barY+140, ColorTextDim)
		}
	}
	DrawText(screen, "Enter to confirm, Esc to cancel", barX+20, barY+170, ColorTextDim)
}

// confirmAttack executes the attack using the cached plan.
func (s *GameplayScene) confirmAttack() {
	if s.attackPlanResolved == nil {
		return
	}

	log.Printf("Confirming attack with plan %s", s.attackPlanResolved.PlanID)

	// Build reinforcement info from the stored selection
	var reinforcement *ReinforcementInfo
	if s.selectedReinforcement != nil {
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

	// Execute attack -- use cards pre-selected during attack planning
	if s.combatMode == "cards" {
		cardIDs := make([]string, 0)
		for id := range s.selectedCardIDs {
			cardIDs = append(cardIDs, id)
		}
		log.Printf("Executing attack with %d pre-selected cards", len(cardIDs))
		s.game.ExecuteAttackWithCards(s.attackPlanResolved.TargetTerritory, reinforcement, s.attackPlanResolved.PlanID, cardIDs)
	} else {
		s.game.ExecuteAttackWithPlan(s.attackPlanResolved.TargetTerritory, s.attackPlanResolved.PlanID, reinforcement)
	}
	// Clean up
	s.cancelAttackConfirmation()
}

// cancelAttackConfirmation cancels the attack confirmation without attacking.
func (s *GameplayScene) cancelAttackConfirmation() {
	s.showAttackConfirmation = false
	s.showWaitingForAlliance = false
	s.attackPlanResolved = nil
	s.attackPlanTarget = ""
	s.attackPreview = nil
	s.selectedReinforcement = nil
	s.selectedCardIDs = make(map[string]bool)
	s.loadHorseCheckbox = false
	s.loadWeaponCheckbox = false
	s.ClearHighlightedTerritories()
	s.ResetBarHeight()
}

// openEditTerritoryDialog opens the edit territory dialog for the given territory.
func (s *GameplayScene) openEditTerritoryDialog(territoryID string) {
	// Get current territory name and drawing
	currentName := ""
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		currentName, _ = terr["name"].(string)
	}

	s.editTerritoryID = territoryID
	s.editTerritoryInput = &TextInput{
		X: 0, Y: 0, W: 280, H: 30,
		Placeholder: "Territory name",
		Text:        currentName,
		MaxLength:   30,
	}
	// Focus the text input immediately
	s.editTerritoryInput.focused = true

	// Initialize drawing state - copy existing drawing data
	s.editTerritoryDrawing = make(map[string]int)
	s.editTerritoryDrawing0 = make(map[string]int)
	if terr, ok := s.territories[territoryID].(map[string]interface{}); ok {
		if drawing, ok := terr["drawing"].(map[string]interface{}); ok {
			for k, v := range drawing {
				if colorIdx, ok := v.(float64); ok && int(colorIdx) >= 1 && int(colorIdx) <= MaxDrawingColorIndex {
					s.editTerritoryDrawing[k] = int(colorIdx)
					s.editTerritoryDrawing0[k] = int(colorIdx)
				}
			}
		}
	}
	// Only set defaults on first open; preserve selections across dialog opens
	if s.editTerritoryTool == "" {
		s.editTerritoryTool = "pencil"
	}
	if s.editTerritoryColor == 0 {
		s.editTerritoryColor = 1
	}
	if s.editTerritoryBrushSize == 0 {
		s.editTerritoryBrushSize = 1
	}
	s.editTerritoryIsDrawing = false

	s.editTerritorySaveBtn = &Button{
		W: 120, H: 35,
		Text:    "Save",
		Primary: true,
		OnClick: func() {
			s.saveEditTerritory()
		},
	}

	s.editTerritoryCancelBtn = &Button{
		W: 120, H: 35,
		Text: "Cancel",
		OnClick: func() {
			s.showEditTerritory = false
		},
	}

	s.showEditTerritory = true
}

// saveEditTerritory saves both name and drawing changes in a single atomic message.
func (s *GameplayScene) saveEditTerritory() {
	name := strings.TrimSpace(s.editTerritoryInput.Text)
	if name == "" {
		return // Don't save empty names
	}

	// Send a single combined message with both name and drawing to avoid race conditions.
	// The server handles both atomically in one transaction.
	s.game.DrawTerritory(s.editTerritoryID, s.editTerritoryDrawing, name)

	s.showEditTerritory = false
}

// editTerritoryCanvasBounds computes the canvas area and zoom for the territory drawing dialog.
// Returns: canvasX, canvasY, canvasW, canvasH (screen coords of the canvas area),
// pixelSize (screen pixels per drawing pixel), boundMinX, boundMinY, boundMaxX, boundMaxY (grid cell bounds),
// territoryCellSet (set of "gx,gy" strings for cells in this territory).
func (s *GameplayScene) editTerritoryCanvasBounds(panelX, panelY, panelW, panelH int) (
	canvasX, canvasY, canvasW, canvasH int,
	pixelSize float32,
	boundMinX, boundMinY, boundMaxX, boundMaxY int,
	territoryCellSet map[string]bool,
) {
	// Available canvas area inside the panel
	canvasMargin := 15
	toolbarW := 110
	canvasX = panelX + canvasMargin
	canvasY = panelY + 38 // Below title bar
	canvasW = panelW - canvasMargin*2 - toolbarW
	canvasH = panelH - 38 - 90 // Leave room for name input + buttons at bottom

	// Find all cells of this territory
	grid := s.mapData["grid"].([]interface{})
	territoryCellSet = make(map[string]bool)

	var numID int
	if len(s.editTerritoryID) > 1 && s.editTerritoryID[0] == 't' {
		fmt.Sscanf(s.editTerritoryID[1:], "%d", &numID)
	}

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	boundMinX = width
	boundMinY = height
	boundMaxX = 0
	boundMaxY = 0

	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			if int(row[x].(float64)) == numID {
				key := fmt.Sprintf("%d,%d", x, y)
				territoryCellSet[key] = true
				if x < boundMinX {
					boundMinX = x
				}
				if x > boundMaxX {
					boundMaxX = x
				}
				if y < boundMinY {
					boundMinY = y
				}
				if y > boundMaxY {
					boundMaxY = y
				}
			}
		}
	}

	// Add 1-cell padding around bounds
	if boundMinX > 0 {
		boundMinX--
	}
	if boundMinY > 0 {
		boundMinY--
	}
	if boundMaxX < width-1 {
		boundMaxX++
	}
	if boundMaxY < height-1 {
		boundMaxY++
	}

	// Calculate pixel size to fit the bounding box in the canvas
	drawW := (boundMaxX - boundMinX + 1) * game.DrawingSubPixels // Drawing pixels wide
	drawH := (boundMaxY - boundMinY + 1) * game.DrawingSubPixels // Drawing pixels tall

	if drawW > 0 && drawH > 0 {
		pxW := float32(canvasW) / float32(drawW)
		pxH := float32(canvasH) / float32(drawH)
		pixelSize = pxW
		if pxH < pxW {
			pixelSize = pxH
		}
		// Clamp minimum and maximum pixel size
		if pixelSize < 1 {
			pixelSize = 1
		}
		if pixelSize > 10 {
			pixelSize = 10
		}
	} else {
		pixelSize = 4
	}

	return
}

// updateEditTerritory handles input for the edit territory dialog.
func (s *GameplayScene) updateEditTerritory() {
	s.editTerritoryInput.Update()
	s.editTerritorySaveBtn.Update()
	s.editTerritoryCancelBtn.Update()

	// ESC to cancel
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.showEditTerritory = false
		return
	}

	// Handle drawing interaction on canvas
	panelW := 700
	panelH := 550
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	canvasX, canvasY, canvasW, canvasH, pixelSize, boundMinX, boundMinY, boundMaxX, boundMaxY, territoryCellSet :=
		s.editTerritoryCanvasBounds(panelX, panelY, panelW, panelH)

	// Calculate actual rendered size and center it (must match drawEditTerritory exactly)
	sp := game.DrawingSubPixels
	drawPixelsW := (boundMaxX - boundMinX + 1) * sp
	drawPixelsH := (boundMaxY - boundMinY + 1) * sp
	actualCanvasW := int(float32(drawPixelsW) * pixelSize)
	actualCanvasH := int(float32(drawPixelsH) * pixelSize)
	offsetX := canvasX + (canvasW-actualCanvasW)/2
	offsetY := canvasY + (canvasH-actualCanvasH)/2

	mx, my := ebiten.CursorPosition()

	// Check if mouse is within canvas area
	inCanvas := mx >= offsetX && mx < offsetX+actualCanvasW &&
		my >= offsetY && my < offsetY+actualCanvasH

	// Handle mouse button for drawing
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && inCanvas {
		// Convert screen position to drawing pixel coordinates
		relX := float32(mx-offsetX) / pixelSize
		relY := float32(my-offsetY) / pixelSize
		centerX := int(relX) + boundMinX*sp
		centerY := int(relY) + boundMinY*sp

		// Apply brush to all pixels in the brush area
		brushRadius := s.editTerritoryBrushSize - 1 // size 1 = single pixel, size 2 = 3x3 area centered, etc.
		for dy := -brushRadius; dy <= brushRadius; dy++ {
			for dx := -brushRadius; dx <= brushRadius; dx++ {
				drawX := centerX + dx
				drawY := centerY + dy

				// Check if this pixel is within a territory cell
				gridX := drawX / sp
				gridY := drawY / sp
				cellKey := fmt.Sprintf("%d,%d", gridX, gridY)
				if territoryCellSet[cellKey] {
					pixelKey := fmt.Sprintf("%d,%d", drawX, drawY)
					if s.editTerritoryTool == "pencil" {
						s.editTerritoryDrawing[pixelKey] = s.editTerritoryColor
					} else if s.editTerritoryTool == "eraser" {
						delete(s.editTerritoryDrawing, pixelKey)
					}
				}
			}
		}
		s.editTerritoryIsDrawing = true
	} else {
		s.editTerritoryIsDrawing = false
	}

	// Handle tool selection via toolbar clicks
	toolbarX := panelX + panelW - 110 - 10
	toolbarY := panelY + 42

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Pencil button
		if mx >= toolbarX && mx < toolbarX+100 && my >= toolbarY && my < toolbarY+30 {
			s.editTerritoryTool = "pencil"
		}
		// Eraser button
		if mx >= toolbarX && mx < toolbarX+100 && my >= toolbarY+35 && my < toolbarY+65 {
			s.editTerritoryTool = "eraser"
		}

		// Brush size buttons - 4 sizes in a row
		sizeY := toolbarY + 75
		for i := 0; i < 4; i++ {
			sx := toolbarX + i*25
			if mx >= sx && mx < sx+22 && my >= sizeY && my < sizeY+22 {
				s.editTerritoryBrushSize = i + 1
			}
		}

		// Color palette - 2 columns of 5
		colorStartY := toolbarY + 115
		for i, colorIdx := range DrawingColorOrder {
			col := i % 2
			row := i / 2
			cx := toolbarX + col*50
			cy := colorStartY + row*30
			if mx >= cx && mx < cx+44 && my >= cy && my < cy+24 {
				s.editTerritoryColor = colorIdx
				s.editTerritoryTool = "pencil" // Selecting a color switches to pencil
			}
		}
	}
}

// drawEditTerritory draws the edit territory dialog overlay.
func (s *GameplayScene) drawEditTerritory(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 200}, false)

	// Dialog panel - larger to accommodate drawing canvas
	panelW := 700
	panelH := 550
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Edit Territory")

	// Compute canvas bounds
	canvasX, canvasY, canvasW, canvasH, pixelSize, boundMinX, boundMinY, boundMaxX, boundMaxY, territoryCellSet :=
		s.editTerritoryCanvasBounds(panelX, panelY, panelW, panelH)

	// Calculate actual rendered size and center it
	sp := game.DrawingSubPixels
	drawPixelsW := (boundMaxX - boundMinX + 1) * sp
	drawPixelsH := (boundMaxY - boundMinY + 1) * sp
	actualCanvasW := float32(drawPixelsW) * pixelSize
	actualCanvasH := float32(drawPixelsH) * pixelSize
	offsetX := float32(canvasX) + (float32(canvasW)-actualCanvasW)/2
	offsetY := float32(canvasY) + (float32(canvasH)-actualCanvasH)/2

	// Draw canvas background
	vector.DrawFilledRect(screen, float32(canvasX), float32(canvasY),
		float32(canvasW), float32(canvasH), color.RGBA{10, 10, 30, 255}, false)
	vector.StrokeRect(screen, float32(canvasX), float32(canvasY),
		float32(canvasW), float32(canvasH), 1, ColorBorderDark, false)

	// Draw the territory cells zoomed in
	grid := s.mapData["grid"].([]interface{})
	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	for gy := boundMinY; gy <= boundMaxY; gy++ {
		if gy < 0 || gy >= height {
			continue
		}
		row := grid[gy].([]interface{})
		for gx := boundMinX; gx <= boundMaxX; gx++ {
			if gx < 0 || gx >= width {
				continue
			}
			cellKey := fmt.Sprintf("%d,%d", gx, gy)
			isTerrCell := territoryCellSet[cellKey]

			cellScreenX := offsetX + float32((gx-boundMinX)*sp)*pixelSize
			cellScreenY := offsetY + float32((gy-boundMinY)*sp)*pixelSize
			cellScreenSize := pixelSize * float32(sp)

			// Determine base cell color
			var cellColor color.RGBA
			territoryID := int(row[gx].(float64))
			if territoryID == 0 {
				// Water
				cellColor = color.RGBA{15, 45, 90, 255}
			} else if isTerrCell {
				// This territory - use owner color
				tid := fmt.Sprintf("t%d", territoryID)
				if terr, ok := s.territories[tid].(map[string]interface{}); ok {
					owner, _ := terr["owner"].(string)
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
						cellColor = color.RGBA{100, 100, 100, 255}
					}
				} else {
					cellColor = color.RGBA{100, 100, 100, 255}
				}
			} else {
				// Neighboring territory or land - dimmed
				cellColor = color.RGBA{40, 40, 50, 255}
			}

			// Draw the cell base color
			vector.DrawFilledRect(screen, cellScreenX, cellScreenY, cellScreenSize, cellScreenSize, cellColor, false)

			// Draw existing drawing pixels for this cell
			if isTerrCell {
				for subY := 0; subY < sp; subY++ {
					for subX := 0; subX < sp; subX++ {
						drawKey := fmt.Sprintf("%d,%d", gx*sp+subX, gy*sp+subY)
						if colorIdx, ok := s.editTerritoryDrawing[drawKey]; ok {
							if dc, ok := DrawingColors[colorIdx]; ok {
								px := cellScreenX + float32(subX)*pixelSize
								py := cellScreenY + float32(subY)*pixelSize
								vector.DrawFilledRect(screen, px, py, pixelSize, pixelSize, dc, false)
							}
						}
					}
				}
			}

			// Draw cell border (thicker) between different territories
			borderColor := color.RGBA{0, 0, 0, 200}
			if gx+1 <= boundMaxX && gx+1 < width {
				nextRow := grid[gy].([]interface{})
				nextID := int(nextRow[gx+1].(float64))
				if nextID != territoryID {
					bx := cellScreenX + cellScreenSize
					vector.StrokeLine(screen, bx, cellScreenY, bx, cellScreenY+cellScreenSize, 2, borderColor, false)
				}
			}
			if gy+1 <= boundMaxY && gy+1 < height {
				nextRow := grid[gy+1].([]interface{})
				nextID := int(nextRow[gx].(float64))
				if nextID != territoryID {
					by := cellScreenY + cellScreenSize
					vector.StrokeLine(screen, cellScreenX, by, cellScreenX+cellScreenSize, by, 2, borderColor, false)
				}
			}
		}
	}

	// Draw toolbar area on the right
	toolbarX := panelX + panelW - 110 - 10
	toolbarY := panelY + 42

	// Pencil tool button
	pencilBg := color.RGBA{30, 30, 60, 255}
	if s.editTerritoryTool == "pencil" {
		pencilBg = color.RGBA{60, 60, 120, 255}
	}
	vector.DrawFilledRect(screen, float32(toolbarX), float32(toolbarY), 100, 30, pencilBg, false)
	vector.StrokeRect(screen, float32(toolbarX), float32(toolbarY), 100, 30, 1, ColorBorder, false)
	DrawText(screen, "Pencil", toolbarX+25, toolbarY+8, ColorText)

	// Eraser tool button
	eraserBg := color.RGBA{30, 30, 60, 255}
	if s.editTerritoryTool == "eraser" {
		eraserBg = color.RGBA{60, 60, 120, 255}
	}
	vector.DrawFilledRect(screen, float32(toolbarX), float32(toolbarY+35), 100, 30, eraserBg, false)
	vector.StrokeRect(screen, float32(toolbarX), float32(toolbarY+35), 100, 30, 1, ColorBorder, false)
	DrawText(screen, "Eraser", toolbarX+25, toolbarY+43, ColorText)

	// Brush size selector - 4 sizes in a row
	sizeY := toolbarY + 75
	for i := 0; i < 4; i++ {
		sx := float32(toolbarX + i*25)
		sy := float32(sizeY)
		sizeBg := color.RGBA{30, 30, 60, 255}
		if s.editTerritoryBrushSize == i+1 {
			sizeBg = color.RGBA{60, 60, 120, 255}
		}
		vector.DrawFilledRect(screen, sx, sy, 22, 22, sizeBg, false)
		vector.StrokeRect(screen, sx, sy, 22, 22, 1, ColorBorder, false)

		// Draw a dot that represents the brush size
		dotSize := float32(2 + i*2)
		dotX := sx + (22-dotSize)/2
		dotY := sy + (22-dotSize)/2
		vector.DrawFilledRect(screen, dotX, dotY, dotSize, dotSize, ColorText, false)
	}

	// Color palette - 2 columns of 5
	colorStartY := toolbarY + 115
	for i, colorIdx := range DrawingColorOrder {
		col := i % 2
		row := i / 2
		cx := float32(toolbarX + col*50)
		cy := float32(colorStartY + row*30)

		dc := DrawingColors[colorIdx]
		vector.DrawFilledRect(screen, cx, cy, 44, 24, dc, false)

		// Highlight selected color
		if s.editTerritoryColor == colorIdx && s.editTerritoryTool == "pencil" {
			vector.StrokeRect(screen, cx-1, cy-1, 46, 26, 2, color.RGBA{255, 255, 255, 255}, false)
		} else {
			vector.StrokeRect(screen, cx, cy, 44, 24, 1, color.RGBA{0, 0, 0, 200}, false)
		}
	}

	// Name input area
	nameY := panelY + panelH - 85
	DrawText(screen, "Name:", panelX+20, nameY, ColorTextMuted)
	s.editTerritoryInput.X = panelX + 70
	s.editTerritoryInput.Y = nameY - 3
	s.editTerritoryInput.W = panelW - 100
	s.editTerritoryInput.Draw(screen)

	// Buttons
	btnY := panelY + panelH - 50
	s.editTerritorySaveBtn.X = panelX + panelW/2 - 130
	s.editTerritorySaveBtn.Y = btnY
	s.editTerritorySaveBtn.Draw(screen)

	s.editTerritoryCancelBtn.X = panelX + panelW/2 + 10
	s.editTerritoryCancelBtn.Y = btnY
	s.editTerritoryCancelBtn.Draw(screen)
}

// ==================== Card Combat UI ====================

// commitAttackCards sends the selected attack cards and executes the attack.
func (s *GameplayScene) commitAttackCards() {
	if s.attackPlanResolved == nil {
		s.showAttackCardSelect = false
		return
	}

	// Collect selected card IDs
	cardIDs := make([]string, 0)
	for id, selected := range s.selectedAttackCardIDs {
		if selected {
			cardIDs = append(cardIDs, id)
		}
	}

	// Build reinforcement info
	var reinforcement *ReinforcementInfo
	if s.selectedReinforcement != nil {
		reinforcement = &ReinforcementInfo{
			UnitType:      s.selectedReinforcement.UnitType,
			FromTerritory: s.selectedReinforcement.FromTerritory,
			WaterBodyID:   s.selectedReinforcement.WaterBodyID,
		}
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
		if s.selectedReinforcement.UnitType == "horse" {
			if s.loadWeaponCheckbox && s.selectedReinforcement.CanCarryWeapon {
				reinforcement.CarryWeapon = true
				reinforcement.WeaponFrom = s.selectedReinforcement.FromTerritory
			}
		}
	}

	log.Printf("Committing attack with %d cards", len(cardIDs))
	s.game.ExecuteAttackWithCards(s.attackPlanResolved.TargetTerritory, reinforcement, s.attackPlanResolved.PlanID, cardIDs)

	// Clean up
	s.showAttackCardSelect = false
	s.selectedAttackCardIDs = make(map[string]bool)
	s.cancelAttackConfirmation()
}

// commitDefenseCards sends the selected defense cards to the server.
func (s *GameplayScene) commitDefenseCards() {
	cardIDs := make([]string, 0)
	for id, selected := range s.selectedDefenseCardIDs {
		if selected {
			cardIDs = append(cardIDs, id)
		}
	}

	log.Printf("Committing defense with %d cards", len(cardIDs))
	s.game.SelectDefenseCards(cardIDs)

	s.showDefenseCardSelect = false
	s.selectedDefenseCardIDs = make(map[string]bool)
}

// ShowDefenseCardRequest shows the defense card selection dialog.
func (s *GameplayScene) ShowDefenseCardRequest(battleID, attackerName, terrName string, atkCardCount, baseAtkStr, baseDefStr int) {
	s.defenseCardBattleID = battleID
	s.defenseCardAttackerName = attackerName
	s.defenseCardTerrName = terrName
	s.defenseCardAtkCount = atkCardCount
	s.defenseCardAtkStr = baseAtkStr
	s.defenseCardDefStr = baseDefStr

	// Use card hand selection mode instead of modal dialog
	contextMsg := fmt.Sprintf("%s attacking %s! Atk %d vs Def %d -- Select defense cards:", attackerName, terrName, baseAtkStr, baseDefStr)
	s.enterCardSelectionMode("defense", contextMsg)

	// Highlight territory under attack (find territory ID from name)
	for tid, tdata := range s.territories {
		if terr, ok := tdata.(map[string]interface{}); ok {
			if name, ok := terr["name"].(string); ok && name == terrName {
				s.SetHighlightedTerritories([]TerritoryHighlight{
					{TerritoryID: tid, Color: color.RGBA{255, 200, 50, 255}},
				})
				break
			}
		}
	}
}

// ShowCardDrawn is called when a card is purchased. The card appears in the
// player's hand automatically, so no notification is needed.
func (s *GameplayScene) ShowCardDrawn(name, desc, rarity, cardType string) {
	// No-op: the card visually appears in the card hand
}

// ShowCardReveal shows the card reveal animation after combat.
// Skips entirely if no cards were played by either side.
func (s *GameplayScene) ShowCardReveal(data *protocol.CardRevealPayload) {
	if len(data.AttackerCards) == 0 && len(data.DefenderCards) == 0 {
		return // No cards to reveal
	}
	s.showCardReveal = true
	s.cardRevealData = data
	s.cardRevealTimer = 0
	// Territory highlighting is kept from the preceding attack plan/confirmation
}

// dismissCardReveal closes the card reveal dialog and sends ack to the server.
func (s *GameplayScene) dismissCardReveal() {
	// Send acknowledgment so the server can proceed with combat result
	if s.cardRevealData != nil && s.cardRevealData.EventID != "" {
		s.game.SendClientReady(s.cardRevealData.EventID, protocol.EventCardReveal)
	}

	s.showCardReveal = false
	s.cardRevealData = nil
	s.ResetBarHeight()
}

// getRarityColor returns the color for a card rarity.
func getRarityColor(rarity string) color.RGBA {
	switch rarity {
	case "uncommon":
		return color.RGBA{80, 140, 255, 255} // Blue
	case "rare":
		return color.RGBA{180, 80, 255, 255} // Purple
	case "ultra_rare":
		return color.RGBA{255, 200, 50, 255} // Gold
	default:
		return color.RGBA{200, 200, 200, 255} // White/Common
	}
}

// cardSelectLayout computes layout for the card selection dialogs based on card count.
func cardSelectLayout(nCards int) (cardW, cardH, cardGap, panelW int) {
	cardW = 110
	cardH = 110
	cardGap = 8
	totalCardsW := nCards*(cardW+cardGap) - cardGap
	panelW = totalCardsW + 40 // 20px padding each side
	if panelW < 400 {
		panelW = 400
	}
	return
}

// drawAttackCardSelect draws the attack card selection dialog.
func (s *GameplayScene) drawAttackCardSelect(screen *ebiten.Image) {
	if !s.showAttackCardSelect {
		return
	}

	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight), color.RGBA{0, 0, 0, 150}, false)

	cardW, cardH, cardGap, panelW := cardSelectLayout(len(s.myAttackCards))
	panelH := 300
	panelX := (ScreenWidth - panelW) / 2
	panelY := (ScreenHeight - panelH) / 2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Select Attack Cards")

	DrawText(screen, "Choose cards to play face-down (click to toggle):", panelX+20, panelY+45, ColorTextMuted)

	// Draw attack cards
	totalW := len(s.myAttackCards)*(cardW+cardGap) - cardGap
	startX := panelX + (panelW-totalW)/2
	cardY := panelY + 70

	for i, card := range s.myAttackCards {
		cx := startX + i*(cardW+cardGap)
		selected := s.selectedAttackCardIDs[card.ID]
		s.drawCardContentSel(screen, cx, cardY, cardW, cardH, card, true, selected)
	}

	// Selected count
	selectedCount := 0
	for _, v := range s.selectedAttackCardIDs {
		if v {
			selectedCount++
		}
	}
	DrawText(screen, fmt.Sprintf("%d cards selected", selectedCount), panelX+20, panelY+panelH-70, ColorText)

	// Buttons
	btnY := panelY + panelH - 50
	s.confirmAttackCardsBtn.X = panelX + panelW/2 - 150
	s.confirmAttackCardsBtn.Y = btnY
	if selectedCount > 0 {
		s.confirmAttackCardsBtn.Text = fmt.Sprintf("Play %d Cards", selectedCount)
	} else {
		s.confirmAttackCardsBtn.Text = "Play Cards"
	}
	s.confirmAttackCardsBtn.Disabled = selectedCount == 0
	s.confirmAttackCardsBtn.Draw(screen)

	s.skipAttackCardsBtn.X = panelX + panelW/2 + 10
	s.skipAttackCardsBtn.Y = btnY
	s.skipAttackCardsBtn.Draw(screen)
}

// drawDefenseCardSelect draws the defense card selection dialog.
func (s *GameplayScene) drawDefenseCardSelect(screen *ebiten.Image) {
	if !s.showDefenseCardSelect {
		return
	}

	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight), color.RGBA{0, 0, 0, 150}, false)

	cardW, cardH, cardGap, panelW := cardSelectLayout(len(s.myDefenseCards))
	panelH := 340
	panelX := (ScreenWidth - panelW) / 2
	panelY := (ScreenHeight - panelH) / 2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Defend!")

	DrawText(screen, fmt.Sprintf("%s is attacking %s!", s.defenseCardAttackerName, s.defenseCardTerrName),
		panelX+20, panelY+45, ColorDanger)
	DrawText(screen, fmt.Sprintf("Base Strength:  Attack %d  vs  Defense %d", s.defenseCardAtkStr, s.defenseCardDefStr),
		panelX+20, panelY+63, ColorText)
	DrawText(screen, fmt.Sprintf("They played %d attack card(s). Select your defense:", s.defenseCardAtkCount),
		panelX+20, panelY+78, ColorTextMuted)

	// Draw defense cards
	totalW := len(s.myDefenseCards)*(cardW+cardGap) - cardGap
	startX := panelX + (panelW-totalW)/2
	cardY := panelY + 100

	for i, card := range s.myDefenseCards {
		cx := startX + i*(cardW+cardGap)
		selected := s.selectedDefenseCardIDs[card.ID]
		s.drawCardContentSel(screen, cx, cardY, cardW, cardH, card, false, selected)
	}

	// Selected count
	selectedCount := 0
	for _, v := range s.selectedDefenseCardIDs {
		if v {
			selectedCount++
		}
	}
	DrawText(screen, fmt.Sprintf("%d cards selected", selectedCount), panelX+20, panelY+panelH-70, ColorText)

	// Buttons
	btnY := panelY + panelH - 50
	s.confirmDefenseCardsBtn.X = panelX + panelW/2 - 150
	s.confirmDefenseCardsBtn.Y = btnY
	if selectedCount > 0 {
		s.confirmDefenseCardsBtn.Text = fmt.Sprintf("Play %d Cards", selectedCount)
	} else {
		s.confirmDefenseCardsBtn.Text = "Play Cards"
	}
	s.confirmDefenseCardsBtn.Disabled = selectedCount == 0
	s.confirmDefenseCardsBtn.Draw(screen)

	s.skipDefenseCardsBtn.X = panelX + panelW/2 + 10
	s.skipDefenseCardsBtn.Y = btnY
	s.skipDefenseCardsBtn.Draw(screen)
}

// drawCardDrawnPopup draws the popup showing a newly purchased card.
func (s *GameplayScene) drawCardDrawnPopup(screen *ebiten.Image) {
	if !s.showCardDrawn {
		return
	}

	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight), color.RGBA{0, 0, 0, 120}, false)

	panelW := 300
	panelH := 200
	panelX := (ScreenWidth - panelW) / 2
	panelY := (ScreenHeight - panelH) / 2

	title := "New Attack Card!"
	if s.drawnCardType == "defense" {
		title = "New Defense Card!"
	}
	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, title)

	// Card info
	rarityCol := getRarityColor(s.drawnCardRarity)
	DrawLargeText(screen, s.drawnCardName, panelX+20, panelY+50, rarityCol)
	DrawText(screen, s.drawnCardDesc, panelX+20, panelY+80, ColorText)

	rarityLabel := s.drawnCardRarity
	if rarityLabel == "ultra_rare" {
		rarityLabel = "ULTRA RARE"
	}
	DrawText(screen, rarityLabel, panelX+20, panelY+110, rarityCol)

	// OK button
	s.dismissCardDrawnBtn.X = panelX + panelW/2 - 50
	s.dismissCardDrawnBtn.Y = panelY + panelH - 50
	s.dismissCardDrawnBtn.Draw(screen)
}

// drawCardRevealDialog draws the card reveal in the expanded bottom bar.
func (s *GameplayScene) drawCardRevealDialog(screen *ebiten.Image) {
	if !s.showCardReveal || s.cardRevealData == nil {
		return
	}

	data := s.cardRevealData

	// Expand the bottom bar for card reveal
	s.SetBarHeight(200)

	barX := 10
	barY := s.currentBarTop
	barW := ScreenWidth - 20

	// Draw bar background (already drawn by drawBottomBar, but we override content)
	DrawFancyPanel(screen, barX, barY, barW, int(s.currentBarHeight), "")

	// Title
	DrawLargeText(screen, "CARD REVEAL", barX+20, barY+15, ColorText)

	// 2-column layout: Attack Cards | Defense Cards, with totals below
	colW := (barW - 60) / 2

	// Column 1: Attack Cards
	col1X := barX + 20
	DrawText(screen, "ATTACK CARDS:", col1X, barY+42, ColorDanger)
	cardY := barY + 60
	if len(data.AttackerCards) == 0 {
		DrawText(screen, "(none)", col1X+10, cardY, ColorTextDim)
	}
	for i, c := range data.AttackerCards {
		col := getRarityColor(c.Rarity)
		negated := false
		for _, nc := range data.NegatedCards {
			if nc.ID == c.ID {
				negated = true
				break
			}
		}
		text := fmt.Sprintf("- %s: %s", c.Name, c.Description)
		if negated {
			text += " [NEGATED]"
			col = ColorTextDim
		}
		DrawText(screen, text, col1X+10, cardY+i*16, col)
	}

	// Column 2: Defense Cards
	col2X := col1X + colW + 20
	DrawText(screen, "DEFENSE CARDS:", col2X, barY+42, ColorPrimary)
	defCardY := barY + 60
	if len(data.DefenderCards) == 0 {
		DrawText(screen, "(none)", col2X+10, defCardY, ColorTextDim)
	}
	for i, c := range data.DefenderCards {
		col := getRarityColor(c.Rarity)
		negated := false
		for _, nc := range data.NegatedCards {
			if nc.ID == c.ID {
				negated = true
				break
			}
		}
		text := fmt.Sprintf("- %s: %s", c.Name, c.Description)
		if negated {
			text += " [NEGATED]"
			col = ColorTextDim
		}
		DrawText(screen, text, col2X+10, defCardY+i*16, col)
	}

	// Final totals at bottom of bar
	totalsY := barY + int(s.currentBarHeight) - 55
	if data.BribeActivated {
		DrawLargeText(screen, "BRIBE activated! Defense auto-wins", barX+20, totalsY, color.RGBA{255, 200, 50, 255})
	} else {
		DrawText(screen, fmt.Sprintf("Final Attack Strength: %d", data.FinalAttackStr), barX+20, totalsY, ColorDanger)
		DrawText(screen, fmt.Sprintf("Final Defense Strength: %d", data.FinalDefenseStr), barX+20, totalsY+18, ColorPrimary)
	}
	if data.SabotageCount > 0 {
		DrawText(screen, fmt.Sprintf("Sabotage: %d unit(s) destroyed", data.SabotageCount), barX+400, totalsY, ColorDanger)
	}
	if data.SafeRetreat {
		DrawText(screen, "Safe Retreat active", barX+400, totalsY+18, ColorPrimary)
	}

	// OK button
	s.dismissCardRevealBtn.X = barX + barW - 130
	s.dismissCardRevealBtn.Y = barY + int(s.currentBarHeight) - 50
	s.dismissCardRevealBtn.Update()
	s.dismissCardRevealBtn.Draw(screen)

	// Also dismiss on Enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.dismissCardReveal()
	}
}

// maxInt returns the larger of two ints.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// handleCardSelectionClick handles clicking on cards in the selection dialog.
func (s *GameplayScene) handleCardSelectionClick(cards []CardDisplayInfo, selectedIDs map[string]bool) {
	mx, my := ebiten.CursorPosition()

	cardW, cardH, cardGap, panelW := cardSelectLayout(len(cards))
	panelH := 300
	if len(cards) > 0 && cards[0].CardType == "defense" {
		panelH = 340
	}
	panelX := (ScreenWidth - panelW) / 2
	panelY := (ScreenHeight - panelH) / 2

	totalW := len(cards)*(cardW+cardGap) - cardGap
	startX := panelX + (panelW-totalW)/2
	cardY := panelY + 70
	if len(cards) > 0 && cards[0].CardType == "defense" {
		cardY = panelY + 100
	}

	for i, card := range cards {
		cx := startX + i*(cardW+cardGap)
		if mx >= cx && mx <= cx+cardW && my >= cardY && my <= cardY+cardH {
			// Toggle selection
			selectedIDs[card.ID] = !selectedIDs[card.ID]
			break
		}
	}
}

// cardHandLayout returns shared layout constants for the card hand display.
// barY tracks the current bottom bar top so cards move with the bar.
func (s *GameplayScene) cardHandLayout() (barY, cardW, cardH, cardStep, peekH, sidebarEnd int) {
	w := 110
	gap := 15
	// barY is the TOP of the bottom bar -- cards tuck under it
	// cardH must be <= barHeight so cards don't poke below the bar
	return s.currentBarTop, w, 100, w + gap, 30, 300
}

// cardHandPos returns the x position for a card at index i (attack or defense).
func (s *GameplayScene) cardHandPos(i int, isAttack bool) int {
	_, cardW, _, cardStep, _, sidebarEnd := s.cardHandLayout()
	if isAttack {
		return sidebarEnd + i*cardStep
	}
	// Defense cards: rightmost card at index 0 is flush right, each subsequent card to the left
	return ScreenWidth - 15 - cardW - i*cardStep
}

// drawCardContent draws a single card's full content at the given position.
func (s *GameplayScene) drawCardContent(screen *ebiten.Image, cx, cy, cardW, cardH int, card CardDisplayInfo, isAttack bool) {
	s.drawCardContentSel(screen, cx, cy, cardW, cardH, card, isAttack, false)
}

// drawCardContentSel draws a card with an optional selection highlight.
func (s *GameplayScene) drawCardContentSel(screen *ebiten.Image, cx, cy, cardW, cardH int, card CardDisplayInfo, isAttack, selected bool) {
	rarityCol := getRarityColor(card.Rarity)

	// Card background
	bgColor := color.RGBA{35, 25, 30, 245}
	if !isAttack {
		bgColor = color.RGBA{25, 25, 40, 245}
	}
	if selected {
		if isAttack {
			bgColor = color.RGBA{50, 35, 30, 245}
		} else {
			bgColor = color.RGBA{30, 40, 55, 245}
		}
	}
	vector.DrawFilledRect(screen, float32(cx), float32(cy), float32(cardW), float32(cardH), bgColor, false)

	// Border - matching game panel style: outer bright + inner dark
	outerBorder := rarityCol
	if selected {
		outerBorder = ColorSuccess
	}
	vector.StrokeRect(screen, float32(cx), float32(cy), float32(cardW), float32(cardH), 2, outerBorder, false)
	vector.StrokeRect(screen, float32(cx+2), float32(cy+2), float32(cardW-4), float32(cardH-4), 1, ColorBorderDark, false)

	// Selection check mark
	if selected {
		DrawText(screen, "[X]", cx+cardW-22, cy+5, ColorSuccess)
	}

	// Title (bold = draw twice offset by 1px)
	DrawText(screen, card.Name, cx+6, cy+6, rarityCol)
	DrawText(screen, card.Name, cx+7, cy+6, rarityCol)

	// Separator line under title
	vector.StrokeLine(screen, float32(cx+5), float32(cy+20), float32(cx+cardW-5), float32(cy+20), 1, ColorBorderDark, false)

	// Word-wrap description
	desc := card.Description
	lineY := cy + 25
	maxChars := (cardW - 14) / 6
	for len(desc) > 0 && lineY < cy+cardH-22 {
		line := desc
		if len(line) > maxChars {
			cut := maxChars
			for cut > 0 && line[cut] != ' ' {
				cut--
			}
			if cut == 0 {
				cut = maxChars
			}
			line = desc[:cut]
			desc = desc[cut:]
			if len(desc) > 0 && desc[0] == ' ' {
				desc = desc[1:]
			}
		} else {
			desc = ""
		}
		DrawText(screen, line, cx+7, lineY, ColorText)
		lineY += 14
	}

	// Bottom row: type label and rarity
	if isAttack {
		DrawText(screen, "ATK", cx+6, cy+cardH-16, ColorDanger)
	} else {
		DrawText(screen, "DEF", cx+6, cy+cardH-16, ColorPrimary)
	}
	rarityLabel := card.Rarity
	if rarityLabel == "ultra_rare" {
		rarityLabel = "ULTRA"
	}
	DrawText(screen, rarityLabel, cx+cardW-6-int(MeasureText(rarityLabel, FontSizeBody)), cy+cardH-16, rarityCol)
}

// updateCardHoverAnimation updates the card hover state and animation each frame.
func (s *GameplayScene) updateCardHoverAnimation() {
	if s.combatMode != "cards" {
		return
	}

	barY, cardW, cardH, cardStep, peekH, sidebarEnd := s.cardHandLayout()
	mx, my := ebiten.CursorPosition()

	// Calculate current card top based on animation progress
	fullTop := barY - cardH
	currentCardTop := barY - peekH - int(float64(cardH-peekH)*s.cardHoverProgress)
	if currentCardTop < fullTop {
		currentCardTop = fullTop
	}

	// Determine which card is hovered
	newHoveredIdx := -1
	newHoveredIsAtk := false

	// First check the currently active (raised/animating) card since it's drawn on top
	activeIdx, activeIsAtk := s.getActiveCardIdx()
	if activeIdx >= 0 {
		var cx int
		if activeIsAtk && activeIdx < len(s.myAttackCards) {
			cx = sidebarEnd + activeIdx*cardStep
		} else if !activeIsAtk && activeIdx < len(s.myDefenseCards) {
			cx = ScreenWidth - 15 - cardW - activeIdx*cardStep
		}
		if mx >= cx && mx <= cx+cardW && my >= currentCardTop && my < barY {
			newHoveredIdx = activeIdx
			newHoveredIsAtk = activeIsAtk
		}
	}

	// Check attack cards - iterate from top (highest index drawn last = on top)
	// Attack cards are hidden during defense card selection
	hideAttackCards := s.cardSelectionMode == "defense"
	if newHoveredIdx == -1 && !hideAttackCards {
		for i := len(s.myAttackCards) - 1; i >= 0; i-- {
			cx := sidebarEnd + i*cardStep
			// For selected cards, check the full raised area too
			topY := barY - peekH
			if s.isCardSelected(s.myAttackCards[i].ID) {
				topY = fullTop
			}
			if mx >= cx && mx <= cx+cardW && my >= topY && my < barY {
				newHoveredIdx = i
				newHoveredIsAtk = true
				break
			}
		}
	}

	// Check defense cards (only if no card hovered yet)
	// Defense cards are hidden during attack planning/confirmation
	hideDefenseCards := s.showAttackPlan || s.showAttackConfirmation
	if newHoveredIdx == -1 && !hideDefenseCards {
		// Defense cards: index 0 is rightmost (drawn last = on top)
		for i := 0; i < len(s.myDefenseCards); i++ {
			cx := ScreenWidth - 15 - cardW - i*cardStep
			topY := barY - peekH
			if s.isCardSelected(s.myDefenseCards[i].ID) {
				topY = fullTop
			}
			if mx >= cx && mx <= cx+cardW && my >= topY && my < barY {
				newHoveredIdx = i
				newHoveredIsAtk = false
				break
			}
		}
	}

	// Handle click to toggle card selection:
	// - In explicit card selection mode (defense cards after being attacked)
	// - During attack planning (attack cards can be pre-selected)
	allowCardToggle := s.cardSelectionMode != "" || s.showAttackPlan
	if allowCardToggle && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if newHoveredIdx >= 0 {
			var cardID string
			if newHoveredIsAtk && newHoveredIdx < len(s.myAttackCards) {
				cardID = s.myAttackCards[newHoveredIdx].ID
			} else if !newHoveredIsAtk && newHoveredIdx < len(s.myDefenseCards) {
				cardID = s.myDefenseCards[newHoveredIdx].ID
			}
			if cardID != "" {
				// Determine valid selection based on context
				validSelection := false
				if s.showAttackPlan {
					// During attack planning, only attack cards
					validSelection = newHoveredIsAtk
				} else {
					validSelection = (s.cardSelectionMode == "attack" && newHoveredIsAtk) ||
						(s.cardSelectionMode == "defense" && !newHoveredIsAtk)
				}
				if validSelection {
					if s.selectedCardIDs[cardID] {
						delete(s.selectedCardIDs, cardID)
					} else {
						s.selectedCardIDs[cardID] = true
					}
				}
			}
		}
	}

	// Track hover target changes
	if newHoveredIdx >= 0 {
		// New card hovered
		if newHoveredIdx != s.hoveredCardIdx || newHoveredIsAtk != s.hoveredCardIsAtk {
			// Switched to a different card - reset progress
			s.cardHoverProgress = 0
		}
		s.lastHoveredIdx = newHoveredIdx
		s.lastHoveredIsAtk = newHoveredIsAtk
	}
	s.hoveredCardIdx = newHoveredIdx
	s.hoveredCardIsAtk = newHoveredIsAtk

	// Animate
	if s.hoveredCardIdx >= 0 {
		// Rising
		s.cardHoverProgress += 0.12
		if s.cardHoverProgress > 1.0 {
			s.cardHoverProgress = 1.0
		}
	} else if s.cardHoverProgress > 0 {
		// Falling back down
		s.cardHoverProgress -= 0.12
		if s.cardHoverProgress < 0 {
			s.cardHoverProgress = 0
			s.lastHoveredIdx = -1
		}
	}
}

// isCardSelected returns whether a card ID is currently selected in the card hand.
func (s *GameplayScene) isCardSelected(cardID string) bool {
	return s.selectedCardIDs[cardID]
}

// enterCardSelectionMode starts card selection via the card hand.
func (s *GameplayScene) enterCardSelectionMode(mode string, contextMsg string) {
	s.cardSelectionMode = mode
	s.cardSelectContextMsg = contextMsg
	s.selectedCardIDs = make(map[string]bool)
}

// exitCardSelectionMode clears card selection state.
func (s *GameplayScene) exitCardSelectionMode() {
	s.cardSelectionMode = ""
	s.cardSelectContextMsg = ""
	s.selectedCardIDs = make(map[string]bool)
}

// confirmCardSelection commits the selected cards to the server.
func (s *GameplayScene) confirmCardSelection() {
	cardIDs := make([]string, 0)
	for id := range s.selectedCardIDs {
		cardIDs = append(cardIDs, id)
	}

	if s.cardSelectionMode == "attack" {
		log.Printf("Committing attack card selection: %d cards", len(cardIDs))
		// Build reinforcement info from attack plan state
		var reinforcement *ReinforcementInfo
		if s.selectedReinforcement != nil {
			reinforcement = &ReinforcementInfo{
				UnitType:      s.selectedReinforcement.UnitType,
				FromTerritory: s.selectedReinforcement.FromTerritory,
				WaterBodyID:   s.selectedReinforcement.WaterBodyID,
			}
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
			if s.selectedReinforcement.UnitType == "horse" {
				if s.loadWeaponCheckbox && s.selectedReinforcement.CanCarryWeapon {
					reinforcement.CarryWeapon = true
					reinforcement.WeaponFrom = s.selectedReinforcement.FromTerritory
				}
			}
		}
		if s.attackPlanResolved != nil {
			s.game.ExecuteAttackWithCards(s.attackPlanResolved.TargetTerritory, reinforcement, s.attackPlanResolved.PlanID, cardIDs)
		}
		s.cancelAttackConfirmation()
	} else if s.cardSelectionMode == "defense" {
		log.Printf("Committing defense card selection: %d cards", len(cardIDs))
		s.game.SelectDefenseCards(cardIDs)
		s.ClearHighlightedTerritories()
	}

	s.exitCardSelectionMode()
}

// skipCardSelection skips card selection (plays no cards).
func (s *GameplayScene) skipCardSelection() {
	if s.cardSelectionMode == "attack" {
		log.Println("Skipping attack card selection")
		var reinforcement *ReinforcementInfo
		if s.selectedReinforcement != nil {
			reinforcement = &ReinforcementInfo{
				UnitType:      s.selectedReinforcement.UnitType,
				FromTerritory: s.selectedReinforcement.FromTerritory,
				WaterBodyID:   s.selectedReinforcement.WaterBodyID,
			}
		}
		if s.attackPlanResolved != nil {
			s.game.ExecuteAttackWithCards(s.attackPlanResolved.TargetTerritory, reinforcement, s.attackPlanResolved.PlanID, nil)
		}
		s.cancelAttackConfirmation()
	} else if s.cardSelectionMode == "defense" {
		log.Println("Skipping defense card selection")
		s.game.SelectDefenseCards(nil)
		s.ClearHighlightedTerritories()
	}

	s.exitCardSelectionMode()
}

// drawCardSelectionBar draws the card selection context in the bottom bar.
func (s *GameplayScene) drawCardSelectionBar(screen *ebiten.Image) {
	if s.cardSelectionMode == "" {
		return
	}

	barX := 10
	barY := s.currentBarTop
	barW := ScreenWidth - 20

	// Context message
	DrawLargeText(screen, s.cardSelectContextMsg, barX+20, barY+15, ColorText)

	// Selected count
	selectedCount := len(s.selectedCardIDs)
	countText := fmt.Sprintf("%d cards selected -- Click cards below to toggle", selectedCount)
	DrawText(screen, countText, barX+20, barY+45, ColorTextMuted)

	// Buttons
	btnY := barY + 30
	s.cardSelectConfirmBtn.X = barX + barW - 300
	s.cardSelectConfirmBtn.Y = btnY
	s.cardSelectConfirmBtn.W = 140
	s.cardSelectConfirmBtn.H = 35
	if selectedCount > 0 {
		s.cardSelectConfirmBtn.Text = fmt.Sprintf("Play %d Cards", selectedCount)
	} else {
		s.cardSelectConfirmBtn.Text = "Confirm"
	}
	s.cardSelectConfirmBtn.Disabled = selectedCount == 0
	s.cardSelectConfirmBtn.Update()
	s.cardSelectConfirmBtn.Draw(screen)

	s.cardSelectSkipBtn.X = barX + barW - 150
	s.cardSelectSkipBtn.Y = btnY
	s.cardSelectSkipBtn.W = 120
	s.cardSelectSkipBtn.H = 35
	s.cardSelectSkipBtn.Update()
	s.cardSelectSkipBtn.Draw(screen)
}

// getActiveCardIdx returns the index and type of the card currently being
// hovered or animating down. Returns (-1, false) if none.
func (s *GameplayScene) getActiveCardIdx() (int, bool) {
	if s.hoveredCardIdx >= 0 {
		return s.hoveredCardIdx, s.hoveredCardIsAtk
	}
	if s.lastHoveredIdx >= 0 && s.cardHoverProgress > 0 {
		return s.lastHoveredIdx, s.lastHoveredIsAtk
	}
	return -1, false
}

// drawCardHand draws the card hand BEHIND the status bar.
// Cards are always fully rendered at their resting position (mostly hidden behind the bar).
// The status bar is drawn on top, hiding the card bodies.
// Selected cards (in card selection mode) are drawn fully raised.
func (s *GameplayScene) drawCardHand(screen *ebiten.Image) {
	if s.combatMode != "cards" {
		return
	}

	barY, cardW, cardH, cardStep, peekH, sidebarEnd := s.cardHandLayout()
	fullTop := barY - cardH

	activeIdx, activeIsAtk := s.getActiveCardIdx()

	// Attack cards - hidden during defense card selection
	hideAttackCards := s.cardSelectionMode == "defense"
	if !hideAttackCards {
		// Draw left to right; skip the active card (drawn last)
		for i, card := range s.myAttackCards {
			if activeIsAtk && i == activeIdx {
				continue // Draw on top after all others
			}
			cx := sidebarEnd + i*cardStep
			// Selected cards stay fully raised
			if s.isCardSelected(card.ID) {
				s.drawCardContentSel(screen, cx, fullTop, cardW, cardH, card, true, true)
			} else {
				s.drawCardContent(screen, cx, barY-peekH, cardW, cardH, card, true)
			}
		}
		// Draw active attack card last (on top of neighbors)
		if activeIsAtk && activeIdx >= 0 && activeIdx < len(s.myAttackCards) {
			card := s.myAttackCards[activeIdx]
			cx := sidebarEnd + activeIdx*cardStep
			selected := s.isCardSelected(card.ID)
			if selected {
				s.drawCardContentSel(screen, cx, fullTop, cardW, cardH, card, true, true)
			} else {
				cardTop := barY - peekH - int(float64(cardH-peekH)*s.cardHoverProgress)
				if cardTop < fullTop {
					cardTop = fullTop
				}
				s.drawCardContent(screen, cx, cardTop, cardW, cardH, card, true)
			}
		}
	}

	// Defense cards - hidden during attack planning/confirmation
	hideDefenseCards := s.showAttackPlan || s.showAttackConfirmation
	if !hideDefenseCards {
		// Draw right to left (index 0 = rightmost, drawn last = on top)
		// Skip the active card (drawn last)
		for i := len(s.myDefenseCards) - 1; i >= 0; i-- {
			if !activeIsAtk && i == activeIdx {
				continue
			}
			card := s.myDefenseCards[i]
			cx := ScreenWidth - 15 - cardW - i*cardStep
			if s.isCardSelected(card.ID) {
				s.drawCardContentSel(screen, cx, fullTop, cardW, cardH, card, false, true)
			} else {
				s.drawCardContent(screen, cx, barY-peekH, cardW, cardH, card, false)
			}
		}
		// Draw active defense card last (on top of neighbors)
		if !activeIsAtk && activeIdx >= 0 && activeIdx < len(s.myDefenseCards) {
			card := s.myDefenseCards[activeIdx]
			cx := ScreenWidth - 15 - cardW - activeIdx*cardStep
			selected := s.isCardSelected(card.ID)
			if selected {
				s.drawCardContentSel(screen, cx, fullTop, cardW, cardH, card, false, true)
			} else {
				cardTop := barY - peekH - int(float64(cardH-peekH)*s.cardHoverProgress)
				if cardTop < fullTop {
					cardTop = fullTop
				}
				s.drawCardContent(screen, cx, cardTop, cardW, cardH, card, false)
			}
		}
	}
}

// drawCardHandHovered redraws ONLY the portion of the active card that is
// above the status bar. Uses a clipped sub-image so the card stays behind
// the bar during animation and only the raised portion peeks out.
// Handles both rising (hovered) and falling (last hovered, animating down).
func (s *GameplayScene) drawCardHandHovered(screen *ebiten.Image) {
	if s.combatMode != "cards" || s.cardHoverProgress <= 0 {
		return
	}

	activeIdx, activeIsAtk := s.getActiveCardIdx()
	if activeIdx < 0 {
		return
	}

	barY, cardW, cardH, cardStep, peekH, sidebarEnd := s.cardHandLayout()

	fullTop := barY - cardH
	cardTop := barY - peekH - int(float64(cardH-peekH)*s.cardHoverProgress)
	if cardTop < fullTop {
		cardTop = fullTop
	}

	// Only need to redraw if card has risen above the peek strip
	visibleAboveBar := barY - cardTop
	if visibleAboveBar <= peekH {
		return
	}

	// If the active card is selected (locked in), it's already drawn fully raised
	// by drawCardHand — don't draw a second animating copy on top.
	if activeIsAtk && activeIdx < len(s.myAttackCards) {
		if s.isCardSelected(s.myAttackCards[activeIdx].ID) {
			return
		}
	} else if !activeIsAtk && activeIdx < len(s.myDefenseCards) {
		if s.isCardSelected(s.myDefenseCards[activeIdx].ID) {
			return
		}
	}

	// Create a clipped sub-image covering only the area above the status bar
	clipRect := image.Rect(0, cardTop, ScreenWidth, barY)
	clipped := screen.SubImage(clipRect).(*ebiten.Image)

	if activeIsAtk && activeIdx < len(s.myAttackCards) {
		cx := sidebarEnd + activeIdx*cardStep
		card := s.myAttackCards[activeIdx]
		s.drawCardContent(clipped, cx, cardTop, cardW, cardH, card, true)
	} else if !activeIsAtk && activeIdx < len(s.myDefenseCards) {
		cx := ScreenWidth - 15 - cardW - activeIdx*cardStep
		card := s.myDefenseCards[activeIdx]
		s.drawCardContent(clipped, cx, cardTop, cardW, cardH, card, false)
	}
}
