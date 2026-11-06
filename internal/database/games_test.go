package database

import (
	"errors"
	"path/filepath"
	"testing"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateGame_HostIsConnectedPlayer(t *testing.T) {
	db := newTestDB(t)
	host, _ := db.CreatePlayer("Host")

	game, err := db.CreateGame("g", host.ID, "orange", GameSettings{MaxPlayers: 4}, true, "")
	if err != nil {
		t.Fatal(err)
	}

	// Cleanup right after creation must not remove the new lobby
	if err := db.CleanupAbandonedLobbies(); err != nil {
		t.Fatal(err)
	}
	players, err := db.GetGamePlayers(game.ID)
	if err != nil || len(players) != 1 || !players[0].IsConnected {
		t.Fatalf("expected host as the one connected player, got %v (err %v)", players, err)
	}
}

func TestCleanup_RemovesLobbyWithOnlyAIConnected(t *testing.T) {
	db := newTestDB(t)
	host, _ := db.CreatePlayer("Host")
	game, _ := db.CreateGame("g", host.ID, "orange", GameSettings{MaxPlayers: 4}, true, "")
	if err := db.AddAIPlayer(game.ID, "blue", "medium"); err != nil {
		t.Fatal(err)
	}
	db.SetPlayerConnected(game.ID, host.ID, false)

	if err := db.CleanupAbandonedLobbies(); err != nil {
		t.Fatal(err)
	}

	if _, err := db.GetGame(game.ID); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("expected lobby to be deleted, got %v", err)
	}
	var aiPlayers int
	db.conn.QueryRow(`SELECT COUNT(*) FROM players WHERE id LIKE 'ai-%'`).Scan(&aiPlayers)
	if aiPlayers != 0 {
		t.Fatalf("expected orphaned AI player records to be removed, found %d", aiPlayers)
	}
}

func TestJoinGame_RespectsCapacity(t *testing.T) {
	db := newTestDB(t)
	host, _ := db.CreatePlayer("Host")
	other, _ := db.CreatePlayer("Other")
	game, _ := db.CreateGame("g", host.ID, "orange", GameSettings{MaxPlayers: 2}, true, "")

	if err := db.AddAIPlayer(game.ID, "blue", "medium"); err != nil {
		t.Fatal(err)
	}
	if err := db.JoinGame(game.ID, other.ID, "green"); !errors.Is(err, ErrGameFull) {
		t.Fatalf("expected game full, got %v", err)
	}
}

func TestSetAIDifficulty_OnlyChangesAIPlayers(t *testing.T) {
	db := newTestDB(t)
	host, _ := db.CreatePlayer("Host")
	game, _ := db.CreateGame("g", host.ID, "orange", GameSettings{MaxPlayers: 4}, true, "")
	if err := db.AddAIPlayer(game.ID, "blue", "easy"); err != nil {
		t.Fatal(err)
	}

	aiDifficulty := func() (string, string) {
		players, err := db.GetGamePlayers(game.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range players {
			if p.IsAI {
				return p.PlayerID, p.AIDifficulty
			}
		}
		t.Fatal("no AI player")
		return "", ""
	}

	aiID, difficulty := aiDifficulty()
	if difficulty != "easy" {
		t.Fatalf("expected easy, got %q", difficulty)
	}
	if err := db.SetAIDifficulty(game.ID, aiID, "hard"); err != nil {
		t.Fatal(err)
	}
	if _, difficulty = aiDifficulty(); difficulty != "hard" {
		t.Fatalf("expected hard, got %q", difficulty)
	}
	if err := db.SetAIDifficulty(game.ID, host.ID, "hard"); err == nil {
		t.Fatal("expected error changing a human player's difficulty")
	}
}
