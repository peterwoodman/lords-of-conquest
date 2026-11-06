// Package sim plays complete CPU-vs-CPU games directly on the game state, without the
// server, for balance testing. It drives the phases the same way the server does.
package sim

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"

	"lords-of-conquest/internal/game"
	"lords-of-conquest/internal/game/ai"
	"lords-of-conquest/pkg/maps"
)

// Config describes one simulated game.
type Config struct {
	Difficulties  []game.AIDifficulty // one CPU per entry
	ChanceLevel   game.ChanceLevel
	CombatMode    game.CombatMode
	VictoryCities int
	Map           maps.GeneratorOptions
	MaxRounds     int // the game is a draw after this many rounds
	Seed          int64
}

// PlayerStats summarises one CPU's game.
type PlayerStats struct {
	ID          string
	Difficulty  game.AIDifficulty
	Attacks     int
	AttacksWon  int
	CitiesBuilt int
	Cities      int // at the end of the game
	Territories int // at the end of the game
	Eliminated  bool
}

// Result is the outcome of a simulated game.
type Result struct {
	WinnerID string // "" for a draw
	Rounds   int
	Players  []*PlayerStats
}

// Winner returns the winner's stats, or nil for a draw.
func (r Result) Winner() *PlayerStats {
	for _, p := range r.Players {
		if p.ID == r.WinnerID {
			return p
		}
	}
	return nil
}

// maxSteps bounds the number of decisions in one game, in case a phase stops advancing.
const maxSteps = 200000

var colors = []game.PlayerColor{
	game.ColorOrange, game.ColorCyan, game.ColorGreen, game.ColorYellow,
	game.ColorPurple, game.ColorRed, game.ColorBlue,
}

// Run plays one game to completion.
func Run(cfg Config) (Result, error) {
	if len(cfg.Difficulties) > len(colors) {
		return Result{}, fmt.Errorf("at most %d players", len(colors))
	}
	m, _ := maps.NewGenerator(cfg.Map).Generate()

	players := make([]*game.Player, len(cfg.Difficulties))
	stats := make(map[string]*PlayerStats)
	var result Result
	for i, d := range cfg.Difficulties {
		id := fmt.Sprintf("cpu%d", i+1)
		p := game.NewPlayer(id, fmt.Sprintf("CPU%d (%s)", i+1, d.Label()), colors[i])
		p.IsAI = true
		p.AIDifficulty = d
		p.Alliance = game.AllianceAsk
		players[i] = p
		stats[id] = &PlayerStats{ID: id, Difficulty: d}
		result.Players = append(result.Players, stats[id])
	}

	s, err := game.InitializeGame(maps.ToGameData(m), players, game.Settings{
		ChanceLevel:   cfg.ChanceLevel,
		VictoryCities: cfg.VictoryCities,
		MaxPlayers:    len(players),
		CombatMode:    cfg.CombatMode,
	})
	if err != nil {
		return result, err
	}

	g := &runner{s: s, rng: rand.New(rand.NewSource(cfg.Seed)), stats: stats}
	over, err := g.play(cfg.MaxRounds)
	if err != nil {
		return result, err
	}

	result.Rounds = s.Round
	if over {
		if w := s.GetWinner(); w != nil {
			result.WinnerID = w.ID
		}
	}
	for id, st := range stats {
		st.Cities = s.CountCities(id)
		st.Territories = len(s.GetPlayerTerritories(id))
		st.Eliminated = s.Players[id].Eliminated
	}
	return result, nil
}

type runner struct {
	s     *game.GameState
	rng   *rand.Rand
	stats map[string]*PlayerStats
}

// play runs the game until it ends (true) or reaches maxRounds (false).
func (g *runner) play(maxRounds int) (bool, error) {
	s := g.s
	for range maxSteps {
		if s.Round > 0 && s.IsEliminationVictory() {
			return true, nil
		}
		if s.Round > maxRounds {
			return false, nil
		}
		current := s.Players[s.CurrentPlayerID]
		if current == nil && !s.StockpilePlacementPending {
			return false, fmt.Errorf("no current player in %s", s.Phase)
		}

		switch s.Phase {
		case game.PhaseTerritorySelection:
			id := ai.ChooseTerritory(s, current.ID, current.AIDifficulty, g.rng)
			if err := s.SelectTerritory(current.ID, id); err != nil {
				return false, fmt.Errorf("select territory %q: %w", id, err)
			}

		case game.PhaseProduction:
			if err := g.production(); err != nil {
				return false, err
			}

		case game.PhaseCommerce:
			if s.CommerceSubPhase == game.SubPhaseTrade {
				if err := s.SkipTrade(current.ID); err != nil {
					return false, fmt.Errorf("skip trade: %w", err)
				}
			} else {
				g.shipment(current)
			}

		case game.PhaseConquest:
			over, err := g.conquest(current)
			if err != nil || over {
				return over, err
			}

		case game.PhaseDevelopment:
			if err := g.development(current); err != nil {
				return false, err
			}

		default:
			return false, fmt.Errorf("unknown phase %v", s.Phase)
		}
	}
	return false, errors.New("game did not finish within the step limit")
}

func (g *runner) production() error {
	s := g.s
	if s.StockpilePlacementPending {
		needing := s.GetPlayersNeedingStockpile()
		sort.Strings(needing)
		for _, id := range needing {
			p := s.Players[id]
			territory := ai.ChooseStockpile(s, id, p.AIDifficulty, g.rng)
			if err := s.PlaceStockpile(id, territory); err != nil {
				return fmt.Errorf("place stockpile for %s: %w", id, err)
			}
		}
		s.ProductionPending = true
	}
	if !s.ProductionPending {
		return errors.New("production phase with nothing pending")
	}
	pm := game.NewPhaseManager(s)
	for _, id := range s.PlayerOrder {
		if p := s.Players[id]; p != nil && !p.Eliminated {
			results, _ := pm.CalculateProductionForPlayer(id)
			pm.ApplyProductionResults(id, results)
		}
	}
	s.CompleteProduction()
	return nil
}

func (g *runner) shipment(p *game.Player) {
	s := g.s
	d := ai.PlanShipment(s, p.ID, p.AIDifficulty, g.rng)
	var err error
	switch d.Kind {
	case ai.ShipStockpile:
		err = s.MoveStockpile(p.ID, d.To)
	case ai.ShipUnit:
		err = s.MoveUnit(p.ID, d.UnitType, d.From, d.To, d.WaterBodyID, false, false)
	default:
		err = s.SkipShipment(p.ID)
	}
	if err != nil && s.CurrentPlayerID == p.ID {
		s.SkipShipment(p.ID)
	}
}

// conquest makes one attack or ends the player's conquest turn. Returns true if the game ended.
func (g *runner) conquest(p *game.Player) (bool, error) {
	s := g.s
	d := ai.PlanAttack(s, p.ID, p.AIDifficulty, g.rng)
	if d == nil {
		if err := s.EndConquest(p.ID); err != nil {
			return false, fmt.Errorf("end conquest: %w", err)
		}
		return g.roundEndedInVictory(), nil
	}

	var result *game.CombatResult
	var err error
	if s.Settings.CombatMode == game.CombatModeCards {
		result, err = g.cardAttack(p, d)
	} else {
		result, err = s.AttackWithAllies(p.ID, d.Target, d.Brought, d.AttackerAllies, d.DefenderAllies)
	}
	if err != nil {
		// A planned attack should always be valid; don't loop on it
		s.EndConquest(p.ID)
		return false, fmt.Errorf("attack on %s: %w", d.Target, err)
	}
	st := g.stats[p.ID]
	st.Attacks++
	if result.AttackerWins {
		st.AttacksWon++
	}
	if s.IsEliminationVictory() {
		return true, nil
	}
	return g.roundEndedInVictory(), nil
}

// roundEndedInVictory reports whether conquest wrapped up the round with a winner. NextPhase
// then leaves the phase at conquest, and the current player has no attacks left.
func (g *runner) roundEndedInVictory() bool {
	s := g.s
	current := s.Players[s.CurrentPlayerID]
	return s.Phase == game.PhaseConquest && current != nil && current.AttacksRemaining <= 0 && s.IsGameOver()
}

// cardAttack resolves a card-mode attack the way the server does, with the defender
// choosing its cards as a CPU.
func (g *runner) cardAttack(p *game.Player, d *ai.AttackDecision) (*game.CombatResult, error) {
	s := g.s
	if err := s.ValidateAttack(p.ID, d.Target, d.Brought); err != nil {
		return nil, err
	}
	target := s.Territories[d.Target]
	attackCards := cardsOfType(p.RemoveCardsFromHand(d.AttackCards), game.CardTypeAttack)

	var defenseCards []game.CombatCard
	if defender := s.Players[target.Owner]; defender != nil {
		baseAttack := s.CalculateAttackWithAllies(p.ID, target, d.Brought, d.AttackerAllies)
		baseDefense := s.CalculateDefenseWithAllies(target, d.DefenderAllies)
		ids := ai.ChooseDefenseCards(s, defender.ID, d.Target, baseAttack, baseDefense, len(attackCards), defender.AIDifficulty, g.rng)
		defenseCards = cardsOfType(defender.RemoveCardsFromHand(ids), game.CardTypeDefense)
	}

	plan := &game.AttackPlan{TargetTerritory: d.Target, BroughtUnit: d.Brought}
	result, _ := s.ExecuteCardAttackWithAllies(p.ID, plan, d.AttackerAllies, d.DefenderAllies, attackCards, defenseCards)

	p.AttacksRemaining--
	if !result.AttackerWins && p.AttacksRemaining == 1 {
		p.AttacksRemaining = 0
	}
	if p.AttacksRemaining <= 0 && s.CurrentPlayerID == p.ID {
		s.AdvanceConquestTurn()
	}
	return result, nil
}

func cardsOfType(cards []game.CombatCard, t game.CardType) []game.CombatCard {
	var result []game.CombatCard
	for _, c := range cards {
		if c.CardType == t {
			result = append(result, c)
		}
	}
	return result
}

func (g *runner) development(p *game.Player) error {
	s := g.s
	for range 20 {
		d := ai.PlanDevelopment(s, p.ID, p.AIDifficulty, g.rng)
		if d == nil {
			break
		}
		var err error
		if d.Type == game.BuildBoat && d.WaterBodyID != "" {
			err = s.BuildBoatInWater(p.ID, d.Territory, d.WaterBodyID, d.UseGold)
		} else {
			err = s.Build(p.ID, d.Type, d.Territory, d.UseGold)
		}
		if err != nil {
			return fmt.Errorf("build %s on %s: %w", d.Type, d.Territory, err)
		}
		if d.Type == game.BuildCity {
			g.stats[p.ID].CitiesBuilt++
		}
	}
	for _, purchase := range ai.PlanCardPurchases(s, p.ID, p.AIDifficulty) {
		if _, err := s.BuyCard(p.ID, purchase.Type, purchase.Resource); err != nil {
			return fmt.Errorf("buy %s card: %w", purchase.Type, err)
		}
	}
	if err := s.EndDevelopment(p.ID); err != nil {
		return fmt.Errorf("end development: %w", err)
	}
	return nil
}
