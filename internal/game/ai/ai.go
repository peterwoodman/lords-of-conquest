// Package ai makes decisions for CPU players. Functions read game state and return
// a decision. Some try out moves on the state while evaluating them, but always leave
// it as they found it; callers must hold the game lock.
package ai

import (
	"math/rand"
	"sort"

	"lords-of-conquest/internal/game"
)

// option is a candidate decision with its score (higher is better).
type option struct {
	id    string
	score float64
}

// mediumMargin is how far below the best score Medium will still consider an option.
const mediumMargin = 2.0

// choose picks an option according to difficulty:
//   - Easy: 25% of the time anything, otherwise a random option from the top half.
//   - Medium: one of the top three options scoring within mediumMargin of the best,
//     weighted toward the best.
//   - Hard: the best option.
//
// Ties are broken randomly. Returns "" if there are no options.
func choose(opts []option, d game.AIDifficulty, rng *rand.Rand) string {
	if len(opts) == 0 {
		return ""
	}
	ranked := rank(opts, rng)

	switch game.ParseAIDifficulty(string(d)) {
	case game.AIEasy:
		if rng.Float64() < 0.25 {
			return ranked[rng.Intn(len(ranked))].id
		}
		top := (len(ranked) + 1) / 2
		return ranked[rng.Intn(top)].id
	case game.AIHard:
		return ranked[0].id
	default:
		weights := []float64{0.6, 0.3, 0.1}
		var candidates []option
		for i, o := range ranked {
			if i >= len(weights) || o.score < ranked[0].score-mediumMargin {
				break
			}
			candidates = append(candidates, o)
		}
		total := 0.0
		for i := range candidates {
			total += weights[i]
		}
		r := rng.Float64() * total
		for i, o := range candidates {
			r -= weights[i]
			if r < 0 {
				return o.id
			}
		}
		return candidates[0].id
	}
}

// rank returns the options sorted best first, with ties in random order.
func rank(opts []option, rng *rand.Rand) []option {
	ranked := make([]option, len(opts))
	copy(ranked, opts)
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].id < ranked[j].id })
	rng.Shuffle(len(ranked), func(i, j int) { ranked[i], ranked[j] = ranked[j], ranked[i] })
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	return ranked
}
