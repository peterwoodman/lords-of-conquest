package server

import (
	"errors"
	"log"

	"lords-of-conquest/internal/database"
	"lords-of-conquest/internal/protocol"
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
	case protocol.TypeAddAI:
		err = h.handleAddAI(client, msg)
	case protocol.TypePlayerReady:
		err = h.handlePlayerReady(client, msg)
	case protocol.TypeStartGame:
		err = h.handleStartGame(client, msg)
	case protocol.TypeListGames:
		err = h.handleListGames(client, msg)
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

	game, err := h.hub.server.db.CreateGame(payload.Name, client.PlayerID, settings, payload.IsPublic)
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

	// Send lobby state to this client and notify others
	h.sendLobbyState(client, gameID)
	h.hub.notifyGamePlayers(gameID, protocol.TypePlayerJoined, protocol.PlayerJoinedPayload{
		PlayerID: client.PlayerID,
		Name:     client.Name,
	})

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

	// TODO: Initialize game state and save to database

	// Notify all players
	h.hub.notifyGamePlayers(client.GameID, protocol.TypeGameStarted, protocol.GameStartedPayload{
		GameID: client.GameID,
	})

	return nil
}

// handleListGames handles listing public games.
func (h *Handlers) handleListGames(client *Client, msg *protocol.Message) error {
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
