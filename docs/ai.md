# Lords of Conquest - CPU Players

This document is the plan for making CPU players smarter. It replaces the
earlier personality-based design (aggressive / defensive / passive), which was
never implemented beyond a label and a card-buying preference.

## Decisions

| Topic | Decision |
|-------|----------|
| Skill model | Difficulty (Easy / Medium / Hard) **replaces** personality |
| Selection | Per CPU. "Add CPU" adds a Medium CPU; the host clicks a CPU row in the lobby player list to cycle Easy → Medium → Hard |
| Fairness | All difficulties play fair: they see only what a human sees (no peeking at cards in hand, no resource bonuses) |
| Scope | Territory selection, stockpile placement, shipment, conquest, development, card buying/playing, alliance voting |
| Out of scope | Trading. CPUs keep declining trades and never propose them |

## Current State

All CPU logic lives in `internal/server/handlers.go` (`aiSelectTerritory`,
`aiPlaceStockpile`, `aiShipment`, `aiConquest`, `aiCardAttack`,
`aiDevelopment`, `aiBuyCards`), mixed with persistence and broadcasting.

| Phase | Current behaviour |
|-------|-------------------|
| Territory selection | Random unowned territory |
| Stockpile placement | Random owned territory |
| Shipment | 50% chance to move the stockpile to a random valid destination; never moves units |
| Conquest | Attacks the target with the best attack/defence ratio if it is ≥ 1.0; never brings horses, weapons or boats |
| Card combat | Plays every attack card it holds; as defender plays every defence card it holds |
| Development | First affordable city on the first city-less territory, else a weapon, else a boat |
| Card buying | Up to 2 cards with its most plentiful resource |
| Alliances | Always neutral; treated as offline so "ask" requests never reach it |

Existing problems the rewrite also fixes:

- CPU attacks call `state.Attack` with no allies, so humans set to support
  the defender, or allied with the defender, are ignored. CPU attacks
  should collect allies the same way `RequestAttackPlan` does.
- A CPU dumps its whole hand of defence cards on the first attack against it.

## Architecture

New package `internal/game/ai` (it depends only on `internal/game`, not on the
server, so it can be unit-tested and run in headless simulations).

```go
type Difficulty int

const (
    Easy Difficulty = iota
    Medium
    Hard
)

// Each function reads state and returns a decision. None of them mutate state.
func ChooseTerritory(s *game.GameState, playerID string, d Difficulty, rng *rand.Rand) string
func ChooseStockpile(s *game.GameState, playerID string, d Difficulty, rng *rand.Rand) string
func PlanShipment(s *game.GameState, playerID string, d Difficulty, rng *rand.Rand) *ShipmentDecision
func PlanAttack(s *game.GameState, playerID string, d Difficulty, rng *rand.Rand) *AttackDecision // nil = end turn
func PlanDevelopment(s *game.GameState, playerID string, d Difficulty, rng *rand.Rand) []BuildDecision
func PlanCardPurchases(s *game.GameState, playerID string, d Difficulty, rng *rand.Rand) []CardPurchase
func ChooseDefenseCards(s *game.GameState, playerID string, battle BattleView, d Difficulty, rng *rand.Rand) []string
func VoteAlliance(s *game.GameState, playerID string, battle BattleView, d Difficulty, rng *rand.Rand) game.AllianceSide
```

The server handlers keep their existing role (lock, apply, log, save,
broadcast, chain `checkAndTriggerAI`) but delegate every choice to this package.
An injected `*rand.Rand` makes tests deterministic.

### Shared evaluation helpers (`ai/eval.go`)

- `frontier(s, pid)`: owned territories adjacent to an enemy-owned or
  unclaimed territory.
- `threat(s, pid, terrID)`: strongest attack any single opponent could
  currently bring against a territory (base adjacency plus reachable horses and
  boats), computed with the same rules as `CalculateAttackStrength`.
- `winProbability(s, attack, defense)`: probability of winning given
  `Settings.ChanceLevel`. Low: `attack >= defense`; Medium: 1 if greater,
  0.5 if equal, 0 if less; High: `attack / (attack + defense)`.
- `cityProgress(s, pid)`: cities owned versus `VictoryCities`, to detect
  "one city from winning" for self and opponents.
- `clusterSize(s, pid, terrID)`: size of the connected owned region containing
  a territory.

## Difficulty Levels

| Aspect | Easy | Medium | Hard |
|--------|------|--------|------|
| Decision noise | Picks from the top ~50% of options at random; 25% fully random | Picks from the top 3 with weighted randomness | Always the best-scored option (random tie-break) |
| Territory selection | Resource scoring only, weak adjacency | Full scoring (adjacency first) | Full scoring + denies opponents' clusters and gold |
| Attack threshold (win probability) | ≥ 0.75, ignores reinforcements | ≥ 0.6, uses best single reinforcement | ≥ 0.5 for high-value targets, ≥ 0.65 otherwise; picks the reinforcement that maximises win probability |
| Target value | Weakest target | Weighted by value (stockpile, city, resource, adjacency) | Weighted by value + exposure after capture + leader suppression |
| Development | Random affordable build | Value-ranked builds | Value-ranked builds, saves resources toward a city, reacts to threats |
| Stockpile | Random interior territory | Safest territory | Safest territory, re-evaluated each shipment |
| Cards | Plays all attack cards; plays one defence card | Plays enough cards to reach the threshold | Spends the fewest cards needed; holds defence cards for valuable territories |
| Alliances | Neutral | Votes by simple rule | Votes based on the leader and its own exposure |

## Territory Selection

The user-specified priority: **owning adjacent territories matters most**, then
gold, then grassland, then other resources.

```
score(t) =  W_adj      * ownedNeighbours(t)          // contiguity, dominant term
          + W_cluster  * (joins two separate owned clusters ? 1 : 0)
          + W_res      * resourceValue(t.Resource)
          + W_cityGrid * resourceNeighboursOwned(t)  // future city doubling
          + W_coast    * (coastal && noOwnedCoast ? 1 : 0)
          - W_exposed  * enemyNeighbours(t)          // Medium/Hard only
          + W_deny     * opponentOwnedNeighbours(t) * opponentClusterSize  // Hard only
```

Resource values: gold 5, grassland 3, coal / iron / timber 2, none 0.

Default weights (tuned later by simulation):

| Weight | Easy | Medium | Hard |
|--------|------|--------|------|
| `W_adj` | 1 | 6 | 6 |
| `W_cluster` | 0 | 4 | 5 |
| `W_res` | 1 | 1 | 1 |
| `W_cityGrid` | 0 | 0.5 | 1 |
| `W_coast` | 0 | 1 | 1.5 |
| `W_exposed` | 0 | 0.5 | 1 |
| `W_deny` | 0 | 0 | 0.5 |

With `W_adj = 6`, one owned neighbour (6) beats gold on its own (5), so the
CPU keeps its land contiguous and takes gold when there is a contiguous gold
territory. The first pick (no owned territories yet) is driven by resource
value plus how many good unclaimed territories surround it, which gives a gold
or grassland seed in a resource-rich area.

## Stockpile Placement and Shipment

**Safety score** for a candidate territory:
`safety = clusterSize - 3 * enemyNeighbours - threat / max(1, ownDefense)`,
with a bonus for territories that are not coastal to an enemy-held water body.

- Placement: Easy picks a random territory with no enemy neighbours (falls back
  to random); Medium/Hard pick the highest safety score.
- Shipment: move the stockpile only when the current location's safety is
  meaningfully worse than the best valid destination (Hard also moves away
  from a territory an opponent could beat next round).
- Unit shipment (Medium/Hard): move one horse, weapon or boat to the frontier
  territory with the highest `threat - defense`, or toward the highest-value
  attack target for next round (Hard). Easy never moves units.

## Conquest

For each attackable target (and each reinforcement option on Medium/Hard):

```
value(target) =  10 if it holds an opponent's stockpile
               +  6 if it has a city (+6 more if that player is one city from winning)
               +  resourceValue(target.Resource)
               +  2 * ownedNeighbours(target)        // consolidates our territory
               +  3 if capture eliminates a player
               -  exposure after capture (Hard)      // threat it will face next round
```

Choose the (target, reinforcement) pair that maximises
`winProbability * value`, subject to the difficulty's win-probability threshold.
End the turn when no option clears the threshold. Rules:

- Never bring a unit whose loss would leave our stockpile territory beatable
  (Medium/Hard).
- Hard lowers its threshold to 0.4 against the stockpile or the last city of a
  player who is one city from winning.
- Allies: before attacking, collect attacker and defender allies from alliance
  settings exactly as `RequestAttackPlan` does, and include them in the
  strength calculation. "Ask" players who are online are prompted, as for a
  human attack.

### Card mode

- Attack: compute the base win probability, then add attack cards in order of
  efficiency (flat bonuses first, then conditional cards whose bonus applies
  here) until the threshold is reached. Easy plays all cards; Medium stops at
  the threshold; Hard also keeps one strong card when the attack is already
  comfortable.
- Defence: Easy plays one random card. Medium plays cards until the visible
  attack strength is beaten, but only for stockpile, city or gold territories.
  Hard does the same while weighting the territory's value and the cards left.
  It plays Sabotage only when weapons contribute, Bribe only on stockpile or
  city defence when it has 3 gold, and Double Defense on high-base territories.
- Buying: keep the hand balanced, using attack cards when strong and defence
  cards when exposed. Never spend resources that would prevent building a city
  this development phase (Medium/Hard).

## Development

Generate every affordable build (resources first, then the gold alternative)
and score it:

```
city(t)   = 4 * resourceNeighboursOwned(t)       // production doubling
          + 3 if t itself has a resource
          + 8 if this city reaches VictoryCities
          - 2 * enemyNeighbours(t)               // risk of losing it
weapon(t) = threat(t) - defense(t) on frontier territories (+2 if stockpile / city)
          + 2 if it enables a good attack next round (Hard)
boat(t)   = 3 if it opens an attack across water to a valuable target
          + 1 if we own no boats in that water body
```

- Easy: random affordable build, cities preferred 50% of the time.
- Medium: builds greedily in score order until nothing affordable remains.
- Hard: same, but if it is within 1 resource of a city it skips weapons and
  boats and saves (unless a frontier threat is critical). It prefers resources
  over gold and keeps 2 gold for a Bribe in card mode when it holds one.

## Alliance Voting

CPUs stop being treated as offline for alliance purposes. When a CPU is a
third party on "ask", the server calls `ai.VoteAlliance` immediately, with no
timeout wait.

- Easy: neutral.
- Medium: support the defender if the attacker leads in cities; otherwise neutral.
- Hard: support whichever side keeps the leader (most cities, then most
  territories) from gaining. It never supports an attack on a territory
  adjacent to its own stockpile.

The CPU's own alliance setting stays at "ask" so that it can vote; it is no
longer forced to neutral.

## Data Model and UI

- Database: migration `ALTER TABLE game_players ADD COLUMN ai_difficulty TEXT
  DEFAULT 'medium'`. The old `ai_personality` column is left in place but
  unused (SQLite cannot drop columns cheaply, and old rows stay readable).
- `game.Player`: replace `AIPersonality` with `AIDifficulty` (JSON
  `aiDifficulty`). Saved games with the old field load as Medium.
- Protocol: `AddAIPayload{Difficulty string}` (default "medium"); new
  `SetAIDifficultyPayload{PlayerID, Difficulty}` handled host-only and
  lobby-only; lobby player info gains `ai_difficulty`.
- Client lobby: the CPU row subtext shows "CPU (Easy / Medium / Hard)".
  For the host, clicking a CPU row sends `SetAIDifficulty` with the next
  difficulty in the cycle.
- `server_test.go` end-to-end tests switch to `AddAIPayload{Difficulty: ...}`.

## Pacing

Keep the existing 100 ms trigger, but add a short "thinking" delay per action
(territory pick ~600 ms, attack ~900 ms, build ~400 ms, plus up to 300 ms of
jitter) so that humans can follow CPU turns. Tests and simulations set it to 0.

## Testing

- Unit tests per decision function on small hand-built maps (for example "picks
  adjacent grassland over distant gold", "picks adjacent gold over adjacent
  timber", "never attacks below threshold", "moves stockpile away from threat",
  "builds city where production doubles").
- Headless simulation harness `cmd/aisim`: runs N games CPU vs CPU directly on
  `GameState` (no server) and reports win rate by difficulty, average rounds,
  cities built and attack success. Acceptance target: in 4-player games Hard
  beats Medium ≥ 60% of head-to-heads and Medium beats Easy ≥ 60%.

## Implementation Order

1. **Difficulty plumbing**: database column, `Player.AIDifficulty`, protocol,
   lobby display and click-to-cycle, and test updates. Personality code removed.
2. **AI package skeleton**: `internal/game/ai` with difficulty, rng, the eval
   helpers and unit tests, and the handlers delegating to it (behaviour
   unchanged at first).
3. **Territory selection** scoring.
4. **Stockpile placement and shipment** (stockpile, then units).
5. **Conquest**: target valuation, win probability, reinforcements and allies
   on CPU attacks.
6. **Development** scoring.
7. **Card mode**: buying, attack card selection, defence card selection.
8. **Alliance voting** for CPUs.
9. **Simulation harness** and weight tuning, plus pacing delays.

Steps 3–8 are independent once step 2 lands and can be done in any order.
