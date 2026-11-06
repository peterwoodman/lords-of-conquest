package ai

import (
	"math/rand"

	"lords-of-conquest/internal/game"
)

// selectionWeights tunes territory selection scoring.
type selectionWeights struct {
	adjacent  float64 // per owned neighbour; must dominate resource value
	joins     float64 // per extra owned region the territory connects
	resource  float64 // multiplier on resourceValue
	cityGrid  float64 // per owned neighbour with a resource (future city doubling)
	room      float64 // per unclaimed neighbour (space to grow)
	seed      float64 // first pick only: multiplier on unclaimed neighbours' resource value
	coast     float64 // coastal territory when none owned yet
	exposed   float64 // penalty per enemy neighbour
	deny      float64 // per neighbour of the opponent most adjacent to it
}

var selectionProfiles = map[game.AIDifficulty]selectionWeights{
	game.AIEasy: {
		adjacent: 1,
		resource: 1,
	},
	game.AIMedium: {
		adjacent: 8,
		joins:    4,
		resource: 1,
		cityGrid: 0.5,
		room:     0.25,
		seed:     0.3,
		coast:    1,
		exposed:  0.5,
	},
	game.AIHard: {
		adjacent: 8,
		joins:    5,
		resource: 1,
		cityGrid: 1,
		room:     0.25,
		seed:     0.4,
		coast:    1.5,
		exposed:  0.5,
		deny:     1,
	},
}

// ChooseTerritory picks an unclaimed territory during territory selection.
// Returns "" if none are left.
func ChooseTerritory(s *game.GameState, playerID string, d game.AIDifficulty, rng *rand.Rand) string {
	w := selectionProfiles[game.ParseAIDifficulty(string(d))]
	label, _ := clusters(s, playerID)
	ownsAny := len(label) > 0
	hasCoast := ownsCoast(s, playerID)

	var opts []option
	for _, id := range sortedTerritoryIDs(s) {
		t := s.Territories[id]
		if t.Owner != "" {
			continue
		}
		opts = append(opts, option{id: id, score: scoreSelection(s, playerID, t, w, label, ownsAny, hasCoast)})
	}
	return choose(opts, d, rng)
}

func scoreSelection(s *game.GameState, playerID string, t *game.Territory, w selectionWeights,
	label map[string]int, ownsAny, hasCoast bool) float64 {

	score := w.resource * resourceValue(t.Resource)

	regions := make(map[int]bool)
	resourceNeighbours := 0
	opponentNeighbours := make(map[string]int)
	enemies := 0
	unclaimedValue := 0.0
	for _, n := range neighbours(s, t) {
		switch {
		case n.Owner == playerID:
			score += w.adjacent
			regions[label[n.ID]] = true
			if n.Resource != game.ResourceNone {
				resourceNeighbours++
			}
		case n.Owner == "":
			score += w.room
			unclaimedValue += resourceValue(n.Resource)
		default:
			enemies++
			opponentNeighbours[n.Owner]++
		}
	}

	if len(regions) > 1 {
		score += w.joins * float64(len(regions)-1)
	}
	score += w.cityGrid * float64(resourceNeighbours)
	if !ownsAny {
		score += w.seed * unclaimedValue
	}
	if t.IsCoastal() && !hasCoast {
		score += w.coast
	}
	score -= w.exposed * float64(enemies)

	mostAdjacent := 0
	for _, c := range opponentNeighbours {
		mostAdjacent = max(mostAdjacent, c)
	}
	score += w.deny * float64(mostAdjacent)

	return score
}
