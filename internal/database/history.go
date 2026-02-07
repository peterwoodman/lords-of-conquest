package database

import "time"

// HistoryEvent represents a single game event in the history log.
type HistoryEvent struct {
	ID         int64
	GameID     string
	Round      int
	Phase      string
	PlayerID   string
	PlayerName string
	EventType  string
	Message    string
	CreatedAt  time.Time
}

// Event types for game history
const (
	EventTerritorySelected = "territory_selected"
	EventStockpilePlaced   = "stockpile_placed"
	EventStockpileMoved    = "stockpile_moved"
	EventAttackSuccess     = "attack_success"
	EventAttackFailed      = "attack_failed"
	EventProduction        = "production"
	EventBuild             = "build"
	EventPhaseStart        = "phase_start"
	EventRoundStart        = "round_start"
	EventPlayerEliminated  = "player_eliminated"
	EventGameEnd           = "game_end"
	EventTerritoryRenamed  = "territory_renamed"
)

// AddHistoryEvent adds a new event to the game history.
func (db *DB) AddHistoryEvent(gameID string, round int, phase string, playerID, playerName, eventType, message string) error {
	_, err := db.conn.Exec(`
		INSERT INTO game_history (game_id, round, phase, player_id, player_name, event_type, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, round, phase, playerID, playerName, eventType, message, time.Now())
	return err
}

// GetGameHistory retrieves all history events for a game, ordered chronologically.
func (db *DB) GetGameHistory(gameID string) ([]*HistoryEvent, error) {
	rows, err := db.conn.Query(`
		SELECT id, game_id, round, phase, player_id, player_name, event_type, message, created_at
		FROM game_history
		WHERE game_id = ?
		ORDER BY id ASC
	`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*HistoryEvent
	for rows.Next() {
		e := &HistoryEvent{}
		if err := rows.Scan(&e.ID, &e.GameID, &e.Round, &e.Phase, &e.PlayerID, &e.PlayerName, &e.EventType, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// GetGameHistorySince retrieves history events after a given ID (for incremental updates).
func (db *DB) GetGameHistorySince(gameID string, afterID int64) ([]*HistoryEvent, error) {
	rows, err := db.conn.Query(`
		SELECT id, game_id, round, phase, player_id, player_name, event_type, message, created_at
		FROM game_history
		WHERE game_id = ? AND id > ?
		ORDER BY id ASC
	`, gameID, afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*HistoryEvent
	for rows.Next() {
		e := &HistoryEvent{}
		if err := rows.Scan(&e.ID, &e.GameID, &e.Round, &e.Phase, &e.PlayerID, &e.PlayerName, &e.EventType, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// ClearGameHistory deletes all history for a game (used when game is deleted).
func (db *DB) ClearGameHistory(gameID string) error {
	_, err := db.conn.Exec(`DELETE FROM game_history WHERE game_id = ?`, gameID)
	return err
}
