package ai

import (
	"fmt"
	"math/rand"
	"sort"

	"lords-of-conquest/internal/game"
)

// stockpileValue is how much losing the stockpile counts against a position, relative
// to territoryValue.
const stockpileValue = 10

// minShipmentGain is the smallest drop in expected loss worth spending the shipment on.
const minShipmentGain = 0.5

// ShipmentKind is the action a CPU takes in its shipment turn.
type ShipmentKind int

const (
	ShipSkip ShipmentKind = iota
	ShipStockpile
	ShipUnit
)

// ShipmentDecision is a CPU's single shipment action.
type ShipmentDecision struct {
	Kind        ShipmentKind
	UnitType    string // "horse", "weapon" or "boat" when Kind is ShipUnit
	From        string // unit's territory when Kind is ShipUnit
	To          string // destination territory
	WaterBodyID string // boat's water body when UnitType is "boat"
}

// defenceModel estimates how much a player stands to lose in the next conquest phase.
// Opponents' attack strength doesn't depend on the player's own units, so threats are
// computed once and reused while trying different unit and stockpile positions.
type defenceModel struct {
	s        *game.GameState
	playerID string
	threat   map[string]int
}

func newDefenceModel(s *game.GameState, playerID string) *defenceModel {
	m := &defenceModel{s: s, playerID: playerID, threat: make(map[string]int)}
	for _, id := range sortedTerritoryIDs(s) {
		if t := s.Territories[id]; t.Owner == playerID {
			m.threat[id] = threatTo(s, playerID, t)
		}
	}
	return m
}

// lossChance is the probability the strongest single attack on t succeeds.
func (m *defenceModel) lossChance(t *game.Territory) float64 {
	threat := m.threat[t.ID]
	if threat == 0 {
		return 0
	}
	return winProbability(m.s, threat, m.s.CalculateDefenseStrength(t))
}

// expectedLoss sums the value of each owned territory weighted by its chance of being
// captured, with the stockpile at stockpileAt.
func (m *defenceModel) expectedLoss(stockpileAt string) float64 {
	loss := 0.0
	for _, id := range sortedTerritoryIDs(m.s) {
		t := m.s.Territories[id]
		if t.Owner != m.playerID {
			continue
		}
		value := territoryValue(t)
		if id == stockpileAt {
			value += stockpileValue
		}
		loss += m.lossChance(t) * value
	}
	return loss
}

// territoryValue is what a player loses when t is captured (excluding the stockpile).
func territoryValue(t *game.Territory) float64 {
	v := 1 + resourceValue(t.Resource)
	if t.HasCity {
		v += 6
	}
	if t.HasWeapon {
		v += 1.5
	}
	if t.HasHorse {
		v += 0.5
	}
	return v + float64(t.TotalBoats())
}

// ChooseStockpile picks the territory to place the stockpile on at the start of the game.
// Returns "" if the player owns no territory.
func ChooseStockpile(s *game.GameState, playerID string, d game.AIDifficulty, rng *rand.Rand) string {
	m := newDefenceModel(s, playerID)
	easy := game.ParseAIDifficulty(string(d)) == game.AIEasy

	var opts []option
	for _, id := range sortedTerritoryIDs(s) {
		t := s.Territories[id]
		if t.Owner != playerID {
			continue
		}
		var score float64
		if easy {
			score = -float64(enemyNeighbours(s, playerID, t))
		} else {
			margin := s.CalculateDefenseStrength(t) - m.threat[id]
			score = -stockpileValue*m.lossChance(t) + 0.25*float64(margin)
		}
		opts = append(opts, option{id: id, score: score})
	}
	return choose(opts, d, rng)
}

// PlanShipment picks the shipment action that most reduces the player's expected loss
// in the next conquest phase. Easy only considers moving the stockpile.
func PlanShipment(s *game.GameState, playerID string, d game.AIDifficulty, rng *rand.Rand) ShipmentDecision {
	player := s.Players[playerID]
	if player == nil || player.StockpileTerritory == "" {
		return ShipmentDecision{Kind: ShipSkip}
	}
	m := newDefenceModel(s, playerID)
	stockpileAt := player.StockpileTerritory
	base := m.expectedLoss(stockpileAt)

	decisions := map[string]ShipmentDecision{"skip": {Kind: ShipSkip}}
	opts := []option{{id: "skip", score: 0}}
	add := func(dec ShipmentDecision, loss float64) {
		key := fmt.Sprintf("%d:%s:%s:%s:%s", dec.Kind, dec.UnitType, dec.From, dec.To, dec.WaterBodyID)
		decisions[key] = dec
		opts = append(opts, option{id: key, score: base - loss})
	}

	destinations := s.GetValidStockpileDestinations(playerID)
	sort.Strings(destinations)
	for _, dest := range destinations {
		if dest != stockpileAt {
			add(ShipmentDecision{Kind: ShipStockpile, To: dest}, m.expectedLoss(dest))
		}
	}

	if game.ParseAIDifficulty(string(d)) != game.AIEasy {
		for _, id := range sortedTerritoryIDs(s) {
			from := s.Territories[id]
			if from.Owner != playerID {
				continue
			}
			if from.HasHorse {
				for _, to := range horseDestinations(s, playerID, from) {
					from.HasHorse, to.HasHorse = false, true
					loss := m.expectedLoss(stockpileAt)
					from.HasHorse, to.HasHorse = true, false
					add(ShipmentDecision{Kind: ShipUnit, UnitType: "horse", From: id, To: to.ID}, loss)
				}
			}
			if from.HasWeapon {
				for _, to := range neighbours(s, from) {
					if to.Owner != playerID || to.HasWeapon {
						continue
					}
					from.HasWeapon, to.HasWeapon = false, true
					loss := m.expectedLoss(stockpileAt)
					from.HasWeapon, to.HasWeapon = true, false
					add(ShipmentDecision{Kind: ShipUnit, UnitType: "weapon", From: id, To: to.ID}, loss)
				}
			}
			for _, wb := range sortedBoatWaters(from) {
				for _, to := range boatDestinations(s, playerID, from, wb) {
					from.RemoveBoat(wb)
					to.AddBoat(wb)
					loss := m.expectedLoss(stockpileAt)
					to.RemoveBoat(wb)
					from.AddBoat(wb)
					add(ShipmentDecision{Kind: ShipUnit, UnitType: "boat", From: id, To: to.ID, WaterBodyID: wb}, loss)
				}
			}
		}
	}

	key := choose(opts, d, rng)
	if scoreOf(opts, key) < minShipmentGain {
		return ShipmentDecision{Kind: ShipSkip}
	}
	return decisions[key]
}

// scoreOf returns the score of the option with the given id.
func scoreOf(opts []option, id string) float64 {
	for _, o := range opts {
		if o.id == id {
			return o.score
		}
	}
	return 0
}

// horseDestinations lists owned territories without a horse that a horse at from can
// reach: adjacent, or two steps through an owned territory.
func horseDestinations(s *game.GameState, playerID string, from *game.Territory) []*game.Territory {
	seen := map[string]bool{from.ID: true}
	var result []*game.Territory
	consider := func(t *game.Territory) {
		if !seen[t.ID] && t.Owner == playerID && !t.HasHorse {
			result = append(result, t)
		}
		seen[t.ID] = true
	}
	for _, mid := range neighbours(s, from) {
		consider(mid)
	}
	for _, mid := range neighbours(s, from) {
		if mid.Owner != playerID {
			continue
		}
		for _, t := range neighbours(s, mid) {
			consider(t)
		}
	}
	return result
}

// sortedBoatWaters returns the water bodies where t has boats.
func sortedBoatWaters(t *game.Territory) []string {
	var waters []string
	for wb, count := range t.Boats {
		if count > 0 {
			waters = append(waters, wb)
		}
	}
	sort.Strings(waters)
	return waters
}

// boatDestinations lists owned territories on water body wb with room for a boat.
func boatDestinations(s *game.GameState, playerID string, from *game.Territory, wb string) []*game.Territory {
	water := s.WaterBodies[wb]
	if water == nil {
		return nil
	}
	var result []*game.Territory
	for _, id := range water.Territories {
		t := s.Territories[id]
		if t != nil && t.ID != from.ID && t.Owner == playerID && t.CanAddBoat() {
			result = append(result, t)
		}
	}
	return result
}
