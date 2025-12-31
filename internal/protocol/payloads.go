package protocol

// ==================== Authentication Payloads ====================

// AuthenticatePayload is sent to authenticate/register a player.
type AuthenticatePayload struct {
	Token string `json:"token,omitempty"` // Existing token for returning players
	Name  string `json:"name"`            // Display name
}

// AuthResultPayload is the response to authentication.
type AuthResultPayload struct {
	Success  bool   `json:"success"`
	PlayerID string `json:"player_id"`
	Token    string `json:"token"` // Save this for reconnecting
	Name     string `json:"name"`
	Error    string `json:"error,omitempty"`
}

// ==================== Lobby Payloads ====================

// CreateGamePayload is sent to create a new game.
type CreateGamePayload struct {
	Name     string       `json:"name"`
	IsPublic bool         `json:"is_public"`
	Settings GameSettings `json:"settings"`
	MapData  *MapData     `json:"map_data,omitempty"` // Generated map data
}

// MapData contains the full map information for generated maps.
type MapData struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Width       int                 `json:"width"`
	Height      int                 `json:"height"`
	Grid        [][]int             `json:"grid"`
	Territories map[string]TerritoryInfo `json:"territories"`
}

// TerritoryInfo contains territory metadata.
type TerritoryInfo struct {
	Name     string `json:"name"`
	Resource string `json:"resource,omitempty"`
}

// GameCreatedPayload is the response when a game is created.
type GameCreatedPayload struct {
	GameID   string `json:"game_id"`
	JoinCode string `json:"join_code"`
}

// GameSettings are the configurable game parameters.
type GameSettings struct {
	MaxPlayers    int    `json:"max_players"`
	GameLevel     string `json:"game_level"`     // beginner, intermediate, advanced, expert
	ChanceLevel   string `json:"chance_level"`   // low, medium, high
	VictoryCities int    `json:"victory_cities"` // 3-8
	MapID         string `json:"map_id"`
}

// JoinGamePayload is sent to join a game by ID.
type JoinGamePayload struct {
	GameID         string `json:"game_id"`
	PreferredColor string `json:"preferred_color,omitempty"`
}

// JoinByCodePayload is sent to join a game by join code.
type JoinByCodePayload struct {
	JoinCode       string `json:"join_code"`
	PreferredColor string `json:"preferred_color,omitempty"`
}

// JoinedGamePayload is the response when successfully joining a game.
type JoinedGamePayload struct {
	GameID   string `json:"game_id"`
	JoinCode string `json:"join_code"`
}

// LeaveGamePayload is sent to leave a game.
type LeaveGamePayload struct {
	// No additional fields needed - inferred from client context
}

// DeleteGamePayload is sent to delete a game (creator only).
type DeleteGamePayload struct {
	GameID string `json:"game_id"`
}

// GameDeletedPayload is sent when a game is deleted.
type GameDeletedPayload struct {
	GameID string `json:"game_id"`
	Reason string `json:"reason,omitempty"`
}

// AddAIPayload is sent to add an AI player.
type AddAIPayload struct {
	Personality string `json:"personality"` // aggressive, defensive, passive
}

// RemovePlayerPayload is sent to remove a player (host only).
type RemovePlayerPayload struct {
	PlayerID string `json:"player_id"`
}

// UpdateSettingsPayload is sent to update game settings.
type UpdateSettingsPayload struct {
	Settings GameSettings `json:"settings"`
}

// UpdateSettingPayload is sent to update a single game setting.
type UpdateSettingPayload struct {
	Key   string `json:"key"`   // Setting name: "chanceLevel", "victoryCities", "maxPlayers"
	Value string `json:"value"` // New value as string
}

// PlayerReadyPayload indicates player ready state.
type PlayerReadyPayload struct {
	Ready bool `json:"ready"`
}

// GameListPayload contains a list of games.
type GameListPayload struct {
	Games []GameListItem `json:"games"`
}

// GameListItem is a summary of a game.
type GameListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	JoinCode    string `json:"join_code,omitempty"`
	Status      string `json:"status"`
	PlayerCount int    `json:"player_count"`
	MaxPlayers  int    `json:"max_players"`
	HostName    string `json:"host_name,omitempty"`
	IsYourTurn  bool   `json:"is_your_turn,omitempty"`
}

// YourGamesPayload contains games the player is in.
type YourGamesPayload struct {
	Games []GameListItem `json:"games"`
}

// LobbyStatePayload contains the current lobby state.
type LobbyStatePayload struct {
	GameID   string        `json:"game_id"`
	GameName string        `json:"game_name"`
	JoinCode string        `json:"join_code"`
	HostID   string        `json:"host_id"`
	IsPublic bool          `json:"is_public"`
	Settings GameSettings  `json:"settings"`
	Players  []LobbyPlayer `json:"players"`
}

// LobbyPlayer is a player in the lobby.
type LobbyPlayer struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Color         string `json:"color"`
	IsAI          bool   `json:"is_ai"`
	AIPersonality string `json:"ai_personality,omitempty"`
	Ready         bool   `json:"ready"`
	IsConnected   bool   `json:"is_connected"`
}

// PlayerJoinedPayload is sent when a player joins.
type PlayerJoinedPayload struct {
	PlayerID string `json:"player_id"`
	Name     string `json:"name"`
}

// PlayerLeftPayload is sent when a player leaves.
type PlayerLeftPayload struct {
	PlayerID string `json:"player_id"`
}

// ==================== Game Flow Payloads ====================

// GameStartedPayload is sent when the game begins.
type GameStartedPayload struct {
	GameID string `json:"game_id"`
}

// PhaseChangedPayload is sent when the phase changes.
type PhaseChangedPayload struct {
	Phase         string `json:"phase"`
	Round         int    `json:"round"`
	Skipped       bool   `json:"skipped"`
	CurrentPlayer string `json:"current_player"`
}

// TurnChangedPayload is sent when the active player changes.
type TurnChangedPayload struct {
	CurrentPlayer string `json:"current_player"`
	TimeLimit     int    `json:"time_limit,omitempty"`
}

// ActionResultPayload is the result of a player action.
type ActionResultPayload struct {
	ActionID    string      `json:"action_id"`
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	StateUpdate interface{} `json:"state_update,omitempty"`
}

// GameStatePayload contains the full game state.
type GameStatePayload struct {
	State interface{} `json:"state"`
}

// GameHistoryPayload contains game history events.
type GameHistoryPayload struct {
	Events []HistoryEvent `json:"events"`
}

// HistoryEvent is a single event in the game history log.
type HistoryEvent struct {
	ID         int64  `json:"id"`
	Round      int    `json:"round"`
	Phase      string `json:"phase"`
	PlayerID   string `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	EventType  string `json:"event_type"`
	Message    string `json:"message"`
}

// GameEndedPayload is sent when the game concludes.
type GameEndedPayload struct {
	WinnerID   string      `json:"winner_id"`
	WinnerName string      `json:"winner_name"`
	Reason     string      `json:"reason"` // "cities", "elimination", "surrender"
	FinalState interface{} `json:"final_state"`
}

// PhaseSkippedPayload is sent when a phase is skipped due to chance.
type PhaseSkippedPayload struct {
	Phase  string `json:"phase"`  // The phase that was skipped
	Reason string `json:"reason"` // Funny reason for skipping
}

// ==================== Action Payloads ====================

// SelectTerritoryPayload selects a territory.
type SelectTerritoryPayload struct {
	TerritoryID string `json:"territory_id"`
}

// PlaceStockpilePayload places the initial stockpile.
type PlaceStockpilePayload struct {
	TerritoryID string `json:"territory_id"`
}

// ProposeTradePayload proposes a trade.
type ProposeTradePayload struct {
	TargetPlayer string        `json:"target_player"`
	Offer        ResourceCount `json:"offer"`
	Request      ResourceCount `json:"request"`
}

// ResourceCount represents amounts of each resource.
type ResourceCount struct {
	Coal   int `json:"coal"`
	Gold   int `json:"gold"`
	Iron   int `json:"iron"`
	Timber int `json:"timber"`
	Horses int `json:"horses"`
}

// RespondTradePayload responds to a trade offer.
type RespondTradePayload struct {
	TradeID  string `json:"trade_id"`
	Accepted bool   `json:"accepted"`
}

// MoveStockpilePayload moves the stockpile.
type MoveStockpilePayload struct {
	Destination string `json:"destination"`
}

// MoveUnitPayload moves a unit.
type MoveUnitPayload struct {
	UnitType    string `json:"unit_type"`
	From        string `json:"from"`
	To          string `json:"to"`
	CarryWeapon bool   `json:"carry_weapon,omitempty"`
}

// PlanAttackPayload begins attack planning.
type PlanAttackPayload struct {
	TargetTerritory string `json:"target_territory"`
}

// AttackPreviewPayload shows combat preview.
type AttackPreviewPayload struct {
	TargetTerritory         string                `json:"target_territory"`
	AttackStrength          int                   `json:"attack_strength"`
	DefenseStrength         int                   `json:"defense_strength"`
	AttackerAllyStrength    int                   `json:"attacker_ally_strength"`  // Strength from allies
	DefenderAllyStrength    int                   `json:"defender_ally_strength"`  // Strength from allies
	CanAttack               bool                  `json:"can_attack"`
	AvailableReinforcements []ReinforcementOption `json:"available_reinforcements"`
}

// ReinforcementOption is a possible unit to bring.
type ReinforcementOption struct {
	UnitType          string `json:"unit_type"`
	From              string `json:"from"`
	WaterBodyID       string `json:"water_body_id,omitempty"` // For boats
	StrengthBonus     int    `json:"strength_bonus"`
	CanCarryWeapon    bool   `json:"can_carry_weapon,omitempty"`
	WeaponAvailableAt string `json:"weapon_available_at,omitempty"`
	CanCarryHorse     bool   `json:"can_carry_horse,omitempty"`
	HorseAvailableAt  string `json:"horse_available_at,omitempty"`
}

// BringForcesPayload adds reinforcement to attack.
type BringForcesPayload struct {
	UnitType       string `json:"unit_type"`
	From           string `json:"from"`
	PickupWeaponAt string `json:"pickup_weapon_at,omitempty"`
	PickupHorseAt  string `json:"pickup_horse_at,omitempty"`
}

// SetAlliancePayload sets the player's alliance preference.
type SetAlliancePayload struct {
	Setting string `json:"setting"` // "ask", "neutral", "defender", or a player_id
}

// AllianceRequestPayload notifies of alliance opportunity during combat.
type AllianceRequestPayload struct {
	BattleID       string `json:"battle_id"`
	AttackerID     string `json:"attacker_id"`
	AttackerName   string `json:"attacker_name"`
	DefenderID     string `json:"defender_id"`
	DefenderName   string `json:"defender_name"`
	TerritoryID    string `json:"territory_id"`
	TerritoryName  string `json:"territory_name"`
	YourStrength   int    `json:"your_strength"`
	TimeLimit      int    `json:"time_limit"` // seconds
	ExpiresAt      int64  `json:"expires_at"` // unix timestamp
}

// AllianceVotePayload is a player's alliance choice.
type AllianceVotePayload struct {
	BattleID string `json:"battle_id"`
	Side     string `json:"side"` // "attacker", "defender", or "neutral"
}

// AllianceResultPayload confirms an alliance vote was received.
type AllianceResultPayload struct {
	BattleID string `json:"battle_id"`
	Accepted bool   `json:"accepted"`
}

// BuildPayload builds a unit or city.
type BuildPayload struct {
	Type        string `json:"type"` // "city", "weapon", or "boat"
	Territory   string `json:"territory"`
	WaterBodyID string `json:"water_body_id,omitempty"` // Required for boats when multiple water bodies available
	UseGold     bool   `json:"use_gold"`
}

// ExecuteAttackPayload executes a planned attack.
type ExecuteAttackPayload struct {
	TargetTerritory string `json:"target_territory"`
	BringUnit       string `json:"bring_unit,omitempty"`     // "horse", "weapon", or "boat"
	BringFrom       string `json:"bring_from,omitempty"`     // Territory ID
	WaterBodyID     string `json:"water_body_id,omitempty"`  // For boats: which water body
	CarryWeapon     bool   `json:"carry_weapon,omitempty"`   // For horse/boat
	WeaponFrom      string `json:"weapon_from,omitempty"`    // Territory ID
	CarryHorse      bool   `json:"carry_horse,omitempty"`    // For boat
	HorseFrom       string `json:"horse_from,omitempty"`     // Territory ID
}

// CombatResultPayload reports the result of combat.
type CombatResultPayload struct {
	Success         bool     `json:"success"`
	AttackerWins    bool     `json:"attacker_wins"`
	AttackStrength  int      `json:"attack_strength"`
	DefenseStrength int      `json:"defense_strength"`
	TargetTerritory string   `json:"target_territory"`
	UnitsDestroyed  []string `json:"units_destroyed,omitempty"`
	UnitsCaptured   []string `json:"units_captured,omitempty"`
}

// ==================== System Payloads ====================

// WelcomePayload is sent on connection.
type WelcomePayload struct {
	ServerVersion string `json:"server_version"`
}

// ReconnectPayload is sent to restore a session.
type ReconnectPayload struct {
	Token  string `json:"token"`
	GameID string `json:"game_id,omitempty"`
}

// DisconnectPayload notifies of a player disconnect.
type DisconnectPayload struct {
	PlayerID string `json:"player_id"`
	Reason   string `json:"reason"`
}
