# Lords of Conquest - AI System

## Overview

The AI system replicates the three personality types from the original game: **Aggressive**, **Defensive**, and **Passive**. Each AI makes decisions across all game phases based on its personality.

## AI Architecture

```go
type AI interface {
    // Called during territory selection phase
    SelectTerritory(state *GameState, available []Territory) Territory
    
    // Called during trade phase
    ProposeTrade(state *GameState, players []Player) *TradeOffer
    RespondToTrade(state *GameState, offer *TradeOffer) bool
    
    // Called during shipment phase
    DecideShipment(state *GameState) *ShipmentAction
    
    // Called during conquest phase
    PlanAttacks(state *GameState) []AttackPlan
    VoteAlliance(state *GameState, battle *Battle) AllianceSide
    
    // Called during development phase
    PlanDevelopment(state *GameState) []BuildAction
}
```

## Personality Traits

### Aggressive AI

**Philosophy**: Attack first, ask questions later.

| Phase | Behavior |
|-------|----------|
| Territory Selection | Prioritizes territories adjacent to opponents; values offensive positions |
| Trade | Rarely trades; demands unfavorable terms; breaks deals easily |
| Shipment | Moves units toward enemy borders; stockpile near front lines |
| Conquest | Attacks whenever odds are reasonable (≥70%); takes risks |
| Alliance | Sides with attackers; loves chaos |
| Development | Prioritizes weapons over cities; builds cities only when safe |

**Evaluation Weights**:
```go
AggressiveWeights = Weights{
    ResourceValue:    0.3,
    OffensivePosition: 0.5,
    DefensivePosition: 0.1,
    CityBuilding:     0.1,
}
```

### Defensive AI

**Philosophy**: Secure positions, build strength, attack only when certain.

| Phase | Behavior |
|-------|----------|
| Territory Selection | Prioritizes easily defensible territories; values chokepoints |
| Trade | Trades fairly; honors agreements; seeks alliances |
| Shipment | Positions units for defense; stockpile in safest territory |
| Conquest | Only attacks with overwhelming force (≥90%); consolidates |
| Alliance | Sides with defenders; maintains stability |
| Development | Balances weapons and cities; prefers interior city placement |

**Evaluation Weights**:
```go
DefensiveWeights = Weights{
    ResourceValue:    0.4,
    OffensivePosition: 0.1,
    DefensivePosition: 0.3,
    CityBuilding:     0.2,
}
```

### Passive AI

**Philosophy**: Build economy, avoid conflict, win through development.

| Phase | Behavior |
|-------|----------|
| Territory Selection | Prioritizes resource-rich territories; avoids contested areas |
| Trade | Trades generously; seeks positive relationships |
| Shipment | Moves stockpile to safest location; rarely moves units |
| Conquest | Rarely attacks; only when near victory or under threat |
| Alliance | Often neutral; sides with perceived underdog |
| Development | Heavy city focus; weapons only when threatened |

**Evaluation Weights**:
```go
PassiveWeights = Weights{
    ResourceValue:    0.5,
    OffensivePosition: 0.0,
    DefensivePosition: 0.2,
    CityBuilding:     0.3,
}
```

---

## Decision Algorithms

### Territory Selection

Territories are scored based on:

```go
func ScoreTerritory(t Territory, personality Personality) float64 {
    score := 0.0
    
    // Base resource value
    if t.Resource != None {
        score += personality.ResourceValue * ResourceValues[t.Resource]
    }
    
    // Adjacency to own territories (contiguity bonus)
    score += float64(len(t.AdjacentOwned)) * 0.2
    
    // Adjacency to enemy territories
    score += float64(len(t.AdjacentEnemy)) * personality.OffensivePosition
    
    // Defensive value (fewer adjacent territories = easier to defend)
    defensibility := 1.0 / float64(len(t.Adjacent))
    score += defensibility * personality.DefensivePosition
    
    // Coastal access (for boat building)
    if t.CoastalTiles > 0 {
        score += 0.1 * float64(t.CoastalTiles)
    }
    
    return score
}
```

### Combat Decision

```go
func (ai *AI) ShouldAttack(state *GameState, target Territory) bool {
    strength := CalculateAttackStrength(state, target)
    defense := CalculateDefenseStrength(state, target)
    
    ratio := float64(strength) / float64(defense)
    
    // Adjust threshold based on personality and chance level
    threshold := ai.personality.AttackThreshold
    if state.Settings.ChanceLevel == High {
        threshold *= 1.2 // More conservative with high randomness
    }
    
    // Strategic value adjustments
    if target.HasCity {
        threshold *= 0.8 // Lower threshold for city captures
    }
    if target.ID == state.GetStockpileTerritory(target.Owner) {
        threshold *= 0.7 // Much lower threshold for stockpile captures
    }
    
    return ratio >= threshold
}
```

### Attack Thresholds by Personality

| Personality | Base Threshold | Min Threshold |
|-------------|----------------|---------------|
| Aggressive | 0.7 (70%) | 0.5 |
| Defensive | 1.3 (130%) | 1.0 |
| Passive | 1.5 (150%) | 1.2 |

### Development Priority

```go
func (ai *AI) PrioritizeDevelopment(state *GameState) []BuildAction {
    options := []BuildOption{}
    
    // Evaluate city building
    for _, t := range state.MyTerritories() {
        if !t.HasCity && CanBuildCity(state, t) {
            value := ai.evaluateCityValue(state, t)
            options = append(options, BuildOption{Type: City, Territory: t, Value: value})
        }
    }
    
    // Evaluate weapons
    for _, t := range state.MyTerritories() {
        if !t.HasWeapon && CanBuildWeapon(state) {
            value := ai.evaluateWeaponValue(state, t)
            options = append(options, BuildOption{Type: Weapon, Territory: t, Value: value})
        }
    }
    
    // Evaluate boats
    for _, t := range state.MyTerritories() {
        if t.CoastalTiles > t.Boats && CanBuildBoat(state) {
            value := ai.evaluateBoatValue(state, t)
            options = append(options, BuildOption{Type: Boat, Territory: t, Value: value})
        }
    }
    
    // Sort by value and return affordable actions
    sort.Slice(options, func(i, j int) bool {
        return options[i].Value > options[j].Value
    })
    
    return ai.selectAffordableActions(state, options)
}
```

### City Value Evaluation

```go
func (ai *AI) evaluateCityValue(state *GameState, t Territory) float64 {
    value := 0.0
    
    // Count resources that would be doubled
    resourceCount := 0
    for _, adj := range t.AdjacentOwned {
        if adj.Resource != None && adj.Resource != Horses {
            resourceCount++
        }
    }
    if t.Resource != None && t.Resource != Horses {
        resourceCount++
    }
    value += float64(resourceCount) * 2.0
    
    // Defensive bonus
    value += float64(len(t.AdjacentOwned)) * 0.3
    
    // Penalty for border territories (risk of capture)
    borderPenalty := float64(len(t.AdjacentEnemy)) * 0.5
    value -= borderPenalty * (1.0 - ai.personality.OffensivePosition)
    
    // Victory progress bonus
    citiesOwned := state.CountCities(ai.PlayerID)
    if citiesOwned == state.Settings.VictoryCities-1 {
        value *= 2.0 // Critical city!
    }
    
    return value * ai.personality.CityBuilding
}
```

---

## Trade AI

### Trade Valuation

```go
// Resource values vary by what the AI needs
func (ai *AI) GetResourceValue(resource Resource, state *GameState) float64 {
    baseValue := BaseResourceValues[resource]
    
    // Increase value if we need it
    needs := ai.analyzeNeeds(state)
    if needs[resource] > 0 {
        baseValue *= 1.0 + float64(needs[resource])*0.2
    }
    
    // Gold is universal
    if resource == Gold {
        baseValue *= 1.2
    }
    
    return baseValue
}

func (ai *AI) EvaluateTrade(offer *TradeOffer, state *GameState) bool {
    offering := 0.0
    for resource, amount := range offer.Offer {
        offering += ai.GetResourceValue(resource, state) * float64(amount)
    }
    
    requesting := 0.0
    for resource, amount := range offer.Request {
        requesting += ai.GetResourceValue(resource, state) * float64(amount)
    }
    
    ratio := offering / requesting
    
    // Personality affects acceptance threshold
    switch ai.Personality {
    case Aggressive:
        return ratio >= 1.3 // Demands favorable trades
    case Defensive:
        return ratio >= 0.9 // Fair trades
    case Passive:
        return ratio >= 0.7 // Accepts some bad trades for goodwill
    }
    
    return false
}
```

---

## Alliance Voting

```go
func (ai *AI) VoteAlliance(state *GameState, battle *Battle) AllianceSide {
    attacker := state.GetPlayer(battle.AttackerID)
    defender := state.GetPlayer(battle.DefenderID)
    
    // Evaluate threats
    attackerThreat := ai.evaluateThreat(attacker)
    defenderThreat := ai.evaluateThreat(defender)
    
    // Factor in current standings
    attackerCities := state.CountCities(attacker.ID)
    defenderCities := state.CountCities(defender.ID)
    
    switch ai.Personality {
    case Aggressive:
        // Side with attacker unless they're winning
        if attackerCities >= state.Settings.VictoryCities-1 {
            return DefenderSide
        }
        return AttackerSide
        
    case Defensive:
        // Side with defender unless they're the biggest threat
        if defenderThreat > attackerThreat {
            return AttackerSide
        }
        return DefenderSide
        
    case Passive:
        // Support the underdog
        if battle.AttackStrength > battle.DefenseStrength {
            return DefenderSide
        }
        return Neutral
    }
    
    return Neutral
}
```

---

## Surrender Logic

The original game featured AI surrender with humorous messages. We replicate this:

```go
func (ai *AI) ShouldSurrender(state *GameState) (bool, string) {
    myTerritories := len(state.GetTerritories(ai.PlayerID))
    myCities := state.CountCities(ai.PlayerID)
    myResources := state.GetStockpile(ai.PlayerID).Total()
    
    // Check if hopelessly behind
    leadingPlayer := state.GetLeadingPlayer()
    leadingCities := state.CountCities(leadingPlayer.ID)
    
    hopelessness := 0.0
    
    if myTerritories < 3 {
        hopelessness += 0.4
    }
    if myCities == 0 && leadingCities >= 2 {
        hopelessness += 0.3
    }
    if myResources < 3 {
        hopelessness += 0.2
    }
    if ai.stockpileLost {
        hopelessness += 0.3
    }
    
    if hopelessness >= 0.7 {
        return true, ai.getSurrenderMessage()
    }
    
    return false, ""
}

var surrenderMessages = []string{
    "You win! I never liked this game anyway.",
    "I hereby declare you the winner. Happy now?",
    "My armies have mutinied. You win.",
    "I'm taking my toys and going home!",
    "Victory is yours... this time.",
    "I surrender! Please don't hurt my horses!",
}
```

---

## Performance Considerations

### Thinking Time

To make AI feel more natural and allow players time to observe, AI actions are delayed:

```go
const (
    MinThinkingTime = 500 * time.Millisecond
    MaxThinkingTime = 2 * time.Second
)

func (ai *AI) ThinkingTime(action ActionType) time.Duration {
    base := MinThinkingTime
    
    switch action {
    case SelectTerritory:
        base = 800 * time.Millisecond
    case PlanAttack:
        base = 1200 * time.Millisecond
    case Build:
        base = 600 * time.Millisecond
    }
    
    // Add randomness
    jitter := time.Duration(rand.Intn(500)) * time.Millisecond
    return base + jitter
}
```

### Move Calculation Timeout

As in the original game, complex calculations have a timeout:

```go
const AICalculationTimeout = 10 * time.Second

func (ai *AI) CalculateMove(state *GameState) Action {
    ctx, cancel := context.WithTimeout(context.Background(), AICalculationTimeout)
    defer cancel()
    
    resultChan := make(chan Action)
    go func() {
        resultChan <- ai.calculateBestMove(state)
    }()
    
    select {
    case result := <-resultChan:
        return result
    case <-ctx.Done():
        // Return best move found so far
        return ai.getBestMoveFoundSoFar()
    }
}
```

---

## Testing AI

### AI vs AI Games

For balance testing and debugging:

```go
func RunAIGame(personalities []Personality, mapID string, settings GameSettings) GameResult {
    state := NewGame(settings, mapID)
    
    ais := make([]AI, len(personalities))
    for i, p := range personalities {
        ais[i] = NewAI(p, state.Players[i].ID)
    }
    
    for !state.IsGameOver() {
        currentAI := ais[state.CurrentPlayerIndex]
        action := currentAI.GetAction(state)
        state.ApplyAction(action)
    }
    
    return GameResult{
        Winner: state.Winner(),
        Rounds: state.Round,
        // ... statistics
    }
}
```

### Metrics to Track

- Win rate per personality
- Average game length
- Cities built vs conquered ratio
- Resource efficiency
- Attack success rate

