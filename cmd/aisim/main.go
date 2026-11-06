// Command aisim plays CPU-vs-CPU games without a server and reports how each difficulty
// performs. Use it to check that Hard beats Medium and Medium beats Easy when tuning
// the AI weights in internal/game/ai.
//
//	go run ./cmd/aisim -games 200 -players hard,medium
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"lords-of-conquest/internal/game"
	"lords-of-conquest/internal/game/ai/sim"
	"lords-of-conquest/internal/protocol"
	"lords-of-conquest/pkg/maps"
)

func main() {
	games := flag.Int("games", 100, "number of games to play")
	players := flag.String("players", "easy,medium,hard", "comma-separated difficulty of each CPU")
	chance := flag.String("chance", "medium", "chance level: low, medium or high")
	cards := flag.Bool("cards", false, "use card combat")
	cities := flag.Int("cities", protocol.DefaultVictoryCities, "cities needed to win")
	rounds := flag.Int("rounds", 60, "rounds before a game counts as a draw")
	width := flag.Int("width", 40, "map width (20-60)")
	territories := flag.Int("territories", 50, "target territory count (24-120)")
	islands := flag.Int("islands", 2, "island spread (1-5)")
	resources := flag.Int("resources", 60, "resource coverage percentage (10-100)")
	seed := flag.Int64("seed", time.Now().UnixNano(), "seed for CPU decisions")
	workers := flag.Int("workers", runtime.NumCPU(), "games to play in parallel")
	flag.Parse()

	var difficulties []game.AIDifficulty
	for _, name := range strings.Split(*players, ",") {
		d := game.AIDifficulty(strings.TrimSpace(strings.ToLower(name)))
		if d != game.AIEasy && d != game.AIMedium && d != game.AIHard {
			fail("unknown difficulty %q", name)
		}
		difficulties = append(difficulties, d)
	}
	if len(difficulties) < protocol.MinPlayers || len(difficulties) > protocol.MaxPlayers {
		fail("need %d-%d players", protocol.MinPlayers, protocol.MaxPlayers)
	}

	chanceLevels := map[string]game.ChanceLevel{"low": game.ChanceLow, "medium": game.ChanceMedium, "high": game.ChanceHigh}
	chanceLevel, ok := chanceLevels[*chance]
	if !ok {
		fail("unknown chance level %q", *chance)
	}
	mode := game.CombatModeClassic
	if *cards {
		mode = game.CombatModeCards
	}

	// The game package logs every phase change
	log.SetOutput(io.Discard)

	cfg := sim.Config{
		Difficulties:  difficulties,
		ChanceLevel:   chanceLevel,
		CombatMode:    mode,
		VictoryCities: *cities,
		MaxRounds:     *rounds,
		Map: maps.GeneratorOptions{
			Width: *width, Territories: *territories, WaterBorder: true,
			Islands: *islands, Resources: *resources,
		},
	}

	start := time.Now()
	results := make([]sim.Result, *games)
	errs := make([]error, *games)
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range max(1, *workers) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				c := cfg
				c.Seed = *seed + int64(i)
				results[i], errs[i] = sim.Run(c)
			}
		}()
	}
	for i := range *games {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	report(os.Stdout, results, errs, time.Since(start))
}

type tally struct {
	seats, wins, attacks, attacksWon, citiesBuilt, eliminated int
}

func report(w io.Writer, results []sim.Result, errs []error, elapsed time.Duration) {
	byDifficulty := make(map[game.AIDifficulty]*tally)
	played, draws, rounds, failed := 0, 0, 0, 0
	for i, r := range results {
		if errs[i] != nil {
			failed++
			fmt.Fprintf(w, "game %d failed: %v\n", i+1, errs[i])
			continue
		}
		played++
		rounds += r.Rounds
		if r.WinnerID == "" {
			draws++
		}
		for _, p := range r.Players {
			t := byDifficulty[p.Difficulty]
			if t == nil {
				t = &tally{}
				byDifficulty[p.Difficulty] = t
			}
			t.seats++
			if p.ID == r.WinnerID {
				t.wins++
			}
			t.attacks += p.Attacks
			t.attacksWon += p.AttacksWon
			t.citiesBuilt += p.CitiesBuilt
			if p.Eliminated {
				t.eliminated++
			}
		}
	}
	if played == 0 {
		fmt.Fprintln(w, "no games completed")
		return
	}

	decided := played - draws
	fmt.Fprintf(w, "%d games in %s: %d decided, %d draws, %d failed, %.1f rounds on average\n\n",
		played+failed, elapsed.Round(time.Millisecond), decided, draws, failed, float64(rounds)/float64(played))

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "difficulty\tseats\twins\twin rate\tshare of wins\tattacks/game\tattack success\tcities built/game\teliminated\t")
	for _, d := range []game.AIDifficulty{game.AIEasy, game.AIMedium, game.AIHard} {
		t := byDifficulty[d]
		if t == nil {
			continue
		}
		fmt.Fprintf(tw, "%s\t%d\t%d\t%s\t%s\t%.1f\t%s\t%.1f\t%s\t\n",
			d.Label(), t.seats, t.wins,
			percent(t.wins, t.seats), percent(t.wins, decided),
			ratio(t.attacks, t.seats), percent(t.attacksWon, t.attacks),
			ratio(t.citiesBuilt, t.seats), percent(t.eliminated, t.seats))
	}
	tw.Flush()
}

func percent(n, d int) string {
	if d == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(n)/float64(d))
}

func ratio(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "aisim: "+format+"\n", args...)
	os.Exit(2)
}
