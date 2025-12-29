package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Player represents a player in the database.
type Player struct {
	ID         string
	Token      string
	Name       string
	CreatedAt  time.Time
	LastSeenAt time.Time
}

// ErrPlayerNotFound is returned when a player is not found.
var ErrPlayerNotFound = errors.New("player not found")

// CreatePlayer creates a new player with a generated token.
func (db *DB) CreatePlayer(name string) (*Player, error) {
	id := uuid.New().String()
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = db.conn.Exec(`
		INSERT INTO players (id, token, name, created_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, token, name, now, now)
	if err != nil {
		return nil, err
	}

	return &Player{
		ID:         id,
		Token:      token,
		Name:       name,
		CreatedAt:  now,
		LastSeenAt: now,
	}, nil
}

// GetPlayerByToken retrieves a player by their token.
func (db *DB) GetPlayerByToken(token string) (*Player, error) {
	var p Player
	err := db.conn.QueryRow(`
		SELECT id, token, name, created_at, last_seen_at
		FROM players WHERE token = ?
	`, token).Scan(&p.ID, &p.Token, &p.Name, &p.CreatedAt, &p.LastSeenAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPlayerByID retrieves a player by their ID.
func (db *DB) GetPlayerByID(id string) (*Player, error) {
	var p Player
	err := db.conn.QueryRow(`
		SELECT id, token, name, created_at, last_seen_at
		FROM players WHERE id = ?
	`, id).Scan(&p.ID, &p.Token, &p.Name, &p.CreatedAt, &p.LastSeenAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePlayerName updates a player's display name.
func (db *DB) UpdatePlayerName(id, name string) error {
	result, err := db.conn.Exec(`
		UPDATE players SET name = ? WHERE id = ?
	`, name, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPlayerNotFound
	}
	return nil
}

// UpdatePlayerLastSeen updates the last seen timestamp.
func (db *DB) UpdatePlayerLastSeen(id string) error {
	_, err := db.conn.Exec(`
		UPDATE players SET last_seen_at = ? WHERE id = ?
	`, time.Now(), id)
	return err
}

// GetPlayerGames returns all games a player is in.
func (db *DB) GetPlayerGames(playerID string) ([]*GameInfo, error) {
	rows, err := db.conn.Query(`
		SELECT g.id, g.name, g.join_code, g.is_public, g.status, 
		       g.host_player_id, g.created_at,
		       (SELECT COUNT(*) FROM game_players WHERE game_id = g.id) as player_count
		FROM games g
		JOIN game_players gp ON g.id = gp.game_id
		WHERE gp.player_id = ?
		ORDER BY g.created_at DESC
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*GameInfo
	for rows.Next() {
		var g GameInfo
		var joinCode sql.NullString
		if err := rows.Scan(&g.ID, &g.Name, &joinCode, &g.IsPublic, &g.Status,
			&g.HostPlayerID, &g.CreatedAt, &g.PlayerCount); err != nil {
			return nil, err
		}
		if joinCode.Valid {
			g.JoinCode = joinCode.String
		}
		games = append(games, &g)
	}
	return games, rows.Err()
}

// generateToken creates a secure random token.
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

