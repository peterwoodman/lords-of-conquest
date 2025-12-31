package database

type migration struct {
	id   int
	name string
	sql  string
}

var migrations = []migration{
	{
		id:   1,
		name: "initial_schema",
		sql: `
			-- Players table: stores player tokens (no accounts, just tokens)
			CREATE TABLE players (
				id TEXT PRIMARY KEY,
				token TEXT UNIQUE NOT NULL,
				name TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX idx_players_token ON players(token);

			-- Games table: stores game metadata
			CREATE TABLE games (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				join_code TEXT UNIQUE,
				is_public BOOLEAN DEFAULT FALSE,
				status TEXT NOT NULL DEFAULT 'waiting',
				host_player_id TEXT NOT NULL,
				settings_json TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				started_at DATETIME,
				ended_at DATETIME,
				FOREIGN KEY (host_player_id) REFERENCES players(id)
			);
			CREATE INDEX idx_games_join_code ON games(join_code);
			CREATE INDEX idx_games_status ON games(status);
			CREATE INDEX idx_games_public ON games(is_public, status);

			-- Game players: links players to games
			CREATE TABLE game_players (
				game_id TEXT NOT NULL,
				player_id TEXT NOT NULL,
				slot INTEGER NOT NULL,
				color TEXT NOT NULL,
				is_ai BOOLEAN DEFAULT FALSE,
				ai_personality TEXT,
				is_ready BOOLEAN DEFAULT FALSE,
				is_connected BOOLEAN DEFAULT FALSE,
				joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (game_id, player_id),
				FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
				FOREIGN KEY (player_id) REFERENCES players(id)
			);
			CREATE INDEX idx_game_players_game ON game_players(game_id);
			CREATE INDEX idx_game_players_player ON game_players(player_id);

			-- Game state: stores the current game state as JSON
			CREATE TABLE game_state (
				game_id TEXT PRIMARY KEY,
				state_json TEXT NOT NULL,
				current_player_id TEXT,
				round INTEGER DEFAULT 0,
				phase TEXT,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
			);

			-- Game actions: log of all actions for replay/debugging
			CREATE TABLE game_actions (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				game_id TEXT NOT NULL,
				player_id TEXT,
				action_type TEXT NOT NULL,
				action_json TEXT NOT NULL,
				result_json TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
			);
			CREATE INDEX idx_game_actions_game ON game_actions(game_id);
		`,
	},
	{
		id:   2,
		name: "add_max_players_column",
		sql: `
			-- Add max_players column to games table for easier querying
			ALTER TABLE games ADD COLUMN max_players INTEGER DEFAULT 2;
		`,
	},
	{
		id:   3,
		name: "add_map_json_column",
		sql: `
			-- Add map_json column to store generated map data for persistence across restarts
			ALTER TABLE games ADD COLUMN map_json TEXT;
		`,
	},
	{
		id:   4,
		name: "add_game_history",
		sql: `
			-- Game history table: stores chronological game events for display
			CREATE TABLE game_history (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				game_id TEXT NOT NULL,
				round INTEGER NOT NULL,
				phase TEXT NOT NULL,
				player_id TEXT,
				player_name TEXT,
				event_type TEXT NOT NULL,
				message TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
			);
			CREATE INDEX idx_game_history_game ON game_history(game_id);
			CREATE INDEX idx_game_history_game_order ON game_history(game_id, id);
		`,
	},
	{
		id:   5,
		name: "add_alliance_setting",
		sql: `
			-- Add alliance_setting column to game_players
			-- Values: "ask" (default), "neutral", "defender", or a player_id
			ALTER TABLE game_players ADD COLUMN alliance_setting TEXT DEFAULT 'ask';
		`,
	},
	{
		id:   6,
		name: "add_game_winner",
		sql: `
			-- Add winner columns to games table
			ALTER TABLE games ADD COLUMN winner_id TEXT;
			ALTER TABLE games ADD COLUMN win_reason TEXT;
		`,
	},
}
