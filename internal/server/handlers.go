package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"lords-of-conquest/internal/database"
	"lords-of-conquest/internal/game"
	"lords-of-conquest/internal/protocol"
	"lords-of-conquest/pkg/maps"
)

// Handlers processes incoming messages.
type Handlers struct {
	hub *Hub
}

// NewHandlers creates a new handler set.
func NewHandlers(hub *Hub) *Handlers {
	return &Handlers{hub: hub}
}

// Handle routes a message to the appropriate handler.
func (h *Handlers) Handle(client *Client, msg *protocol.Message) {
	var err error

	switch msg.Type {
	case protocol.TypeAuthenticate:
		err = h.handleAuthenticate(client, msg)
	case protocol.TypeCreateGame:
		err = h.handleCreateGame(client, msg)
	case protocol.TypeJoinGame:
		err = h.handleJoinGame(client, msg)
	case protocol.TypeJoinByCode:
		err = h.handleJoinByCode(client, msg)
	case protocol.TypeLeaveGame:
		err = h.handleLeaveGame(client, msg)
	case protocol.TypeDeleteGame:
		err = h.handleDeleteGame(client, msg)
	case protocol.TypeAddAI:
		err = h.handleAddAI(client, msg)
	case protocol.TypeUpdateSettings:
		err = h.handleUpdateSettings(client, msg)
	case protocol.TypePlayerReady:
		err = h.handlePlayerReady(client, msg)
	case protocol.TypeStartGame:
		err = h.handleStartGame(client, msg)
	case protocol.TypeSelectTerritory:
		err = h.handleSelectTerritory(client, msg)
	case protocol.TypePlaceStockpile:
		err = h.handlePlaceStockpile(client, msg)
	case protocol.TypeMoveStockpile:
		err = h.handleMoveStockpile(client, msg)
	case protocol.TypeMoveUnit:
		err = h.handleMoveUnit(client, msg)
	case protocol.TypeEndPhase:
		err = h.handleEndPhase(client, msg)
	case protocol.TypePlanAttack:
		err = h.handlePlanAttack(client, msg)
	case protocol.TypeExecuteAttack:
		err = h.handleExecuteAttack(client, msg)
	case protocol.TypeBuild:
		err = h.handleBuild(client, msg)
	case protocol.TypeSetAlliance:
		err = h.handleSetAlliance(client, msg)
	case protocol.TypeAllianceVote:
		err = h.handleAllianceVote(client, msg)
	case protocol.TypeProposeTrade:
		err = h.handleProposeTrade(client, msg)
	case protocol.TypeRespondTrade:
		err = h.handleRespondTrade(client, msg)
	case protocol.TypeListGames:
		err = h.handleListGames(client, msg)
	case protocol.TypeYourGames:
		err = h.handleYourGames(client, msg)
	default:
		err = errors.New("unknown message type")
	}

	if err != nil {
		h.sendError(client, msg.ID, err)
	}
}

// handleAuthenticate handles player authentication/registration.
func (h *Handlers) handleAuthenticate(client *Client, msg *protocol.Message) error {
	var payload protocol.AuthenticatePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	db := h.hub.server.db
	var player *database.Player
	var err error

	// Try to find existing player by token
	if payload.Token != "" {
		player, err = db.GetPlayerByToken(payload.Token)
		if err != nil && !errors.Is(err, database.ErrPlayerNotFound) {
			return err
		}
	}

	// Create new player if not found
	if player == nil {
		name := payload.Name
		if name == "" {
			name = "Player"
		}
		player, err = db.CreatePlayer(name)
		if err != nil {
			return err
		}
		log.Printf("Created new player: %s (%s)", player.Name, player.ID)
	} else {
		// Update name if provided
		if payload.Name != "" && payload.Name != player.Name {
			db.UpdatePlayerName(player.ID, payload.Name)
			player.Name = payload.Name
		}
		db.UpdatePlayerLastSeen(player.ID)
		log.Printf("Player reconnected: %s (%s)", player.Name, player.ID)
	}

	// Associate client with player
	h.hub.SetClientPlayer(client, player.ID)
	client.Name = player.Name

	// Send auth result with token
	response := protocol.AuthResultPayload{
		Success:  true,
		PlayerID: player.ID,
		Token:    player.Token,
		Name:     player.Name,
	}

	respMsg, _ := protocol.NewMessage(protocol.TypeAuthResult, response)
	respMsg.ID = msg.ID
	client.Send(respMsg)

	// Send list of player's active games
	games, _ := db.GetPlayerGames(player.ID)
	if len(games) > 0 {
		gameList := make([]protocol.GameListItem, len(games))
		for i, g := range games {
			gameList[i] = protocol.GameListItem{
				ID:          g.ID,
				Name:        g.Name,
				JoinCode:    g.JoinCode,
				Status:      string(g.Status),
				PlayerCount: g.PlayerCount,
				MaxPlayers:  g.MaxPlayers,
				IsYourTurn:  false, // TODO: Check if it's this player's turn
			}
		}
		listMsg, _ := protocol.NewMessage(protocol.TypeYourGames, protocol.YourGamesPayload{Games: gameList})
		client.Send(listMsg)
	}

	return nil
}

// handleCreateGame handles game creation.
func (h *Handlers) handleCreateGame(client *Client, msg *protocol.Message) error {
	if client.PlayerID == "" {
		return errors.New("not authenticated")
	}

	var payload protocol.CreateGamePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Serialize map data for persistence
	var mapJSON string
	if payload.MapData != nil {
		rawMap := &maps.RawMap{
			ID:          payload.MapData.ID,
			Name:        payload.MapData.Name,
			Width:       payload.MapData.Width,
			Height:      payload.MapData.Height,
			Grid:        payload.MapData.Grid,
			Territories: make(map[string]maps.RawTerritory),
		}
		for id, t := range payload.MapData.Territories {
			rawMap.Territories[id] = maps.RawTerritory{
				Name:     t.Name,
				Resource: t.Resource,
			}
		}

		// Serialize to JSON for database storage
		mapBytes, err := json.Marshal(rawMap)
		if err != nil {
			return fmt.Errorf("failed to serialize map: %w", err)
		}
		mapJSON = string(mapBytes)

		// Process and register the map in memory
		processedMap := maps.Process(rawMap)
		maps.Register(processedMap)
		log.Printf("Registered generated map: %s (%dx%d, %d territories)",
			rawMap.ID, rawMap.Width, rawMap.Height, len(rawMap.Territories))
	}

	// Set defaults
	settings := database.GameSettings{
		MaxPlayers:    payload.Settings.MaxPlayers,
		GameLevel:     payload.Settings.GameLevel,
		ChanceLevel:   payload.Settings.ChanceLevel,
		VictoryCities: payload.Settings.VictoryCities,
		MapID:         payload.Settings.MapID,
	}
	if settings.MaxPlayers == 0 {
		settings.MaxPlayers = 4
	}
	if settings.VictoryCities == 0 {
		settings.VictoryCities = 3
	}
	if settings.GameLevel == "" {
		settings.GameLevel = "expert"
	}
	if settings.ChanceLevel == "" {
		settings.ChanceLevel = "medium"
	}

	game, err := h.hub.server.db.CreateGame(payload.Name, client.PlayerID, settings, payload.IsPublic, mapJSON)
	if err != nil {
		return err
	}

	// Add creator to the game
	if err := h.hub.server.db.JoinGame(game.ID, client.PlayerID, "orange"); err != nil {
		return err
	}

	// Add client to game channel
	h.hub.AddClientToGame(client, game.ID)
	h.hub.server.db.SetPlayerConnected(game.ID, client.PlayerID, true)

	log.Printf("Game created: %s (%s) by %s", game.Name, game.ID, client.Name)

	// Send response
	response := protocol.GameCreatedPayload{
		GameID:   game.ID,
		JoinCode: game.JoinCode,
	}
	respMsg, _ := protocol.NewMessage(protocol.TypeGameCreated, response)
	respMsg.ID = msg.ID
	client.Send(respMsg)

	// Send lobby state
	h.sendLobbyState(client, game.ID)

	return nil
}

// handleJoinGame handles joining a game by ID.
func (h *Handlers) handleJoinGame(client *Client, msg *protocol.Message) error {
	if client.PlayerID == "" {
		return errors.New("not authenticated")
	}

	var payload protocol.JoinGamePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	return h.joinGame(client, msg.ID, payload.GameID, payload.PreferredColor)
}

// handleJoinByCode handles joining a game by join code.
func (h *Handlers) handleJoinByCode(client *Client, msg *protocol.Message) error {
	if client.PlayerID == "" {
		return errors.New("not authenticated")
	}

	var payload protocol.JoinByCodePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	game, err := h.hub.server.db.GetGameByJoinCode(payload.JoinCode)
	if err != nil {
		return err
	}

	return h.joinGame(client, msg.ID, game.ID, payload.PreferredColor)
}

// joinGame is the common logic for joining a game.
func (h *Handlers) joinGame(client *Client, msgID string, gameID string, preferredColor string) error {
	db := h.hub.server.db

	// Check if already in game
	players, err := db.GetGamePlayers(gameID)
	if err != nil {
		return err
	}

	alreadyIn := false
	for _, p := range players {
		if p.PlayerID == client.PlayerID {
			alreadyIn = true
			break
		}
	}

	if !alreadyIn {
		// Pick color
		color := h.pickColor(players, preferredColor)
		if err := db.JoinGame(gameID, client.PlayerID, color); err != nil {
			return err
		}
		log.Printf("Player %s joined game %s", client.Name, gameID)
	}

	// Add to game channel
	h.hub.AddClientToGame(client, gameID)
	db.SetPlayerConnected(gameID, client.PlayerID, true)

	// Send success response
	game, _ := db.GetGame(gameID)
	response := protocol.JoinedGamePayload{
		GameID:   gameID,
		JoinCode: game.JoinCode,
	}
	respMsg, _ := protocol.NewMessage(protocol.TypeJoinedGame, response)
	respMsg.ID = msgID
	client.Send(respMsg)

	// Check if game has started - send appropriate state
	if game.Status == database.GameStatusStarted {
		// Game is in progress - tell client to switch to game scene, then send game state
		log.Printf("Player %s reconnecting to started game %s", client.Name, gameID)

		// Send game_started so client switches to gameplay scene
		startedMsg, _ := protocol.NewMessage(protocol.TypeGameStarted, protocol.GameStartedPayload{
			GameID: gameID,
		})
		client.Send(startedMsg)

		// Then send the current game state
		h.broadcastGameState(gameID)

		// Send game history
		h.sendGameHistory(client, gameID)
	} else {
		// Game is in lobby - send lobby state
		h.sendLobbyState(client, gameID)
		h.hub.notifyGamePlayers(gameID, protocol.TypePlayerJoined, protocol.PlayerJoinedPayload{
			PlayerID: client.PlayerID,
			Name:     client.Name,
		})
	}

	return nil
}

// handleLeaveGame handles leaving a game.
func (h *Handlers) handleLeaveGame(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	gameID := client.GameID
	db := h.hub.server.db

	// Check if game has started
	game, err := db.GetGame(gameID)
	if err != nil {
		return err
	}

	if game.Status == database.GameStatusStarted {
		// Can't leave a started game, just disconnect
		h.hub.RemoveClientFromGame(client, gameID)
		db.SetPlayerConnected(gameID, client.PlayerID, false)
	} else {
		// Remove from game
		db.LeaveGame(gameID, client.PlayerID)
		h.hub.RemoveClientFromGame(client, gameID)

		// Notify others
		h.hub.notifyGamePlayers(gameID, protocol.TypePlayerLeft, protocol.PlayerLeftPayload{
			PlayerID: client.PlayerID,
		})
	}

	log.Printf("Player %s left game %s", client.Name, gameID)
	return nil
}

// handleAddAI handles adding an AI player.
func (h *Handlers) handleAddAI(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	// Verify client is host
	game, err := h.hub.server.db.GetGame(client.GameID)
	if err != nil {
		return err
	}
	if game.HostPlayerID != client.PlayerID {
		return errors.New("only host can add AI")
	}

	var payload protocol.AddAIPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Get existing players to pick color
	players, _ := h.hub.server.db.GetGamePlayers(client.GameID)
	color := h.pickColor(players, "")

	if err := h.hub.server.db.AddAIPlayer(client.GameID, color, payload.Personality); err != nil {
		return err
	}

	// Broadcast updated lobby state
	h.broadcastLobbyState(client.GameID)

	return nil
}

// handleUpdateSettings handles updating game settings (host only).
func (h *Handlers) handleUpdateSettings(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	// Verify client is host
	game, err := h.hub.server.db.GetGame(client.GameID)
	if err != nil {
		return err
	}
	if game.HostPlayerID != client.PlayerID {
		return errors.New("only host can update settings")
	}

	var payload protocol.UpdateSettingPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Update the specific setting
	switch payload.Key {
	case "chanceLevel":
		if err := h.hub.server.db.UpdateGameSetting(client.GameID, "chance_level", payload.Value); err != nil {
			return err
		}
	case "victoryCities":
		if err := h.hub.server.db.UpdateGameSetting(client.GameID, "victory_cities", payload.Value); err != nil {
			return err
		}
	case "maxPlayers":
		if err := h.hub.server.db.UpdateGameSetting(client.GameID, "max_players", payload.Value); err != nil {
			return err
		}
	default:
		return errors.New("unknown setting: " + payload.Key)
	}

	log.Printf("Host %s updated game setting: %s = %s", client.Name, payload.Key, payload.Value)

	// Broadcast updated lobby state
	h.broadcastLobbyState(client.GameID)

	return nil
}

// handlePlayerReady handles ready state toggle.
func (h *Handlers) handlePlayerReady(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.PlayerReadyPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	if err := h.hub.server.db.SetPlayerReady(client.GameID, client.PlayerID, payload.Ready); err != nil {
		return err
	}

	// Broadcast updated lobby state
	h.broadcastLobbyState(client.GameID)

	return nil
}

// handleStartGame handles starting the game.
func (h *Handlers) handleStartGame(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	db := h.hub.server.db

	// Verify client is host
	game, err := db.GetGame(client.GameID)
	if err != nil {
		return err
	}
	if game.HostPlayerID != client.PlayerID {
		return errors.New("only host can start game")
	}

	// Check all players are ready
	players, err := db.GetGamePlayers(client.GameID)
	if err != nil {
		return err
	}

	if len(players) < 2 {
		return errors.New("need at least 2 players")
	}

	for _, p := range players {
		if !p.IsReady && !p.IsAI {
			return errors.New("not all players are ready")
		}
	}

	// Start the game
	if err := db.StartGame(client.GameID); err != nil {
		return err
	}

	log.Printf("Game started: %s", client.GameID)

	// Initialize game state
	if err := h.initializeGameState(client.GameID, game, players); err != nil {
		log.Printf("Failed to initialize game state: %v", err)
		return err
	}

	// Log game start in history
	h.logHistory(client.GameID, 1, "Selection", "", "", database.EventRoundStart, "Game started - Round 1")

	// Notify all players
	h.hub.notifyGamePlayers(client.GameID, protocol.TypeGameStarted, protocol.GameStartedPayload{
		GameID: client.GameID,
	})

	log.Printf("Broadcasting initial game state for game %s", client.GameID)

	// Send initial game state
	h.broadcastGameState(client.GameID)

	// Send initial game history
	h.broadcastGameHistory(client.GameID)

	// Trigger AI if first player is AI
	go h.checkAndTriggerAI(client.GameID)

	return nil
}

// handleListGames handles listing public games.
func (h *Handlers) handleListGames(client *Client, msg *protocol.Message) error {
	// Clean up abandoned lobbies before listing
	if err := h.hub.server.db.CleanupAbandonedLobbies(); err != nil {
		log.Printf("Warning: Failed to cleanup abandoned lobbies: %v", err)
	}

	games, err := h.hub.server.db.ListPublicGames()
	if err != nil {
		return err
	}

	gameList := make([]protocol.GameListItem, len(games))
	for i, g := range games {
		gameList[i] = protocol.GameListItem{
			ID:          g.ID,
			Name:        g.Name,
			JoinCode:    g.JoinCode,
			Status:      string(g.Status),
			PlayerCount: g.PlayerCount,
			MaxPlayers:  g.MaxPlayers,
		}
	}

	response := protocol.GameListPayload{Games: gameList}
	respMsg, _ := protocol.NewMessage(protocol.TypeGameList, response)
	respMsg.ID = msg.ID
	client.Send(respMsg)

	return nil
}

// sendLobbyState sends the current lobby state to a client.
func (h *Handlers) sendLobbyState(client *Client, gameID string) {
	db := h.hub.server.db

	game, err := db.GetGame(gameID)
	if err != nil {
		return
	}

	players, err := db.GetGamePlayers(gameID)
	if err != nil {
		return
	}

	lobbyPlayers := make([]protocol.LobbyPlayer, len(players))
	for i, p := range players {
		lobbyPlayers[i] = protocol.LobbyPlayer{
			ID:            p.PlayerID,
			Name:          p.PlayerName,
			Color:         p.Color,
			IsAI:          p.IsAI,
			AIPersonality: p.AIPersonality,
			Ready:         p.IsReady,
			IsConnected:   p.IsConnected,
		}
	}

	payload := protocol.LobbyStatePayload{
		GameID:   game.ID,
		GameName: game.Name,
		JoinCode: game.JoinCode,
		HostID:   game.HostPlayerID,
		IsPublic: game.IsPublic,
		Settings: protocol.GameSettings{
			MaxPlayers:    game.Settings.MaxPlayers,
			GameLevel:     game.Settings.GameLevel,
			ChanceLevel:   game.Settings.ChanceLevel,
			VictoryCities: game.Settings.VictoryCities,
			MapID:         game.Settings.MapID,
		},
		Players: lobbyPlayers,
	}

	msg, _ := protocol.NewMessage(protocol.TypeLobbyState, payload)
	client.Send(msg)
}

// broadcastLobbyState sends lobby state to all clients in a game.
func (h *Handlers) broadcastLobbyState(gameID string) {
	h.hub.mu.RLock()
	clients := h.hub.gameClients[gameID]
	h.hub.mu.RUnlock()

	for client := range clients {
		h.sendLobbyState(client, gameID)
	}
}

// sendError sends an error response.
func (h *Handlers) sendError(client *Client, msgID string, err error) {
	payload := protocol.ErrorPayload{
		Code:    protocol.ErrCodeInternalError,
		Message: err.Error(),
	}
	msg, _ := protocol.NewMessage(protocol.TypeError, payload)
	msg.ID = msgID
	client.Send(msg)
}

// pickColor picks an available color for a player.
func (h *Handlers) pickColor(players []*database.GamePlayer, preferred string) string {
	colors := []string{"orange", "cyan", "green", "yellow", "purple", "red", "blue"}

	used := make(map[string]bool)
	for _, p := range players {
		used[p.Color] = true
	}

	if preferred != "" && !used[preferred] {
		return preferred
	}

	for _, c := range colors {
		if !used[c] {
			return c
		}
	}
	return "orange" // Fallback
}

// initializeGameState creates the game state from the map and players.
func (h *Handlers) initializeGameState(gameID string, dbGame *database.Game, dbPlayers []*database.GamePlayer) error {
	// Get the map
	mapData := maps.Get(dbGame.Settings.MapID)
	if mapData == nil {
		return errors.New("map not found: " + dbGame.Settings.MapID)
	}

	// Convert database players to game players
	gamePlayers := make([]*game.Player, 0, len(dbPlayers))
	for _, dbp := range dbPlayers {
		player := game.NewPlayer(dbp.PlayerID, dbp.PlayerName, game.PlayerColor(dbp.Color))
		if dbp.IsAI {
			player.IsAI = true
			player.AIPersonality = parseAIPersonality(dbp.AIPersonality)
			player.Alliance = game.AllianceNeutral // AI is always neutral
		} else {
			// Apply alliance setting from database
			if dbp.AllianceSetting != "" {
				player.Alliance = game.AllianceSetting(dbp.AllianceSetting)
			}
		}
		gamePlayers = append(gamePlayers, player)
	}

	// Convert map data to game map data
	gameMapData := convertMapToGameData(mapData)

	// Convert database settings to game settings
	gameSettings := game.Settings{
		GameLevel:     parseGameLevel(dbGame.Settings.GameLevel),
		ChanceLevel:   parseChanceLevel(dbGame.Settings.ChanceLevel),
		VictoryCities: dbGame.Settings.VictoryCities,
		MapID:         dbGame.Settings.MapID,
		MaxPlayers:    dbGame.Settings.MaxPlayers,
	}

	// Initialize game state
	state, err := game.InitializeGame(gameMapData, gamePlayers, gameSettings)
	if err != nil {
		return err
	}

	// Set the game ID to match the database game
	state.ID = gameID

	// Serialize state to JSON
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}

	// Save to database
	return h.hub.server.db.SaveGameState(gameID, string(stateJSON), state.CurrentPlayerID, state.Round, state.Phase.String())
}

// convertMapToGameData converts a map to game initialization data.
func convertMapToGameData(m *maps.Map) game.MapData {
	territories := make(map[string]game.TerritoryData)
	for id, t := range m.Territories {
		adjacent := make([]string, len(t.AdjacentTerritories))
		for i, adj := range t.AdjacentTerritories {
			adjacent[i] = maps.TerritoryIDToString(adj)
		}

		waters := make([]string, len(t.AdjacentWaters))
		for i, w := range t.AdjacentWaters {
			waters[i] = maps.WaterIDToString(w)
		}

		territories[maps.TerritoryIDToString(id)] = game.TerritoryData{
			Name:         t.Name,
			Resource:     t.Resource,
			Adjacent:     adjacent,
			CoastalTiles: t.CoastalCells,
			WaterBodies:  waters,
		}
	}

	waterBodies := make(map[string]game.WaterBodyData)
	for id, wb := range m.WaterBodies {
		coastal := make([]string, len(wb.CoastalTerritories))
		for i, t := range wb.CoastalTerritories {
			coastal[i] = maps.TerritoryIDToString(t)
		}

		waterBodies[maps.WaterIDToString(id)] = game.WaterBodyData{
			Territories: coastal,
		}
	}

	return game.MapData{
		ID:          m.ID,
		Name:        m.Name,
		Territories: territories,
		WaterBodies: waterBodies,
	}
}

// broadcastGameState sends the current game state to all players.
func (h *Handlers) broadcastGameState(gameID string) {
	// Load game state from database
	stateJSON, err := h.hub.server.db.GetGameState(gameID)
	if err != nil {
		log.Printf("Failed to load game state: %v", err)
		return
	}

	if stateJSON == "" {
		log.Printf("No game state found for game %s", gameID)
		return
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		log.Printf("Failed to unmarshal game state: %v", err)
		return
	}

	// Check for skipped phases and notify clients
	if len(state.SkippedPhases) > 0 {
		for _, skip := range state.SkippedPhases {
			skipPayload := protocol.PhaseSkippedPayload{
				Phase:  skip.Phase.String(),
				Reason: skip.Reason,
			}
			log.Printf("Broadcasting phase skip: %s - %s", skip.Phase.String(), skip.Reason)
			h.hub.notifyGamePlayers(gameID, protocol.TypePhaseSkipped, skipPayload)
		}

		// Clear skipped phases after notifying
		state.SkippedPhases = nil

		// Save the updated state (with cleared skips)
		updatedJSON, _ := json.Marshal(state)
		h.hub.server.db.SaveGameState(gameID, string(updatedJSON),
			state.CurrentPlayerID, state.Round, state.Phase.String())
	}

	// Update player online status from hub
	for playerID, player := range state.Players {
		if player.IsAI {
			player.IsOnline = true // AI is always "online"
		} else {
			player.IsOnline = h.hub.IsPlayerOnline(playerID)
		}
	}

	// Get the map for rendering info
	mapData := maps.Get(state.Settings.MapID)
	if mapData == nil {
		// Map not in registry - try to load from database
		mapData = h.loadMapFromDatabase(gameID, state.Settings.MapID)
		if mapData == nil {
			log.Printf("Map not found in registry or database: %s", state.Settings.MapID)
			return
		}
	}

	// Create payload with state and map rendering data
	payload := protocol.GameStatePayload{
		State: createStatePayload(&state, mapData),
	}

	log.Printf("Broadcasting game state for game %s", gameID)
	h.hub.notifyGamePlayers(gameID, protocol.TypeGameState, payload)
	log.Printf("Game state broadcast complete")

	// Also broadcast history to keep it in sync
	h.broadcastGameHistory(gameID)
}

// loadMapFromDatabase loads a map from the database and registers it.
func (h *Handlers) loadMapFromDatabase(gameID, mapID string) *maps.Map {
	mapJSON, err := h.hub.server.db.GetGameMapJSON(gameID)
	if err != nil {
		log.Printf("Failed to load map JSON from database: %v", err)
		return nil
	}
	if mapJSON == "" {
		log.Printf("No map JSON stored for game %s", gameID)
		return nil
	}

	// Parse and process the map
	mapData, err := maps.LoadFromJSON([]byte(mapJSON))
	if err != nil {
		log.Printf("Failed to parse stored map JSON: %v", err)
		return nil
	}

	// Register it for future lookups
	maps.Register(mapData)
	log.Printf("Loaded and registered map %s from database", mapID)

	return mapData
}

// createStatePayload creates a simplified state payload for clients.
func createStatePayload(state *game.GameState, mapData *maps.Map) map[string]interface{} {
	// Convert territories
	territories := make(map[string]interface{})
	for id, t := range state.Territories {
		territories[id] = map[string]interface{}{
			"id":           id,
			"name":         t.Name,
			"owner":        t.Owner,
			"resource":     t.Resource.String(),
			"hasCity":      t.HasCity,
			"hasWeapon":    t.HasWeapon,
			"hasHorse":     t.HasHorse,
			"boats":        t.Boats,        // Map of water body ID -> count
			"totalBoats":   t.TotalBoats(), // Total for convenience
			"coastalTiles": t.CoastalTiles,
			"waterBodies":  t.WaterBodies,
			"adjacent":     t.Adjacent, // Adjacent territory IDs for city influence check
		}
	}

	// Convert players
	players := make(map[string]interface{})
	for id, p := range state.Players {
		playerData := map[string]interface{}{
			"id":       id,
			"name":     p.Name,
			"color":    string(p.Color),
			"isAI":     p.IsAI,
			"isOnline": p.IsOnline,
			"alliance": string(p.Alliance),
		}

		// Include stockpile information if it exists
		if p.StockpileTerritory != "" {
			playerData["stockpileTerritory"] = p.StockpileTerritory
			playerData["stockpile"] = map[string]interface{}{
				"coal":   p.Stockpile.Coal,
				"gold":   p.Stockpile.Gold,
				"iron":   p.Stockpile.Iron,
				"timber": p.Stockpile.Timber,
			}
		}

		players[id] = playerData
	}

	// Convert water bodies with cell locations for rendering
	waterBodies := make(map[string]interface{})
	for id, wb := range mapData.WaterBodies {
		waterBodies[maps.WaterIDToString(id)] = map[string]interface{}{
			"id":          maps.WaterIDToString(id),
			"cells":       wb.Cells,
			"territories": wb.CoastalTerritories,
		}
	}

	// Add map rendering data
	mapInfo := map[string]interface{}{
		"id":          mapData.ID,
		"name":        mapData.Name,
		"width":       mapData.Width,
		"height":      mapData.Height,
		"grid":        mapData.Grid,
		"waterGrid":   mapData.WaterGrid,
		"waterBodies": waterBodies,
	}

	return map[string]interface{}{
		"gameId":          state.ID,
		"round":           state.Round,
		"phase":           state.Phase.String(),
		"currentPlayerId": state.CurrentPlayerID,
		"playerOrder":     state.PlayerOrder,
		"territories":     territories,
		"players":         players,
		"map":             mapInfo,
	}
}

// handleSelectTerritory handles territory selection during the initial phase.
func (h *Handlers) handleSelectTerritory(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.SelectTerritoryPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Get territory name before selection
	terrName := payload.TerritoryID
	if terr, ok := state.Territories[payload.TerritoryID]; ok {
		terrName = terr.Name
	}

	// Capture round and phase BEFORE selection (it may advance after the last territory)
	historyRound := state.Round
	historyPhase := state.Phase.String()

	// Execute selection
	if err := state.SelectTerritory(client.PlayerID, payload.TerritoryID); err != nil {
		return err
	}

	// Log history event with the round/phase from BEFORE the selection
	h.logHistory(client.GameID, historyRound, historyPhase, client.PlayerID, client.Name,
		database.EventTerritorySelected, fmt.Sprintf("Selected %s", terrName))

	// Save updated state
	stateJSON2, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := h.hub.server.db.SaveGameState(client.GameID, string(stateJSON2),
		state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
		return err
	}

	log.Printf("Player %s selected territory %s", client.Name, payload.TerritoryID)

	// Broadcast updated state
	h.broadcastGameState(client.GameID)

	// Check if next player is AI and trigger their move
	go h.checkAndTriggerAI(client.GameID)

	return nil
}

// handlePlaceStockpile handles placing the stockpile after territory selection.
func (h *Handlers) handlePlaceStockpile(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.PlaceStockpilePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Get territory name before placement
	terrName := payload.TerritoryID
	if terr, ok := state.Territories[payload.TerritoryID]; ok {
		terrName = terr.Name
	}

	// Place stockpile
	if err := state.PlaceStockpile(client.PlayerID, payload.TerritoryID); err != nil {
		return err
	}

	// Log history event
	h.logHistory(client.GameID, state.Round, state.Phase.String(), client.PlayerID, client.Name,
		database.EventStockpilePlaced, fmt.Sprintf("Placed stockpile on %s", terrName))

	// Save updated state
	stateJSON2, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := h.hub.server.db.SaveGameState(client.GameID, string(stateJSON2),
		state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
		return err
	}

	log.Printf("Player %s placed stockpile at %s", client.Name, payload.TerritoryID)

	// Broadcast updated state
	h.broadcastGameState(client.GameID)

	// Trigger AI stockpile placement if still in production phase
	if state.Phase == game.PhaseProduction && state.Round == 1 {
		go h.checkAndTriggerAI(client.GameID)
	} else {
		// Phase changed, check for AI turn in new phase
		go h.checkAndTriggerAI(client.GameID)
	}

	return nil
}

// Helper functions for parsing settings
func parseGameLevel(s string) game.GameLevel {
	switch s {
	case "beginner":
		return game.LevelBeginner
	case "intermediate":
		return game.LevelIntermediate
	case "advanced":
		return game.LevelAdvanced
	case "expert":
		return game.LevelExpert
	default:
		return game.LevelExpert
	}
}

func parseChanceLevel(s string) game.ChanceLevel {
	switch s {
	case "low":
		return game.ChanceLow
	case "medium":
		return game.ChanceMedium
	case "high":
		return game.ChanceHigh
	default:
		return game.ChanceMedium
	}
}

func parseAIPersonality(s string) game.AIPersonality {
	switch s {
	case "aggressive":
		return game.AIAggressive
	case "defensive":
		return game.AIDefensive
	case "passive":
		return game.AIPassive
	default:
		return game.AIAggressive
	}
}

// handleYourGames returns games the player is participating in.
func (h *Handlers) handleYourGames(client *Client, msg *protocol.Message) error {
	if client.PlayerID == "" {
		return errors.New("not authenticated")
	}

	games, err := h.hub.server.db.GetPlayerGames(client.PlayerID)
	if err != nil {
		return err
	}

	gameList := make([]protocol.GameListItem, 0, len(games))
	for _, g := range games {
		// Check if it's this player's turn and if game is actually over
		isYourTurn := false
		isGameOver := false
		if g.Status == database.GameStatusStarted {
			stateJSON, err := h.hub.server.db.GetGameState(g.ID)
			if err == nil && stateJSON != "" {
				var state game.GameState
				if err := json.Unmarshal([]byte(stateJSON), &state); err == nil {
					isYourTurn = state.CurrentPlayerID == client.PlayerID
					// Check if game is actually over but wasn't marked in DB
					if state.IsGameOver() {
						isGameOver = true
						// Fix the database status
						winner := state.GetWinner()
						if winner != nil {
							reason := "elimination"
							if state.CountCities(winner.ID) >= state.Settings.VictoryCities {
								reason = "cities"
							}
							h.hub.server.db.EndGame(g.ID, winner.ID, reason)
						}
					}
				}
			}
		}

		// Skip games that are actually over
		if isGameOver {
			continue
		}

		// Get host player name
		hostName := ""
		if host, err := h.hub.server.db.GetPlayerByID(g.HostPlayerID); err == nil {
			hostName = host.Name
		}

		gameList = append(gameList, protocol.GameListItem{
			ID:          g.ID,
			Name:        g.Name,
			JoinCode:    g.JoinCode,
			Status:      string(g.Status),
			PlayerCount: g.PlayerCount,
			MaxPlayers:  g.MaxPlayers,
			IsYourTurn:  isYourTurn,
			HostName:    hostName,
		})
	}

	response := protocol.YourGamesPayload{Games: gameList}
	respMsg, _ := protocol.NewMessage(protocol.TypeYourGames, response)
	respMsg.ID = msg.ID
	client.Send(respMsg)

	return nil
}

// handleDeleteGame allows the creator to delete a game.
func (h *Handlers) handleDeleteGame(client *Client, msg *protocol.Message) error {
	if client.PlayerID == "" {
		return errors.New("not authenticated")
	}

	var payload protocol.DeleteGamePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Get the game to check ownership
	game, err := h.hub.server.db.GetGame(payload.GameID)
	if err != nil {
		return err
	}

	// Only the creator can delete the game
	if game.HostPlayerID != client.PlayerID {
		return errors.New("only the game creator can delete the game")
	}

	// Delete the game
	if err := h.hub.server.db.DeleteGame(payload.GameID); err != nil {
		return err
	}

	log.Printf("Game %s deleted by creator %s", payload.GameID, client.PlayerID)

	// Notify all players in the game that it was deleted
	h.hub.notifyGamePlayers(payload.GameID, protocol.TypeGameDeleted, protocol.GameDeletedPayload{
		GameID: payload.GameID,
		Reason: "Game deleted by creator",
	})

	// Remove all players from the game session
	h.hub.mu.Lock()
	if session, ok := h.hub.gameClients[payload.GameID]; ok {
		for c := range session {
			c.GameID = ""
		}
		delete(h.hub.gameClients, payload.GameID)
	}
	h.hub.mu.Unlock()

	// Send confirmation to requester
	respMsg, _ := protocol.NewMessage(protocol.TypeGameDeleted, protocol.GameDeletedPayload{
		GameID: payload.GameID,
	})
	respMsg.ID = msg.ID
	client.Send(respMsg)

	// Send updated game list to the client
	h.sendUpdatedGameList(client)

	return nil
}

// sendUpdatedGameList sends the player's updated game list.
func (h *Handlers) sendUpdatedGameList(client *Client) {
	if client.PlayerID == "" {
		return
	}

	games, err := h.hub.server.db.GetPlayerGames(client.PlayerID)
	if err != nil {
		log.Printf("Failed to get player games: %v", err)
		return
	}

	gameList := make([]protocol.GameListItem, len(games))
	for i, g := range games {
		isYourTurn := false
		if g.Status == database.GameStatusStarted {
			stateJSON, err := h.hub.server.db.GetGameState(g.ID)
			if err == nil && stateJSON != "" {
				var state game.GameState
				if err := json.Unmarshal([]byte(stateJSON), &state); err == nil {
					isYourTurn = state.CurrentPlayerID == client.PlayerID
				}
			}
		}

		gameList[i] = protocol.GameListItem{
			ID:          g.ID,
			Name:        g.Name,
			JoinCode:    g.JoinCode,
			Status:      string(g.Status),
			PlayerCount: g.PlayerCount,
			MaxPlayers:  g.MaxPlayers,
			IsYourTurn:  isYourTurn,
		}
	}

	listMsg, _ := protocol.NewMessage(protocol.TypeYourGames, protocol.YourGamesPayload{Games: gameList})
	client.Send(listMsg)
}

// ==================== AI Logic ====================

// checkAndTriggerAI checks if the current player is an AI and triggers their move.
func (h *Handlers) checkAndTriggerAI(gameID string) {
	// Small delay to ensure state is fully saved and broadcast
	time.Sleep(100 * time.Millisecond)

	// Load current game state
	stateJSON, err := h.hub.server.db.GetGameState(gameID)
	if err != nil {
		log.Printf("AI: Failed to load game state: %v", err)
		return
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		log.Printf("AI: Failed to unmarshal game state: %v", err)
		return
	}

	// Check if game is over
	if state.IsGameOver() {
		log.Printf("AI: Game is over")
		return
	}

	// Get player info from database
	players, err := h.hub.server.db.GetGamePlayers(gameID)
	if err != nil {
		return
	}

	// Special case: Production phase round 1 is stockpile placement (all players at once)
	// This needs to happen regardless of whose turn it is
	if state.Phase == game.PhaseProduction && state.Round == 1 {
		log.Printf("AI: Production phase round 1 - checking for stockpile placement")
		h.aiPlaceStockpile(gameID, &state)
		return
	}

	// Check if current player is AI
	currentPlayer := state.Players[state.CurrentPlayerID]
	if currentPlayer == nil {
		return
	}

	var isAI bool
	for _, p := range players {
		if p.PlayerID == state.CurrentPlayerID && p.IsAI {
			isAI = true
			break
		}
	}

	if !isAI {
		return
	}

	log.Printf("AI: It's AI player %s's turn in phase %s", state.CurrentPlayerID, state.Phase)

	// Trigger AI action based on phase
	switch state.Phase {
	case game.PhaseTerritorySelection:
		h.aiSelectTerritory(gameID, &state)
	case game.PhaseProduction:
		// After round 1, production is automatic - just advance
		log.Printf("AI: Production is automatic, skipping")
	case game.PhaseTrade:
		// AI doesn't trade for now, just skip
		h.aiSkipTrade(gameID, &state)
	case game.PhaseShipment:
		h.aiShipment(gameID, &state)
	case game.PhaseConquest:
		h.aiConquest(gameID, &state)
	case game.PhaseDevelopment:
		h.aiDevelopment(gameID, &state)
	default:
		log.Printf("AI: No handler for phase: %s", state.Phase)
	}
}

// aiSelectTerritory makes the AI select a random available territory.
func (h *Handlers) aiSelectTerritory(gameID string, state *game.GameState) {
	// Find all unclaimed territories
	var availableTerritories []string
	for id, territory := range state.Territories {
		if territory.Owner == "" {
			availableTerritories = append(availableTerritories, id)
		}
	}

	if len(availableTerritories) == 0 {
		log.Printf("AI: No territories available to select")
		// Still trigger next AI check in case phase changed
		go h.checkAndTriggerAI(gameID)
		return
	}

	// Select random territory
	selectedID := availableTerritories[rand.Intn(len(availableTerritories))]

	// Get territory name and player name
	terrName := selectedID
	if terr, ok := state.Territories[selectedID]; ok {
		terrName = terr.Name
	}
	playerName := state.CurrentPlayerID
	if player, ok := state.Players[state.CurrentPlayerID]; ok {
		playerName = player.Name
	}

	log.Printf("AI: Selecting territory %s from %d available", selectedID, len(availableTerritories))

	// Execute selection
	if err := state.SelectTerritory(state.CurrentPlayerID, selectedID); err != nil {
		log.Printf("AI: Failed to select territory: %v", err)
		// Try again after a delay
		go h.checkAndTriggerAI(gameID)
		return
	}

	// Log history event
	h.logHistory(gameID, state.Round, state.Phase.String(), state.CurrentPlayerID, playerName,
		database.EventTerritorySelected, fmt.Sprintf("Selected %s", terrName))

	// Save and broadcast
	h.saveAndBroadcastAIState(gameID, state)

	log.Printf("AI: Successfully selected territory %s", selectedID)

	// Check if next player is also AI
	go h.checkAndTriggerAI(gameID)
}

// aiPlaceStockpile places all AI players' stockpiles on the first production phase.
func (h *Handlers) aiPlaceStockpile(gameID string, state *game.GameState) {
	// Get AI player info from database
	dbPlayers, err := h.hub.server.db.GetGamePlayers(gameID)
	if err != nil {
		log.Printf("AI: Failed to get players: %v", err)
		return
	}

	// Create a map of AI players
	aiPlayers := make(map[string]bool)
	for _, p := range dbPlayers {
		if p.IsAI {
			aiPlayers[p.PlayerID] = true
		}
	}

	// Place stockpiles for all AI players that haven't placed yet
	anyPlaced := false
	for playerID, player := range state.Players {
		if !aiPlayers[playerID] {
			continue // Not an AI
		}
		if player.Eliminated || player.StockpileTerritory != "" {
			continue // Already placed or eliminated
		}

		// Find player's territories
		territories := state.GetPlayerTerritories(playerID)
		if len(territories) == 0 {
			log.Printf("AI: Player %s has no territories", playerID)
			continue
		}

		// Pick a random territory
		selectedID := territories[rand.Intn(len(territories))]

		// Get territory name and player name
		terrName := selectedID
		if terr, ok := state.Territories[selectedID]; ok {
			terrName = terr.Name
		}

		log.Printf("AI: Player %s placing stockpile at %s", playerID, selectedID)

		// Place stockpile
		if err := state.PlaceStockpile(playerID, selectedID); err != nil {
			log.Printf("AI: Failed to place stockpile: %v", err)
			continue
		}

		// Log history event
		h.logHistory(gameID, state.Round, state.Phase.String(), playerID, player.Name,
			database.EventStockpilePlaced, fmt.Sprintf("Placed stockpile on %s", terrName))

		anyPlaced = true
	}

	if anyPlaced {
		// Save state
		h.saveAndBroadcastAIState(gameID, state)

		// If phase changed (all stockpiles placed), check for next AI turn
		if state.Phase != game.PhaseProduction || state.Round > 1 {
			go h.checkAndTriggerAI(gameID)
		}
	}
}

// aiSkipTrade skips the trade phase (AI doesn't trade for now).
func (h *Handlers) aiSkipTrade(gameID string, state *game.GameState) {
	log.Printf("AI: Skipping trade phase for player %s", state.CurrentPlayerID)

	if err := state.SkipTrade(state.CurrentPlayerID); err != nil {
		log.Printf("AI: Failed to skip trade: %v", err)
		// Still save and continue to prevent hang
	}

	// Save state
	h.saveAndBroadcastAIState(gameID, state)

	// Check if next player is also AI
	go h.checkAndTriggerAI(gameID)
}

// aiShipment handles the AI's shipment phase.
func (h *Handlers) aiShipment(gameID string, state *game.GameState) {
	player := state.Players[state.CurrentPlayerID]
	if player == nil {
		return
	}

	// Simple AI: 50% chance to skip, 50% chance to move stockpile somewhere random
	moved := false
	if rand.Float32() >= 0.5 {
		// Try to move stockpile
		destinations := state.GetValidStockpileDestinations(state.CurrentPlayerID)
		if len(destinations) > 0 {
			dest := destinations[rand.Intn(len(destinations))]
			log.Printf("AI: Moving stockpile to %s", dest)
			if err := state.MoveStockpile(state.CurrentPlayerID, dest); err != nil {
				log.Printf("AI: Failed to move stockpile: %v", err)
			} else {
				moved = true
			}
		}
	}

	// Skip if we didn't move
	if !moved {
		log.Printf("AI: Skipping shipment")
		if err := state.SkipShipment(state.CurrentPlayerID); err != nil {
			log.Printf("AI: Failed to skip shipment: %v", err)
		}
	}

	// Save state
	h.saveAndBroadcastAIState(gameID, state)

	// Check if next player is also AI
	go h.checkAndTriggerAI(gameID)
}

// aiConquest handles the AI's conquest phase.
func (h *Handlers) aiConquest(gameID string, state *game.GameState) {
	player := state.Players[state.CurrentPlayerID]
	if player == nil {
		return
	}

	// If no attacks remaining, end conquest phase
	if player.AttacksRemaining <= 0 {
		log.Printf("AI: No attacks remaining, ending conquest")
		state.EndConquest(state.CurrentPlayerID)
		h.saveAndBroadcastAIState(gameID, state)
		go h.checkAndTriggerAI(gameID)
		return
	}

	// Get attackable targets
	targets := state.GetAttackableTargets(state.CurrentPlayerID)

	if len(targets) == 0 {
		log.Printf("AI: No attackable targets, ending conquest")
		state.EndConquest(state.CurrentPlayerID)
		h.saveAndBroadcastAIState(gameID, state)
		go h.checkAndTriggerAI(gameID)
		return
	}

	// Simple AI: Try to attack if we have a strength advantage
	var bestTarget string
	var bestOdds float64 = 0

	for _, targetID := range targets {
		plan := state.GetAttackPlan(state.CurrentPlayerID, targetID)
		if plan == nil || !plan.CanAttack {
			continue
		}

		// Calculate odds
		if plan.DefenseStrength == 0 {
			bestTarget = targetID
			bestOdds = 999
			break
		}
		odds := float64(plan.AttackStrength) / float64(plan.DefenseStrength)
		if odds > bestOdds {
			bestOdds = odds
			bestTarget = targetID
		}
	}

	// Only attack if we have at least 1:1 odds (simple AI)
	if bestTarget != "" && bestOdds >= 1.0 {
		// Capture attacker ID BEFORE attack (Attack() may advance turn)
		attackerID := state.CurrentPlayerID

		// Get territory name and player name
		terrName := bestTarget
		if terr, ok := state.Territories[bestTarget]; ok {
			terrName = terr.Name
		}
		playerName := attackerID
		if p, ok := state.Players[attackerID]; ok {
			playerName = p.Name
		}

		log.Printf("AI: Attacking %s (odds: %.2f)", bestTarget, bestOdds)
		result, err := state.Attack(attackerID, bestTarget, nil)
		if err != nil {
			log.Printf("AI: Attack failed: %v", err)
			state.EndConquest(attackerID)
		} else {
			// Broadcast combat result to all players so everyone sees the animation
			unitsDestroyed := make([]string, 0)
			for _, u := range result.UnitsDestroyed {
				unitsDestroyed = append(unitsDestroyed, u.TerritoryID)
			}
			unitsCaptured := make([]string, 0)
			for _, u := range result.UnitsCaptured {
				unitsCaptured = append(unitsCaptured, u.TerritoryID)
			}

			combatResult := protocol.CombatResultPayload{
				Success:         true,
				AttackerID:      attackerID, // Use captured ID, not state.CurrentPlayerID
				AttackerWins:    result.AttackerWins,
				AttackStrength:  result.AttackStrength,
				DefenseStrength: result.DefenseStrength,
				TargetTerritory: bestTarget,
				UnitsDestroyed:  unitsDestroyed,
				UnitsCaptured:   unitsCaptured,
			}
			h.hub.notifyGamePlayers(gameID, protocol.TypeActionResult, combatResult)

			if result.AttackerWins {
				log.Printf("AI: Attack successful!")
				h.logHistory(gameID, state.Round, state.Phase.String(), attackerID, playerName,
					database.EventAttackSuccess, fmt.Sprintf("Captured %s", terrName))
			} else {
				log.Printf("AI: Attack failed, lost the battle")
				h.logHistory(gameID, state.Round, state.Phase.String(), attackerID, playerName,
					database.EventAttackFailed, fmt.Sprintf("Attack on %s failed", terrName))
			}
		}
	} else {
		log.Printf("AI: No favorable attacks (best odds: %.2f), ending conquest", bestOdds)
		state.EndConquest(state.CurrentPlayerID)
	}

	// Save state
	h.saveAndBroadcastAIState(gameID, state)

	// Check if next player is also AI (or if AI has more attacks)
	go h.checkAndTriggerAI(gameID)
}

// aiDevelopment handles the AI's development phase.
func (h *Handlers) aiDevelopment(gameID string, state *game.GameState) {
	player := state.Players[state.CurrentPlayerID]
	if player == nil {
		return
	}

	// Simple AI: Try to build things in order of priority: cities > weapons > boats
	built := false

	// Get player name for history logging
	playerName := state.CurrentPlayerID
	if p, ok := state.Players[state.CurrentPlayerID]; ok {
		playerName = p.Name
	}

	// Try to build a city if we can afford it
	if player.Stockpile.CanAffordStockpile(game.GetBuildCost(game.BuildCity)) || player.Stockpile.Gold >= game.GoldCost(game.BuildCity) {
		// Find a territory without a city
		for id, t := range state.Territories {
			if t.Owner == state.CurrentPlayerID && !t.HasCity {
				useGold := !player.Stockpile.CanAffordStockpile(game.GetBuildCost(game.BuildCity))
				if err := state.Build(state.CurrentPlayerID, game.BuildCity, id, useGold); err == nil {
					log.Printf("AI: Built city at %s", id)
					h.logHistory(gameID, state.Round, state.Phase.String(), state.CurrentPlayerID, playerName,
						database.EventBuild, fmt.Sprintf("Built city on %s", t.Name))
					built = true
					break
				}
			}
		}
	}

	// Try to build a weapon if we can afford it and didn't build a city
	if !built && (player.Stockpile.CanAffordStockpile(game.GetBuildCost(game.BuildWeapon)) || player.Stockpile.Gold >= game.GoldCost(game.BuildWeapon)) {
		// Find a territory without a weapon
		for id, t := range state.Territories {
			if t.Owner == state.CurrentPlayerID && !t.HasWeapon {
				useGold := !player.Stockpile.CanAffordStockpile(game.GetBuildCost(game.BuildWeapon))
				if err := state.Build(state.CurrentPlayerID, game.BuildWeapon, id, useGold); err == nil {
					log.Printf("AI: Built weapon at %s", id)
					h.logHistory(gameID, state.Round, state.Phase.String(), state.CurrentPlayerID, playerName,
						database.EventBuild, fmt.Sprintf("Built weapon on %s", t.Name))
					built = true
					break
				}
			}
		}
	}

	// Try to build a boat if at advanced level and can afford it
	if !built && state.Settings.GameLevel >= game.LevelAdvanced {
		if player.Stockpile.CanAffordStockpile(game.GetBuildCost(game.BuildBoat)) || player.Stockpile.Gold >= game.GoldCost(game.BuildBoat) {
			// Find a coastal territory that can have more boats
			for id, t := range state.Territories {
				if t.Owner == state.CurrentPlayerID && t.IsCoastal() && t.CanAddBoat() {
					useGold := !player.Stockpile.CanAffordStockpile(game.GetBuildCost(game.BuildBoat))
					if err := state.Build(state.CurrentPlayerID, game.BuildBoat, id, useGold); err == nil {
						log.Printf("AI: Built boat at %s", id)
						h.logHistory(gameID, state.Round, state.Phase.String(), state.CurrentPlayerID, playerName,
							database.EventBuild, fmt.Sprintf("Built boat on %s", t.Name))
						built = true
						break
					}
				}
			}
		}
	}

	if !built {
		log.Printf("AI: Nothing to build, ending development")
	}

	// End development phase
	state.EndDevelopment(state.CurrentPlayerID)

	// Save state
	h.saveAndBroadcastAIState(gameID, state)

	// Check if next player is also AI
	go h.checkAndTriggerAI(gameID)
}

// saveAndBroadcastAIState saves the AI's state changes and broadcasts to clients.
func (h *Handlers) saveAndBroadcastAIState(gameID string, state *game.GameState) {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		log.Printf("AI: Failed to marshal state: %v", err)
		return
	}

	if err := h.hub.server.db.SaveGameState(gameID, string(stateJSON),
		state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
		log.Printf("AI: Failed to save state: %v", err)
		return
	}

	h.broadcastGameState(gameID)
}

// handleMoveStockpile handles moving a player's stockpile during shipment phase.
func (h *Handlers) handleMoveStockpile(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.MoveStockpilePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Get destination territory name
	destName := payload.Destination
	if terr, ok := state.Territories[payload.Destination]; ok {
		destName = terr.Name
	}

	// Execute move
	if err := state.MoveStockpile(client.PlayerID, payload.Destination); err != nil {
		return err
	}

	// Log history event
	h.logHistory(client.GameID, state.Round, state.Phase.String(), client.PlayerID, client.Name,
		database.EventStockpileMoved, fmt.Sprintf("Moved stockpile to %s", destName))

	// Save updated state
	stateJSON2, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := h.hub.server.db.SaveGameState(client.GameID, string(stateJSON2),
		state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
		return err
	}

	log.Printf("Player %s moved stockpile to %s", client.Name, payload.Destination)

	// Broadcast updated state
	h.broadcastGameState(client.GameID)

	// Check if next player is AI
	go h.checkAndTriggerAI(client.GameID)

	return nil
}

// handleMoveUnit handles moving a unit during shipment phase (Expert level only).
func (h *Handlers) handleMoveUnit(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.MoveUnitPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Execute move
	if err := state.MoveUnit(client.PlayerID, payload.UnitType, payload.From, payload.To, payload.WaterBodyID, payload.CarryHorse, payload.CarryWeapon); err != nil {
		return err
	}

	// Save updated state
	stateJSON2, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := h.hub.server.db.SaveGameState(client.GameID, string(stateJSON2),
		state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
		return err
	}

	log.Printf("Player %s moved %s from %s to %s", client.Name, payload.UnitType, payload.From, payload.To)

	// Broadcast updated state
	h.broadcastGameState(client.GameID)

	// Check if next player is AI
	go h.checkAndTriggerAI(client.GameID)

	return nil
}

// PendingTrade tracks an ongoing trade waiting for response.
type PendingTrade struct {
	ID              string
	GameID          string
	FromPlayerID    string
	ToPlayerID      string
	OfferCoal       int
	OfferGold       int
	OfferIron       int
	OfferTimber     int
	OfferHorses     int
	OfferHorseTerrs []string
	RequestCoal     int
	RequestGold     int
	RequestIron     int
	RequestTimber   int
	RequestHorses   int
	ResponseChan    chan bool
}

// handleProposeTrade handles a player proposing a trade to another player.
func (h *Handlers) handleProposeTrade(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.ProposeTradePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Check it's trade phase and player's turn
	if state.Phase != game.PhaseTrade {
		return game.ErrInvalidAction
	}
	if state.CurrentPlayerID != client.PlayerID {
		return game.ErrNotYourTurn
	}

	// Create trade offer
	offer := &game.TradeOffer{
		FromPlayerID:    client.PlayerID,
		ToPlayerID:      payload.TargetPlayer,
		OfferCoal:       payload.OfferCoal,
		OfferGold:       payload.OfferGold,
		OfferIron:       payload.OfferIron,
		OfferTimber:     payload.OfferTimber,
		OfferHorses:     payload.OfferHorses,
		OfferHorseTerrs: payload.OfferHorseTerrs,
		RequestCoal:     payload.RequestCoal,
		RequestGold:     payload.RequestGold,
		RequestIron:     payload.RequestIron,
		RequestTimber:   payload.RequestTimber,
		RequestHorses:   payload.RequestHorses,
	}

	// Validate the trade
	if err := state.ValidateTrade(offer); err != nil {
		return err
	}

	// Check if target player is online
	if !h.hub.IsPlayerOnline(payload.TargetPlayer) {
		result := protocol.TradeResultPayload{
			TradeID:  msg.ID,
			Accepted: false,
			Message:  "Player is offline",
		}
		respMsg, _ := protocol.NewMessage(protocol.TypeTradeResult, result)
		client.Send(respMsg)
		return nil
	}

	// Check if target is AI - AI always rejects
	targetPlayer := state.Players[payload.TargetPlayer]
	if targetPlayer != nil && targetPlayer.IsAI {
		result := protocol.TradeResultPayload{
			TradeID:  msg.ID,
			Accepted: false,
			Message:  "AI declined the trade",
		}
		respMsg, _ := protocol.NewMessage(protocol.TypeTradeResult, result)
		client.Send(respMsg)
		return nil
	}

	// Create pending trade
	tradeID := fmt.Sprintf("trade-%s-%d", client.GameID, time.Now().UnixNano())
	trade := &PendingTrade{
		ID:              tradeID,
		GameID:          client.GameID,
		FromPlayerID:    client.PlayerID,
		ToPlayerID:      payload.TargetPlayer,
		OfferCoal:       payload.OfferCoal,
		OfferGold:       payload.OfferGold,
		OfferIron:       payload.OfferIron,
		OfferTimber:     payload.OfferTimber,
		OfferHorses:     payload.OfferHorses,
		OfferHorseTerrs: payload.OfferHorseTerrs,
		RequestCoal:     payload.RequestCoal,
		RequestGold:     payload.RequestGold,
		RequestIron:     payload.RequestIron,
		RequestTimber:   payload.RequestTimber,
		RequestHorses:   payload.RequestHorses,
		ResponseChan:    make(chan bool, 1),
	}

	// Store pending trade
	h.hub.mu.Lock()
	if h.hub.pendingTrades == nil {
		h.hub.pendingTrades = make(map[string]*PendingTrade)
	}
	h.hub.pendingTrades[tradeID] = trade
	h.hub.mu.Unlock()

	// Send proposal to target player
	proposerName := client.Name
	proposal := protocol.TradeProposalPayload{
		TradeID:        tradeID,
		FromPlayerID:   client.PlayerID,
		FromPlayerName: proposerName,
		OfferCoal:      payload.OfferCoal,
		OfferGold:      payload.OfferGold,
		OfferIron:      payload.OfferIron,
		OfferTimber:    payload.OfferTimber,
		OfferHorses:    payload.OfferHorses,
		RequestCoal:    payload.RequestCoal,
		RequestGold:    payload.RequestGold,
		RequestIron:    payload.RequestIron,
		RequestTimber:  payload.RequestTimber,
		RequestHorses:  payload.RequestHorses,
	}

	h.hub.sendToPlayer(payload.TargetPlayer, protocol.TypeTradeProposal, proposal)
	log.Printf("Trade proposal %s sent from %s to %s", tradeID, client.Name, payload.TargetPlayer)

	// Wait for response (synchronous - blocks until response or timeout)
	// Timeout after 60 seconds
	select {
	case <-trade.ResponseChan:
		// Response received, handled in handleRespondTrade
	case <-time.After(60 * time.Second):
		// Timeout - reject trade
		h.hub.mu.Lock()
		delete(h.hub.pendingTrades, tradeID)
		h.hub.mu.Unlock()

		result := protocol.TradeResultPayload{
			TradeID:  tradeID,
			Accepted: false,
			Message:  "Trade timed out",
		}
		respMsg, _ := protocol.NewMessage(protocol.TypeTradeResult, result)
		client.Send(respMsg)
		log.Printf("Trade %s timed out", tradeID)
	}

	return nil
}

// handleRespondTrade handles a player responding to a trade proposal.
func (h *Handlers) handleRespondTrade(client *Client, msg *protocol.Message) error {
	var payload protocol.RespondTradePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Get the pending trade
	h.hub.mu.Lock()
	trade, exists := h.hub.pendingTrades[payload.TradeID]
	if exists {
		delete(h.hub.pendingTrades, payload.TradeID)
	}
	h.hub.mu.Unlock()

	if !exists {
		return errors.New("trade not found or expired")
	}

	// Verify this is the target player
	if trade.ToPlayerID != client.PlayerID {
		return errors.New("not the trade target")
	}

	if payload.Accepted {
		// Load game state
		stateJSON, err := h.hub.server.db.GetGameState(trade.GameID)
		if err != nil {
			trade.ResponseChan <- false
			return err
		}

		var state game.GameState
		if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
			trade.ResponseChan <- false
			return err
		}

		// Build the offer
		offer := &game.TradeOffer{
			FromPlayerID:    trade.FromPlayerID,
			ToPlayerID:      trade.ToPlayerID,
			OfferCoal:       trade.OfferCoal,
			OfferGold:       trade.OfferGold,
			OfferIron:       trade.OfferIron,
			OfferTimber:     trade.OfferTimber,
			OfferHorses:     trade.OfferHorses,
			OfferHorseTerrs: trade.OfferHorseTerrs,
			RequestCoal:     trade.RequestCoal,
			RequestGold:     trade.RequestGold,
			RequestIron:     trade.RequestIron,
			RequestTimber:   trade.RequestTimber,
			RequestHorses:   trade.RequestHorses,
		}

		// Get horse source territories from the target player
		// For requested horses, the target needs to specify which territories
		horseSourceTerrs := make([]string, 0)
		if trade.RequestHorses > 0 {
			// For simplicity, auto-select horses from any owned territories
			for id, terr := range state.Territories {
				if terr.Owner == trade.ToPlayerID && terr.HasHorse {
					horseSourceTerrs = append(horseSourceTerrs, id)
					if len(horseSourceTerrs) >= trade.RequestHorses {
						break
					}
				}
			}
		}

		// Execute the trade
		if err := state.ExecuteTrade(offer, horseSourceTerrs, payload.HorseDestinations); err != nil {
			trade.ResponseChan <- false
			return err
		}

		// Save state
		stateJSON2, err := json.Marshal(state)
		if err != nil {
			trade.ResponseChan <- false
			return err
		}

		if err := h.hub.server.db.SaveGameState(trade.GameID, string(stateJSON2),
			state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
			trade.ResponseChan <- false
			return err
		}

		// Send success to proposer
		result := protocol.TradeResultPayload{
			TradeID:  payload.TradeID,
			Accepted: true,
			Message:  "Trade accepted!",
		}
		h.hub.sendToPlayer(trade.FromPlayerID, protocol.TypeTradeResult, result)

		// Broadcast updated state
		h.broadcastGameState(trade.GameID)

		log.Printf("Trade %s accepted by %s", payload.TradeID, client.Name)
	} else {
		// Send rejection to proposer
		result := protocol.TradeResultPayload{
			TradeID:  payload.TradeID,
			Accepted: false,
			Message:  "Trade declined",
		}
		h.hub.sendToPlayer(trade.FromPlayerID, protocol.TypeTradeResult, result)

		log.Printf("Trade %s rejected by %s", payload.TradeID, client.Name)
	}

	// Signal that we got a response
	trade.ResponseChan <- payload.Accepted

	return nil
}

// handleEndPhase handles a player ending their turn in the current phase.
func (h *Handlers) handleEndPhase(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Check it's the player's turn
	if state.CurrentPlayerID != client.PlayerID {
		return game.ErrNotYourTurn
	}

	// Remember the current phase for logging
	endedPhase := state.Phase.String()

	// Handle based on current phase
	switch state.Phase {
	case game.PhaseTrade:
		if err := state.SkipTrade(client.PlayerID); err != nil {
			return err
		}
	case game.PhaseShipment:
		if err := state.SkipShipment(client.PlayerID); err != nil {
			return err
		}
	case game.PhaseConquest:
		// End conquest phase for this player
		state.EndConquest(client.PlayerID)
	case game.PhaseDevelopment:
		// End development phase for this player
		state.EndDevelopment(client.PlayerID)
	default:
		return errors.New("cannot end phase in current state")
	}

	// Save updated state
	stateJSON2, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := h.hub.server.db.SaveGameState(client.GameID, string(stateJSON2),
		state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
		return err
	}

	log.Printf("Player %s ended their %s phase", client.Name, endedPhase)

	// Broadcast updated state
	h.broadcastGameState(client.GameID)

	// Check if next player is AI
	go h.checkAndTriggerAI(client.GameID)

	return nil
}

// handlePlanAttack handles getting a preview of an attack.
func (h *Handlers) handlePlanAttack(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.PlanAttackPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Get attack plan/preview
	plan := state.GetAttackPlan(client.PlayerID, payload.TargetTerritory)
	if plan == nil {
		return game.ErrInvalidTarget
	}

	// Get target territory for ally calculations
	target := state.Territories[payload.TargetTerritory]
	defenderID := ""
	if target != nil {
		defenderID = target.Owner
	}

	// Calculate ally contributions for preview
	attackerAllies := []string{}
	defenderAllies := []string{}
	if target != nil {
		thirdParties := state.GetThirdPartyPlayers(client.PlayerID, target)
		for _, tpID := range thirdParties {
			tp := state.Players[tpID]
			if tp == nil {
				continue
			}
			alliance := string(tp.Alliance)
			switch alliance {
			case "neutral", "ask", "":
				continue
			case "defender":
				defenderAllies = append(defenderAllies, tpID)
			default:
				if alliance == client.PlayerID {
					attackerAllies = append(attackerAllies, tpID)
				} else if alliance == defenderID {
					defenderAllies = append(defenderAllies, tpID)
				}
			}
		}
	}

	// Calculate strengths with allies
	attackStrength := plan.AttackStrength
	defenseStrength := plan.DefenseStrength
	attackerAllyStrength := 0
	defenderAllyStrength := 0
	if target != nil {
		// Add attacker ally strength
		for _, allyID := range attackerAllies {
			allyStr := state.CalculatePlayerStrengthAtTerritory(allyID, target)
			attackerAllyStrength += allyStr
			attackStrength += allyStr
		}
		// Add defender ally strength
		for _, allyID := range defenderAllies {
			allyStr := state.CalculatePlayerStrengthAtTerritory(allyID, target)
			defenderAllyStrength += allyStr
			defenseStrength += allyStr
		}
	}

	// Convert to protocol format
	reinforcements := make([]protocol.ReinforcementOption, 0, len(plan.Reinforcements))
	for _, r := range plan.Reinforcements {
		opt := protocol.ReinforcementOption{
			UnitType:      r.UnitType,
			From:          r.FromTerritory,
			WaterBodyID:   r.WaterBodyID,
			StrengthBonus: r.Strength,
		}
		for _, carry := range r.CanCarry {
			if carry == "weapon" {
				opt.CanCarryWeapon = true
			}
			if carry == "horse" {
				opt.CanCarryHorse = true
			}
		}
		reinforcements = append(reinforcements, opt)
	}

	// Send preview
	preview := protocol.AttackPreviewPayload{
		TargetTerritory:         plan.TargetID,
		AttackStrength:          attackStrength,
		DefenseStrength:         defenseStrength,
		AttackerAllyStrength:    attackerAllyStrength,
		DefenderAllyStrength:    defenderAllyStrength,
		CanAttack:               plan.CanAttack,
		AvailableReinforcements: reinforcements,
	}

	respMsg, _ := protocol.NewMessage(protocol.TypeAttackPreview, preview)
	respMsg.ID = msg.ID
	client.Send(respMsg)

	return nil
}

// handleExecuteAttack handles executing an attack.
func (h *Handlers) handleExecuteAttack(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.ExecuteAttackPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Build brought unit if specified
	var brought *game.BroughtUnit
	if payload.BringUnit != "" && payload.BringFrom != "" {
		brought = &game.BroughtUnit{
			FromTerritory: payload.BringFrom,
			WaterBodyID:   payload.WaterBodyID,
		}
		switch payload.BringUnit {
		case "horse":
			brought.UnitType = game.UnitHorse
			if payload.CarryWeapon {
				brought.CarryingWeapon = true
				brought.WeaponFromTerritory = payload.WeaponFrom
				if brought.WeaponFromTerritory == "" {
					brought.WeaponFromTerritory = payload.BringFrom
				}
			}
		case "weapon":
			brought.UnitType = game.UnitWeapon
		case "boat":
			brought.UnitType = game.UnitBoat
			if payload.CarryWeapon {
				brought.CarryingWeapon = true
				brought.WeaponFromTerritory = payload.WeaponFrom
				if brought.WeaponFromTerritory == "" {
					brought.WeaponFromTerritory = payload.BringFrom
				}
			}
			if payload.CarryHorse {
				brought.CarryingHorse = true
				brought.HorseFromTerritory = payload.HorseFrom
				if brought.HorseFromTerritory == "" {
					brought.HorseFromTerritory = payload.BringFrom
				}
			}
		}
	}

	// Get territory info before attack
	target := state.Territories[payload.TargetTerritory]
	if target == nil {
		return errors.New("target territory not found")
	}
	terrName := target.Name
	defenderID := target.Owner

	// Collect allies based on alliance settings
	attackerAllies := []string{}
	defenderAllies := []string{}
	askPlayers := []string{} // Players with "ask" setting who are online
	thirdParties := state.GetThirdPartyPlayers(client.PlayerID, target)

	log.Printf("Battle at %s: Found %d third party players: %v", terrName, len(thirdParties), thirdParties)

	for _, tpID := range thirdParties {
		tp := state.Players[tpID]
		if tp == nil {
			log.Printf("  Third party %s: player not found in state", tpID)
			continue
		}

		// Determine which side this player supports based on their alliance setting
		alliance := string(tp.Alliance)
		log.Printf("  Third party %s (%s): alliance setting = '%s'", tpID, tp.Name, alliance)

		switch alliance {
		case "neutral", "":
			// Neutral players don't participate
			log.Printf("    -> Neutral (not participating)")
			continue
		case "ask":
			// Check if player is online
			if h.hub.IsPlayerOnline(tpID) {
				log.Printf("    -> Ask (online, will be prompted)")
				askPlayers = append(askPlayers, tpID)
			} else {
				log.Printf("    -> Ask (offline, treated as neutral)")
			}
		case "defender":
			// Always support the defender
			log.Printf("    -> Supporting defender")
			defenderAllies = append(defenderAllies, tpID)
		default:
			// Check if it's a specific player alliance
			if alliance == client.PlayerID {
				// Allied with attacker
				log.Printf("    -> Supporting attacker (allied with %s)", client.PlayerID)
				attackerAllies = append(attackerAllies, tpID)
			} else if alliance == defenderID {
				// Allied with defender
				log.Printf("    -> Supporting defender (allied with %s)", defenderID)
				defenderAllies = append(defenderAllies, tpID)
			} else {
				log.Printf("    -> Allied with someone else (%s), not participating", alliance)
			}
		}
	}

	// If there are "ask" players, we need to wait for their votes
	if len(askPlayers) > 0 {
		log.Printf("Waiting for %d alliance votes from: %v", len(askPlayers), askPlayers)

		// Create a pending battle
		battleID := fmt.Sprintf("battle-%s-%d", client.GameID, time.Now().UnixNano())
		battle := &PendingBattle{
			ID:           battleID,
			GameID:       client.GameID,
			AttackerID:   client.PlayerID,
			DefenderID:   defenderID,
			TerritoryID:  payload.TargetTerritory,
			ThirdParties: askPlayers,
			Votes:        make(map[string]string),
			VoteChan:     make(chan string, len(askPlayers)),
			ExpiresAt:    time.Now().Add(60 * time.Second),
		}

		h.hub.mu.Lock()
		h.hub.pendingBattles[battleID] = battle
		h.hub.mu.Unlock()

		// Send alliance requests to all "ask" players
		for _, askID := range askPlayers {
			askPlayer := state.Players[askID]
			askStrength := state.CalculatePlayerStrengthAtTerritory(askID, target)

			attackerName := client.Name
			defenderName := "Unclaimed"
			if defender := state.Players[defenderID]; defender != nil {
				defenderName = defender.Name
			}

			request := protocol.AllianceRequestPayload{
				BattleID:      battleID,
				AttackerID:    client.PlayerID,
				AttackerName:  attackerName,
				DefenderID:    defenderID,
				DefenderName:  defenderName,
				TerritoryID:   payload.TargetTerritory,
				TerritoryName: terrName,
				YourStrength:  askStrength,
				TimeLimit:     60,
				ExpiresAt:     battle.ExpiresAt.Unix(),
			}

			log.Printf("Sending alliance request to %s (%s)", askID, askPlayer.Name)
			h.hub.sendToPlayer(askID, protocol.TypeAllianceRequest, request)
		}

		// Wait for votes or timeout
		votesReceived := 0
		timeout := time.After(60 * time.Second)

	voteLoop:
		for votesReceived < len(askPlayers) {
			select {
			case <-battle.VoteChan:
				votesReceived++
				log.Printf("Received vote %d/%d", votesReceived, len(askPlayers))
			case <-timeout:
				log.Printf("Alliance vote timeout, proceeding with %d/%d votes", votesReceived, len(askPlayers))
				break voteLoop
			}
		}

		// Collect the votes
		h.hub.mu.Lock()
		for playerID, side := range battle.Votes {
			switch side {
			case "attacker":
				attackerAllies = append(attackerAllies, playerID)
				log.Printf("  %s voted for attacker", playerID)
			case "defender":
				defenderAllies = append(defenderAllies, playerID)
				log.Printf("  %s voted for defender", playerID)
			default:
				log.Printf("  %s voted neutral", playerID)
			}
		}
		// Clean up the pending battle
		delete(h.hub.pendingBattles, battleID)
		h.hub.mu.Unlock()
	}

	log.Printf("Battle at %s: Final allies - Attacker: %v, Defender: %v", terrName, attackerAllies, defenderAllies)

	// Re-load game state in case it changed during voting
	stateJSON, err = h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Execute attack with ally information
	result, err := state.AttackWithAllies(client.PlayerID, payload.TargetTerritory, brought, attackerAllies, defenderAllies)
	if err != nil {
		return err
	}

	// Log history event
	if result.AttackerWins {
		h.logHistory(client.GameID, state.Round, state.Phase.String(), client.PlayerID, client.Name,
			database.EventAttackSuccess, fmt.Sprintf("Captured %s", terrName))
	} else {
		h.logHistory(client.GameID, state.Round, state.Phase.String(), client.PlayerID, client.Name,
			database.EventAttackFailed, fmt.Sprintf("Attack on %s failed", terrName))
	}

	// Save updated state
	stateJSON2, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := h.hub.server.db.SaveGameState(client.GameID, string(stateJSON2),
		state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
		return err
	}

	// Convert result to protocol format
	unitsDestroyed := make([]string, 0)
	for _, u := range result.UnitsDestroyed {
		unitsDestroyed = append(unitsDestroyed, u.TerritoryID)
	}
	unitsCaptured := make([]string, 0)
	for _, u := range result.UnitsCaptured {
		unitsCaptured = append(unitsCaptured, u.TerritoryID)
	}

	// Broadcast combat result to ALL players so everyone sees the animation
	combatResult := protocol.CombatResultPayload{
		Success:         true,
		AttackerID:      client.PlayerID,
		AttackerWins:    result.AttackerWins,
		AttackStrength:  result.AttackStrength,
		DefenseStrength: result.DefenseStrength,
		TargetTerritory: payload.TargetTerritory,
		UnitsDestroyed:  unitsDestroyed,
		UnitsCaptured:   unitsCaptured,
	}

	h.hub.notifyGamePlayers(client.GameID, protocol.TypeActionResult, combatResult)

	if result.AttackerWins {
		log.Printf("Player %s conquered %s", client.Name, payload.TargetTerritory)
	} else {
		log.Printf("Player %s failed to conquer %s", client.Name, payload.TargetTerritory)
	}

	// Check for game over BEFORE broadcasting state
	if state.IsGameOver() {
		h.handleGameOver(client.GameID, &state)
		return nil
	}

	// Broadcast updated state
	h.broadcastGameState(client.GameID)

	// Check if next player is AI
	go h.checkAndTriggerAI(client.GameID)

	return nil
}

// handleBuild handles building a unit or city.
func (h *Handlers) handleBuild(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.BuildPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Load game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	// Convert type string to BuildType
	var buildType game.BuildType
	switch payload.Type {
	case "city":
		buildType = game.BuildCity
	case "weapon":
		buildType = game.BuildWeapon
	case "boat":
		buildType = game.BuildBoat
	default:
		return game.ErrInvalidTarget
	}

	// Get territory name before build
	terrName := payload.Territory
	if terr, ok := state.Territories[payload.Territory]; ok {
		terrName = terr.Name
	}

	// Execute build - use water body specific version for boats if specified
	if buildType == game.BuildBoat && payload.WaterBodyID != "" {
		if err := state.BuildBoatInWater(client.PlayerID, payload.Territory, payload.WaterBodyID, payload.UseGold); err != nil {
			return err
		}
	} else {
		if err := state.Build(client.PlayerID, buildType, payload.Territory, payload.UseGold); err != nil {
			return err
		}
	}

	// Log history event
	h.logHistory(client.GameID, state.Round, state.Phase.String(), client.PlayerID, client.Name,
		database.EventBuild, fmt.Sprintf("Built %s on %s", payload.Type, terrName))

	// Save updated state
	stateJSON2, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := h.hub.server.db.SaveGameState(client.GameID, string(stateJSON2),
		state.CurrentPlayerID, state.Round, state.Phase.String()); err != nil {
		return err
	}

	log.Printf("Player %s built %s at %s", client.Name, payload.Type, payload.Territory)

	// Check for game over (building 6th city wins)
	if state.IsGameOver() {
		h.handleGameOver(client.GameID, &state)
		return nil
	}

	// Broadcast updated state
	h.broadcastGameState(client.GameID)

	return nil
}

// ==================== History Logging ====================

// logHistory adds an event to the game history log.
func (h *Handlers) logHistory(gameID string, round int, phase, playerID, playerName, eventType, message string) {
	if err := h.hub.server.db.AddHistoryEvent(gameID, round, phase, playerID, playerName, eventType, message); err != nil {
		log.Printf("Failed to log history event: %v", err)
	}
}

// sendGameHistory sends the game history to a client.
func (h *Handlers) sendGameHistory(client *Client, gameID string) {
	events, err := h.hub.server.db.GetGameHistory(gameID)
	if err != nil {
		log.Printf("Failed to get game history: %v", err)
		return
	}

	payload := protocol.GameHistoryPayload{
		Events: make([]protocol.HistoryEvent, len(events)),
	}
	for i, e := range events {
		payload.Events[i] = protocol.HistoryEvent{
			ID:         e.ID,
			Round:      e.Round,
			Phase:      e.Phase,
			PlayerID:   e.PlayerID,
			PlayerName: e.PlayerName,
			EventType:  e.EventType,
			Message:    e.Message,
		}
	}

	msg, _ := protocol.NewMessage(protocol.TypeGameHistory, payload)
	client.Send(msg)
}

// broadcastGameHistory sends the game history to all players in a game.
func (h *Handlers) broadcastGameHistory(gameID string) {
	h.hub.mu.RLock()
	clients := h.hub.gameClients[gameID]
	h.hub.mu.RUnlock()

	for client := range clients {
		h.sendGameHistory(client, gameID)
	}
}

// handleSetAlliance sets a player's alliance preference.
func (h *Handlers) handleSetAlliance(client *Client, msg *protocol.Message) error {
	log.Printf("handleSetAlliance called by player %s in game %s", client.PlayerID, client.GameID)

	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.SetAlliancePayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("Failed to parse SetAlliance payload: %v", err)
		return err
	}

	log.Printf("Player %s wants to set alliance to: '%s'", client.PlayerID, payload.Setting)

	// Validate setting
	setting := payload.Setting
	if setting != "ask" && setting != "neutral" && setting != "defender" {
		// Must be a player ID - verify it exists in the game
		players, err := h.hub.server.db.GetGamePlayers(client.GameID)
		if err != nil {
			return err
		}
		validPlayer := false
		for _, p := range players {
			if p.PlayerID == setting && p.PlayerID != client.PlayerID {
				validPlayer = true
				break
			}
		}
		if !validPlayer {
			return errors.New("invalid alliance setting")
		}
	}

	// Save to database
	if err := h.hub.server.db.SetAllianceSetting(client.GameID, client.PlayerID, setting); err != nil {
		return err
	}

	// Update game state
	stateJSON, err := h.hub.server.db.GetGameState(client.GameID)
	if err != nil {
		return err
	}

	var state game.GameState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return err
	}

	if player := state.Players[client.PlayerID]; player != nil {
		log.Printf("Updating player %s alliance from '%s' to '%s'", client.PlayerID, player.Alliance, setting)
		player.Alliance = game.AllianceSetting(setting)
	} else {
		log.Printf("WARNING: Player %s not found in game state!", client.PlayerID)
	}

	// Save updated state
	newStateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := h.hub.server.db.SaveGameState(client.GameID, string(newStateJSON), state.CurrentPlayerID, state.Round, string(state.Phase)); err != nil {
		log.Printf("Failed to save game state: %v", err)
		return err
	}

	// Verify the save by re-reading
	verifyJSON, _ := h.hub.server.db.GetGameState(client.GameID)
	var verifyState game.GameState
	json.Unmarshal([]byte(verifyJSON), &verifyState)
	if verifyPlayer := verifyState.Players[client.PlayerID]; verifyPlayer != nil {
		log.Printf("Verified: Player %s alliance is now '%s' in saved state", client.PlayerID, verifyPlayer.Alliance)
	}

	// Broadcast updated state to all players
	h.broadcastGameState(client.GameID)

	log.Printf("Player %s set alliance to: %s - SUCCESS", client.PlayerID, setting)
	return nil
}

// handleAllianceVote handles a player's vote during a battle.
func (h *Handlers) handleAllianceVote(client *Client, msg *protocol.Message) error {
	if client.GameID == "" {
		return errors.New("not in a game")
	}

	var payload protocol.AllianceVotePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return err
	}

	// Validate side
	if payload.Side != "attacker" && payload.Side != "defender" && payload.Side != "neutral" {
		return errors.New("invalid alliance side")
	}

	// Record the vote in the pending battle
	h.hub.mu.Lock()
	battle := h.hub.pendingBattles[payload.BattleID]
	if battle != nil {
		battle.Votes[client.PlayerID] = payload.Side
		// Signal that we received a vote
		select {
		case battle.VoteChan <- client.PlayerID:
		default:
		}
	}
	h.hub.mu.Unlock()

	if battle == nil {
		return errors.New("battle not found or already resolved")
	}

	// Send confirmation
	result := protocol.AllianceResultPayload{
		BattleID: payload.BattleID,
		Accepted: true,
	}
	respMsg, _ := protocol.NewMessage(protocol.TypeAllianceResult, result)
	client.Send(respMsg)

	log.Printf("Player %s voted %s for battle %s", client.PlayerID, payload.Side, payload.BattleID)
	return nil
}

// handleGameOver handles the end of a game.
func (h *Handlers) handleGameOver(gameID string, state *game.GameState) {
	winner := state.GetWinner()
	if winner == nil {
		log.Printf("Game over but no winner found for game %s", gameID)
		return
	}

	// Determine the reason for winning
	reason := "elimination"
	if state.CountCities(winner.ID) >= state.Settings.VictoryCities {
		reason = "cities"
	}

	log.Printf("Game %s ended! Winner: %s (%s) by %s", gameID, winner.Name, winner.ID, reason)

	// Update game status in database - log error if it fails
	if err := h.hub.server.db.EndGame(gameID, winner.ID, reason); err != nil {
		log.Printf("ERROR: Failed to mark game %s as finished: %v", gameID, err)
	}

	// Log the victory
	h.logHistory(gameID, state.Round, state.Phase.String(), winner.ID, winner.Name,
		database.EventGameEnd, fmt.Sprintf("%s wins by %s!", winner.Name, reason))

	// Broadcast final state first so clients have it
	h.broadcastGameState(gameID)

	// Send game ended message to all players
	payload := protocol.GameEndedPayload{
		WinnerID:   winner.ID,
		WinnerName: winner.Name,
		Reason:     reason,
	}

	h.hub.notifyGamePlayers(gameID, protocol.TypeGameEnded, payload)
}
