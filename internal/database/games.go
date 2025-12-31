package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GameStatus represents the current status of a game.
type GameStatus string

const (
	GameStatusWaiting  GameStatus = "waiting"  // In lobby, waiting for players
	GameStatusStarted  GameStatus = "started"  // Game in progress
	GameStatusFinished GameStatus = "finished" // Game completed
)

// GameInfo contains basic game information for listings.
type GameInfo struct {
	ID           string
	Name         string
	JoinCode     string
	IsPublic     bool
	Status       GameStatus
	HostPlayerID string
	PlayerCount  int
	MaxPlayers   int
	CreatedAt    time.Time
}

// Game contains full game data.
type Game struct {
	GameInfo
	Settings  GameSettings
	StartedAt *time.Time
	EndedAt   *time.Time
}

// GameSettings contains configurable game parameters.
type GameSettings struct {
	MaxPlayers    int    `json:"max_players"`
	GameLevel     string `json:"game_level"`
	ChanceLevel   string `json:"chance_level"`
	VictoryCities int    `json:"victory_cities"`
	MapID         string `json:"map_id"`
}

// GamePlayer represents a player in a game.
type GamePlayer struct {
	GameID          string
	PlayerID        string
	PlayerName      string
	Slot            int
	Color           string
	IsAI            bool
	AIPersonality   string
	IsReady         bool
	IsConnected     bool
	JoinedAt        time.Time
	AllianceSetting string // "ask", "neutral", "defender", or a player_id
}

// ErrGameNotFound is returned when a game is not found.
var ErrGameNotFound = errors.New("game not found")

// ErrJoinCodeNotFound is returned when a join code is invalid.
var ErrJoinCodeNotFound = errors.New("invalid join code")

// ErrGameFull is returned when a game has reached max players.
var ErrGameFull = errors.New("game is full")

// ErrAlreadyInGame is returned when player is already in the game.
var ErrAlreadyInGame = errors.New("already in game")

// CreateGame creates a new game.
func (db *DB) CreateGame(name string, hostPlayerID string, settings GameSettings, isPublic bool, mapJSON string) (*Game, error) {
	id := uuid.New().String()
	joinCode := generateJoinCode()

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = db.conn.Exec(`
		INSERT INTO games (id, name, join_code, is_public, status, host_player_id, settings_json, max_players, map_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, name, joinCode, isPublic, GameStatusWaiting, hostPlayerID, string(settingsJSON), settings.MaxPlayers, mapJSON, now)
	if err != nil {
		return nil, err
	}

	return &Game{
		GameInfo: GameInfo{
			ID:           id,
			Name:         name,
			JoinCode:     joinCode,
			IsPublic:     isPublic,
			Status:       GameStatusWaiting,
			HostPlayerID: hostPlayerID,
			PlayerCount:  0,
			MaxPlayers:   settings.MaxPlayers,
			CreatedAt:    now,
		},
		Settings: settings,
	}, nil
}

// GetGame retrieves a game by ID.
func (db *DB) GetGame(id string) (*Game, error) {
	var g Game
	var settingsJSON string
	var joinCode sql.NullString
	var startedAt, endedAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, name, join_code, is_public, status, host_player_id, settings_json, 
		       max_players, created_at, started_at, ended_at
		FROM games WHERE id = ?
	`, id).Scan(&g.ID, &g.Name, &joinCode, &g.IsPublic, &g.Status, &g.HostPlayerID,
		&settingsJSON, &g.MaxPlayers, &g.CreatedAt, &startedAt, &endedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGameNotFound
	}
	if err != nil {
		return nil, err
	}

	if joinCode.Valid {
		g.JoinCode = joinCode.String
	}
	if startedAt.Valid {
		g.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		g.EndedAt = &endedAt.Time
	}

	if err := json.Unmarshal([]byte(settingsJSON), &g.Settings); err != nil {
		return nil, err
	}

	// Get player count
	db.conn.QueryRow(`SELECT COUNT(*) FROM game_players WHERE game_id = ?`, id).Scan(&g.PlayerCount)

	return &g, nil
}

// GetGameByJoinCode retrieves a game by its join code.
func (db *DB) GetGameByJoinCode(code string) (*Game, error) {
	var id string
	err := db.conn.QueryRow(`SELECT id FROM games WHERE join_code = ?`, strings.ToUpper(code)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJoinCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return db.GetGame(id)
}

// ListPublicGames returns all public games that are waiting for players.
func (db *DB) ListPublicGames() ([]*GameInfo, error) {
	rows, err := db.conn.Query(`
		SELECT g.id, g.name, g.join_code, g.is_public, g.status, 
		       g.host_player_id, g.max_players, g.created_at,
		       (SELECT COUNT(*) FROM game_players WHERE game_id = g.id) as player_count
		FROM games g
		WHERE g.is_public = TRUE AND g.status = ?
		ORDER BY g.created_at DESC
	`, GameStatusWaiting)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*GameInfo
	for rows.Next() {
		var g GameInfo
		var joinCode sql.NullString
		if err := rows.Scan(&g.ID, &g.Name, &joinCode, &g.IsPublic, &g.Status,
			&g.HostPlayerID, &g.MaxPlayers, &g.CreatedAt, &g.PlayerCount); err != nil {
			return nil, err
		}
		if joinCode.Valid {
			g.JoinCode = joinCode.String
		}

		games = append(games, &g)
	}
	return games, rows.Err()
}

// JoinGame adds a player to a game.
func (db *DB) JoinGame(gameID, playerID, color string) error {
	// Get game to check status and capacity
	game, err := db.GetGame(gameID)
	if err != nil {
		return err
	}

	if game.Status != GameStatusWaiting {
		return errors.New("game already started")
	}

	// Check if player already in game
	var exists int
	db.conn.QueryRow(`SELECT COUNT(*) FROM game_players WHERE game_id = ? AND player_id = ?`,
		gameID, playerID).Scan(&exists)
	if exists > 0 {
		return ErrAlreadyInGame
	}

	// Check capacity
	if game.PlayerCount >= game.MaxPlayers {
		return ErrGameFull
	}

	// Get next slot
	var maxSlot sql.NullInt64
	db.conn.QueryRow(`SELECT MAX(slot) FROM game_players WHERE game_id = ?`, gameID).Scan(&maxSlot)
	slot := 0
	if maxSlot.Valid {
		slot = int(maxSlot.Int64) + 1
	}

	_, err = db.conn.Exec(`
		INSERT INTO game_players (game_id, player_id, slot, color, is_ai, is_ready, is_connected, joined_at)
		VALUES (?, ?, ?, ?, FALSE, FALSE, FALSE, ?)
	`, gameID, playerID, slot, color, time.Now())
	return err
}

// LeaveGame removes a player from a game.
func (db *DB) LeaveGame(gameID, playerID string) error {
	result, err := db.conn.Exec(`
		DELETE FROM game_players WHERE game_id = ? AND player_id = ?
	`, gameID, playerID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("player not in game")
	}
	return nil
}

// GetGamePlayers returns all players in a game.
func (db *DB) GetGamePlayers(gameID string) ([]*GamePlayer, error) {
	rows, err := db.conn.Query(`
		SELECT gp.game_id, gp.player_id, p.name, gp.slot, gp.color, 
		       gp.is_ai, gp.ai_personality, gp.is_ready, gp.is_connected, gp.joined_at,
		       COALESCE(gp.alliance_setting, 'ask')
		FROM game_players gp
		LEFT JOIN players p ON gp.player_id = p.id
		WHERE gp.game_id = ?
		ORDER BY gp.slot
	`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []*GamePlayer
	for rows.Next() {
		var gp GamePlayer
		var aiPersonality sql.NullString
		var playerName sql.NullString
		if err := rows.Scan(&gp.GameID, &gp.PlayerID, &playerName, &gp.Slot, &gp.Color,
			&gp.IsAI, &aiPersonality, &gp.IsReady, &gp.IsConnected, &gp.JoinedAt,
			&gp.AllianceSetting); err != nil {
			return nil, err
		}
		if aiPersonality.Valid {
			gp.AIPersonality = aiPersonality.String
		}
		if playerName.Valid {
			gp.PlayerName = playerName.String
		} else {
			gp.PlayerName = "AI"
		}
		players = append(players, &gp)
	}
	return players, rows.Err()
}

// SetPlayerReady sets a player's ready status.
func (db *DB) SetPlayerReady(gameID, playerID string, ready bool) error {
	_, err := db.conn.Exec(`
		UPDATE game_players SET is_ready = ? WHERE game_id = ? AND player_id = ?
	`, ready, gameID, playerID)
	return err
}

// SetPlayerConnected sets a player's connection status.
func (db *DB) SetPlayerConnected(gameID, playerID string, connected bool) error {
	_, err := db.conn.Exec(`
		UPDATE game_players SET is_connected = ? WHERE game_id = ? AND player_id = ?
	`, connected, gameID, playerID)
	return err
}

// SetAllianceSetting sets a player's alliance preference.
// setting can be "ask", "neutral", "defender", or a player_id
func (db *DB) SetAllianceSetting(gameID, playerID, setting string) error {
	_, err := db.conn.Exec(`
		UPDATE game_players SET alliance_setting = ? WHERE game_id = ? AND player_id = ?
	`, setting, gameID, playerID)
	return err
}

// GetAllianceSetting gets a player's alliance preference.
func (db *DB) GetAllianceSetting(gameID, playerID string) (string, error) {
	var setting string
	err := db.conn.QueryRow(`
		SELECT COALESCE(alliance_setting, 'ask') FROM game_players 
		WHERE game_id = ? AND player_id = ?
	`, gameID, playerID).Scan(&setting)
	if err != nil {
		return "ask", err
	}
	return setting, nil
}

// AddAIPlayer adds an AI player to a game.
func (db *DB) AddAIPlayer(gameID, color, personality string) error {
	game, err := db.GetGame(gameID)
	if err != nil {
		return err
	}

	if game.PlayerCount >= game.MaxPlayers {
		return ErrGameFull
	}

	// Get next slot
	var maxSlot sql.NullInt64
	db.conn.QueryRow(`SELECT MAX(slot) FROM game_players WHERE game_id = ?`, gameID).Scan(&maxSlot)
	slot := 0
	if maxSlot.Valid {
		slot = int(maxSlot.Int64) + 1
	}

	// Create a player entry for the AI (required for foreign key)
	aiID := fmt.Sprintf("ai-%s", uuid.New().String()[:8])
	aiName := fmt.Sprintf("AI (%s)", personality)
	aiToken := uuid.New().String()
	
	_, err = db.conn.Exec(`
		INSERT INTO players (id, token, name, created_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?)
	`, aiID, aiToken, aiName, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create AI player: %w", err)
	}

	// Add AI to game
	_, err = db.conn.Exec(`
		INSERT INTO game_players (game_id, player_id, slot, color, is_ai, ai_personality, is_ready, is_connected, joined_at)
		VALUES (?, ?, ?, ?, TRUE, ?, TRUE, TRUE, ?)
	`, gameID, aiID, slot, color, personality, time.Now())
	return err
}

// StartGame marks a game as started.
func (db *DB) StartGame(gameID string) error {
	now := time.Now()
	_, err := db.conn.Exec(`
		UPDATE games SET status = ?, started_at = ? WHERE id = ?
	`, GameStatusStarted, now, gameID)
	return err
}

// EndGame marks a game as finished.
func (db *DB) EndGame(gameID string) error {
	now := time.Now()
	_, err := db.conn.Exec(`
		UPDATE games SET status = ?, ended_at = ? WHERE id = ?
	`, GameStatusFinished, now, gameID)
	return err
}

// SaveGameState saves the current game state.
func (db *DB) SaveGameState(gameID string, stateJSON string, currentPlayerID string, round int, phase string) error {
	_, err := db.conn.Exec(`
		INSERT INTO game_state (game_id, state_json, current_player_id, round, phase, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(game_id) DO UPDATE SET
			state_json = excluded.state_json,
			current_player_id = excluded.current_player_id,
			round = excluded.round,
			phase = excluded.phase,
			updated_at = excluded.updated_at
	`, gameID, stateJSON, currentPlayerID, round, phase, time.Now())
	return err
}

// GetGameState retrieves the current game state.
func (db *DB) GetGameState(gameID string) (string, error) {
	var stateJSON string
	err := db.conn.QueryRow(`
		SELECT state_json FROM game_state WHERE game_id = ?
	`, gameID).Scan(&stateJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return stateJSON, err
}

// GetGameMapJSON retrieves the stored map JSON for a game.
func (db *DB) GetGameMapJSON(gameID string) (string, error) {
	var mapJSON sql.NullString
	err := db.conn.QueryRow(`
		SELECT map_json FROM games WHERE id = ?
	`, gameID).Scan(&mapJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrGameNotFound
	}
	if err != nil {
		return "", err
	}
	if mapJSON.Valid {
		return mapJSON.String, nil
	}
	return "", nil
}

// LogAction logs a game action.
func (db *DB) LogAction(gameID, playerID, actionType, actionJSON, resultJSON string) error {
	_, err := db.conn.Exec(`
		INSERT INTO game_actions (game_id, player_id, action_type, action_json, result_json)
		VALUES (?, ?, ?, ?, ?)
	`, gameID, playerID, actionType, actionJSON, resultJSON)
	return err
}

// DeleteGame permanently deletes a game and all associated data.
func (db *DB) DeleteGame(gameID string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete in order of dependencies
	_, err = tx.Exec(`DELETE FROM game_actions WHERE game_id = ?`, gameID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM game_state WHERE game_id = ?`, gameID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM game_players WHERE game_id = ?`, gameID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM games WHERE id = ?`, gameID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// CleanupAbandonedLobbies removes lobby games where the host is offline.
func (db *DB) CleanupAbandonedLobbies() error {
	// Delete games that are in waiting status and have no connected players
	_, err := db.conn.Exec(`
		DELETE FROM games 
		WHERE id IN (
			SELECT g.id FROM games g
			WHERE g.status = ?
			AND NOT EXISTS (
				SELECT 1 FROM game_players gp
				WHERE gp.game_id = g.id
				AND gp.is_connected = 1
			)
		)
	`, GameStatusWaiting)
	return err
}

// GetPlayerGames retrieves all games a player is participating in.
func (db *DB) GetPlayerGames(playerID string) ([]*GameInfo, error) {
	rows, err := db.conn.Query(`
		SELECT DISTINCT
			g.id, g.name, g.join_code, g.is_public, g.status,
			g.host_player_id, g.max_players, g.created_at,
			(SELECT COUNT(*) FROM game_players WHERE game_id = g.id) as player_count
		FROM games g
		INNER JOIN game_players gp ON gp.game_id = g.id
		WHERE gp.player_id = ?
		AND g.status != ?
		ORDER BY g.created_at DESC
	`, playerID, GameStatusFinished)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*GameInfo
	for rows.Next() {
		game := &GameInfo{}
		err := rows.Scan(
			&game.ID,
			&game.Name,
			&game.JoinCode,
			&game.IsPublic,
			&game.Status,
			&game.HostPlayerID,
			&game.MaxPlayers,
			&game.CreatedAt,
			&game.PlayerCount,
		)
		if err != nil {
			return nil, err
		}
		games = append(games, game)
	}

	return games, rows.Err()
}

// generateJoinCode creates a human-readable join code.
func generateJoinCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Removed ambiguous chars (0,O,1,I)
	bytes := make([]byte, 8)
	rand.Read(bytes)

	code := make([]byte, 8)
	for i := range code {
		code[i] = chars[bytes[i]%byte(len(chars))]
	}
	// Format as XXXX-XXXX
	return string(code[:4]) + "-" + string(code[4:])
}

