package sim

import (
	"io"
	"log"
	"os"
	"testing"

	"lords-of-conquest/internal/game"
	"lords-of-conquest/pkg/maps"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestRun_GamesFinish(t *testing.T) {
	for _, mode := range []game.CombatMode{game.CombatModeClassic, game.CombatModeCards} {
		for seed := int64(0); seed < 5; seed++ {
			r, err := Run(Config{
				Difficulties:  []game.AIDifficulty{game.AIEasy, game.AIMedium, game.AIHard},
				ChanceLevel:   game.ChanceMedium,
				CombatMode:    mode,
				VictoryCities: 4,
				MaxRounds:     40,
				Seed:          seed,
				Map:           maps.GeneratorOptions{Width: 30, Territories: 30, WaterBorder: true, Islands: 2, Resources: 60},
			})
			if err != nil {
				t.Fatalf("mode %d seed %d: %v", mode, seed, err)
			}
			if r.Rounds < 1 || len(r.Players) != 3 {
				t.Fatalf("mode %d seed %d: unexpected result %+v", mode, seed, r)
			}
			if w := r.Winner(); w != nil && w.Eliminated {
				t.Fatalf("mode %d seed %d: winner %s is eliminated", mode, seed, w.ID)
			}
		}
	}
}
