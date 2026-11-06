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

Implemented in `internal/game/ai/selection.go`:

```
score(t) =  W_adj      * ownedNeighbours(t)          // contiguity, dominant term
          + W_joins    * (owned regions t connects - 1)
          + W_res      * resourceValue(t.Resource)
          + W_cityGrid * ownedNeighboursWithResource(t)  // future city doubling
          + W_room     * unclaimedNeighbours(t)
          + W_seed     * resourceValue of unclaimed neighbours   // first pick only
          + W_coast    * (coastal && noOwnedCoast ? 1 : 0)
          - W_exposed  * enemyNeighbours(t)
          + W_deny     * neighbours owned by the most-adjacent opponent
```

Resource values: gold 5, grassland 3, coal / iron / timber 2, none 0.

Default weights (tuned later by simulation):

| Weight | Easy | Medium | Hard |
|--------|------|--------|------|
| `W_adj` | 1 | 8 | 8 |
| `W_joins` | 0 | 4 | 5 |
| `W_res` | 1 | 1 | 1 |
| `W_cityGrid` | 0 | 0.5 | 1 |
| `W_room` | 0 | 0.25 | 0.25 |
| `W_seed` | 0 | 0.3 | 0.4 |
| `W_coast` | 0 | 1 | 1.5 |
| `W_exposed` | 0 | 0.5 | 0.5 |
| `W_deny` | 0 | 0 | 1 |

With `W_adj = 8`, one owned neighbour outweighs gold plus any open-space and
coast bonus, so Medium and Hard always keep their land contiguous when they
can, and take adjacent gold, then grassland, then other resources. The first
pick (no owned territories yet) is driven by resource value plus the value of
the unclaimed territories around it, which gives a gold or grassland seed in a
resource-rich area. Easy weights resources over adjacency and adds noise.

## Stockpile Placement and Shipment

Implemented in `internal/game/ai/shipment.go`, around an **expected loss**
estimate:

- `threat(t)` is the strongest attack a single opponent could make on an owned
  territory: its adjacent strength plus the best unit it could bring.
- `expectedLoss = Σ winProbability(threat(t), defense(t)) * value(t)`, where
  `value` is 1 + resource value, +6 for a city, +1.5 weapon, +0.5 horse, +1 per
  boat, and +10 where the stockpile is.

Placement: Easy prefers territories with few enemy neighbours (with Easy
noise). Medium/Hard pick the territory least likely to fall, tie-broken by
defence margin.

Shipment allows one action per turn (move the stockpile, move one unit, or
pass). Every possible stockpile move, and for Medium/Hard every horse, weapon
and boat move, is scored by how much it lowers expected loss. The CPU passes
unless the chosen action saves at least 0.5. Easy only considers moving the
stockpile. Units are moved defensively only; positioning for the next attack
is left for later tuning.

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

Implemented in `internal/game/ai/conquest.go`. Each option is scored
`chance * value - (1 - chance) * risk`, where risk is the brought unit's value
plus 1 if this is the first attack (losing it forfeits the second).

| Win chance needed | Easy | Medium | Hard |
|-------------------|------|--------|------|
| Ordinary target | 0.75 | 0.6 | 0.65 |
| Stockpile / city (value ≥ 9) | 0.75 | 0.6 | 0.5 |
| Stockpile or city of a player one city from winning | 0.75 | 0.5 | 0.4 |

- Easy never brings units. Medium and Hard consider every reinforcement the
  attack planner offers (carrying any weapon or horse that adds strength), and
  coastal targets reachable only by boat.
- Medium/Hard never bring a unit whose loss would make their stockpile
  territory more likely to fall.
- Allies: `GameState.SortAllies` (shared with human attacks) sorts third parties
  by alliance setting. Fixed allies count in the strength calculation and the
  battle. Humans on "ask" are not prompted during a CPU attack (the CPU acts
  under the game lock and a vote can take 60 seconds), so they count as
  neutral.

### Card mode

Implemented in `internal/game/ai/cards.go`. Hand sizes are public, so CPUs use
card counts but never card contents.

- Attack: Medium/Hard assume +1 defence per card the defender holds (max 3),
  then add their strongest attack cards (Rally Cavalry, Arsenal, Naval
  Bombardment and Double Attack valued for this battle) until the attack
  reaches the win threshold. They add Safe Retreat when bringing a unit and the
  win isn't certain, and Blitz when playing two or more cards. Each card played
  costs 0.5 in the attack score. Easy ignores defence cards and plays its whole
  hand.
- Defence: Easy plays one random card. Medium defends only territory worth ≥ 6
  (stockpile, city, gold); Hard anything worth ≥ 3. Both assume +1 attack per
  attack card played, use Bribe first when they have 3 gold, then add their
  strongest cards until their chance of losing is ≤ 25%. They keep all their
  cards if those can't improve the outcome.
- Buying (up to 2 per turn): Medium/Hard keep one of each city resource (three
  gold when holding a Bribe), spend gold last, and buy defence cards when their
  defence hand is smaller or their expected loss is high. Easy spends its most
  plentiful resource.

## Development

Implemented in `internal/game/ai/development.go`. `PlanDevelopment` returns
one build at a time, and the server keeps building until it returns nil. Each
affordable build is scored (resources are used when they suffice, otherwise
gold):

```
city(t)   = 4 + 1.5 * value of owned resources it newly doubles (t and neighbours,
                      not grassland, not already next to a city)
          + 0.5 * ownedNeighbours(t)
          + 20 if this city reaches VictoryCities
          - 6 * chance t is captured
weapon(t) = drop in expected loss + 0.75 per enemy neighbour (max 3)
boat(t)   = drop in expected loss
          + 1 + 0.25 per enemy coastal territory it opens up (max 4), +1 if no boat
            on that water yet
```

Paying gold for a weapon or boat costs 1.5 points.

- Easy: no minimum score, Easy noise, so it often wastes resources.
- Medium: builds anything worth ≥ 1, best first (Medium noise).
- Hard: like Medium, but when exactly one resource short of a city it skips
  weapons and boats that would delay the city unless they are worth ≥ 3.

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
