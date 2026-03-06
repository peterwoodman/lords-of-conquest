package client

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"lords-of-conquest/internal/protocol"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)


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

// cardHandLayout returns shared layout constants for the card hand display.
// barY tracks the current bottom bar top so cards move with the bar.
func (s *GameplayScene) cardHandLayout() (barY, cardW, cardH, cardStep, peekH, sidebarEnd int) {
	w := 110
	gap := 15
	// barY is the TOP of the bottom bar -- cards tuck under it
	// cardH must be <= barHeight so cards don't poke below the bar
	return s.currentBarTop, w, 100, w + gap, 30, 300
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
