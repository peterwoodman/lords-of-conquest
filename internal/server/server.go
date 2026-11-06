// Package server implements the Lords of Conquest game server.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"lords-of-conquest/internal/database"
	game "lords-of-conquest/internal/game"
	"lords-of-conquest/internal/protocol"
	"lords-of-conquest/pkg/maps"

	"github.com/coder/websocket"
)

// Server is the main game server.
type Server struct {
	db     *database.DB
	hub    *Hub
	addr   string
	server *http.Server
}

// Config holds server configuration.
type Config struct {
	Addr   string
	DBPath string
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	db, err := database.New(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Server{
		db:   db,
		addr: cfg.Addr,
	}

	s.hub = NewHub(s)

	return s, nil
}

// Start starts the server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.handleWebSocket)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// API endpoints for listing games (can be used by web clients too)
	mux.HandleFunc("/api/games", s.handleListGames)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("Lords of Conquest Server")
	log.Printf("  Address: http://localhost%s", s.addr)
	log.Printf("  Database: %s", "data/lords.db")
	log.Printf("  WebSocket: ws://localhost%s/ws", s.addr)
	log.Printf("")
	log.Printf("Press Ctrl+C to stop")

	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return err
		}
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// handleWebSocket upgrades HTTP connections to WebSocket.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket connection from %s", r.RemoteAddr)

	// Accept WebSocket connection with options
	opts := &websocket.AcceptOptions{
		CompressionMode:      websocket.CompressionContextTakeover,
		CompressionThreshold: 512,
		OriginPatterns:       []string{"*"}, // Allow all origins for now
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		log.Printf("Accept failed: %v", err)
		return
	}

	client := NewClient(s.hub, conn)

	// Register before reading so a disconnect can never be processed first
	s.hub.Register(client)

	go client.processInbox()
	go client.WritePump()
	go client.ReadPump()
}

// handleListGames returns a list of public games.
func (s *Server) handleListGames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	games, err := s.db.ListPublicGames()
	if err != nil {
		http.Error(w, "Failed to list games", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

// PendingBattle tracks an ongoing battle waiting for alliance votes.
type PendingBattle struct {
	ID           string
	GameID       string
	AttackerID   string
	DefenderID   string
	TerritoryID  string
	ThirdParties []string          // Player IDs who can vote
	Votes        map[string]string // PlayerID -> "attacker", "defender", or "neutral"
	VoteChan     chan string       // Signals when votes arrive
	ExpiresAt    time.Time
}

// PendingEvent tracks an event that requires client acknowledgment before proceeding.
type PendingEvent struct {
	ID           string
	Type         string                  // combat, phase_skip, phase_change, etc.
	GameID       string
	RequiredAcks map[string]bool         // PlayerID -> acknowledged (true = received ack)
	Timeout      time.Time
	OnComplete   func()                  // Called when all acks received or timeout
	completed    bool                    // Prevent double-completion
}

// PendingCardBattle tracks a card combat waiting for the defender's card selection.
type PendingCardBattle struct {
	ID               string
	GameID           string
	AttackerID       string
	DefenderID       string
	TargetTerritory  string
	BroughtUnit      *game.BroughtUnit
	AttackerAllies   []string
	DefenderAllies   []string
	AttackCards      []game.CombatCard  // Cards the attacker committed
	DefenseCardsChan chan []string       // Channel to receive defender's card IDs
}

// PendingAttackPlan stores a resolved attack plan waiting for confirmation.
type PendingAttackPlan struct {
	ID               string
	GameID           string
	AttackerID       string
	TargetTerritory  string
	BringUnit        string
	BringFrom        string
	WaterBodyID      string
	CarryWeapon      bool
	WeaponFrom       string
	CarryHorse       bool
	HorseFrom        string
	AttackerAllies   []string // Resolved ally player IDs supporting attacker
	DefenderAllies   []string // Resolved ally player IDs supporting defender
	ExpiresAt        time.Time
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	server *Server

	// Registered clients
	clients map[*Client]bool

	// Clients by player ID
	playerClients map[string]*Client

	// Clients in each game
	gameClients map[string]map[*Client]bool

	// Pending battles waiting for alliance votes
	pendingBattles map[string]*PendingBattle

	// Pending trades waiting for responses
	pendingTrades map[string]*PendingTrade

	// Pending events waiting for client acknowledgment
	pendingEvents map[string]*PendingEvent

	// Pending attack plans waiting for confirmation
	pendingAttackPlans map[string]*PendingAttackPlan

	// Pending card battles waiting for defender's card selection
	pendingCardBattles map[string]*PendingCardBattle

	// Processed maps by game ID (each game stores its own map in the database)
	gameMaps map[string]*maps.Map

	// Per-game locks serializing load-modify-save of game state
	gameLocks map[string]*sync.Mutex

	mu sync.RWMutex
}

// NewHub creates a new Hub.
func NewHub(server *Server) *Hub {
	return &Hub{
		server:             server,
		clients:            make(map[*Client]bool),
		playerClients:      make(map[string]*Client),
		gameClients:        make(map[string]map[*Client]bool),
		pendingBattles:     make(map[string]*PendingBattle),
		pendingEvents:      make(map[string]*PendingEvent),
		pendingAttackPlans: make(map[string]*PendingAttackPlan),
		pendingCardBattles: make(map[string]*PendingCardBattle),
		gameMaps:           make(map[string]*maps.Map),
		gameLocks:          make(map[string]*sync.Mutex),
	}
}

// Register adds a client to the hub and greets it. Call before starting the client's pumps.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()

	h.sendWelcome(client)
}

// Unregister disconnects a client. The cleanup runs on the client's own message
// goroutine, after any message it is already handling.
func (h *Hub) Unregister(client *Client) {
	client.post(func() { h.handleDisconnect(client) })
}

// dispatch routes a message from a client. Replies that another handler may be
// blocked waiting for run immediately; everything else runs in the order received.
func (h *Hub) dispatch(client *Client, msg *protocol.Message) {
	if isReplyMessage(msg.Type) {
		// Replies run off the inbox and read PlayerID/Name, which are only
		// stable once authentication has completed
		if !h.isAuthenticated(client) {
			return
		}
		go h.handleMessage(client, msg)
		return
	}
	client.post(func() { h.handleMessage(client, msg) })
}

// isReplyMessage reports message types that answer a request a handler is waiting on.
// They must not queue behind the handler that is waiting for them.
func isReplyMessage(t protocol.MessageType) bool {
	switch t {
	case protocol.TypeClientReady, protocol.TypeAllianceVote,
		protocol.TypeSelectDefenseCards, protocol.TypeRespondTrade:
		return true
	}
	return false
}

// sendWelcome sends a welcome message to a new client.
func (h *Hub) sendWelcome(client *Client) {
	payload := protocol.WelcomePayload{
		ServerVersion: "0.1.0",
	}
	msg, _ := protocol.NewMessage(protocol.TypeWelcome, payload)
	client.Send(msg)
}

// handleDisconnect handles a client disconnecting.
func (h *Hub) handleDisconnect(client *Client) {
	var gameID, playerID string
	var shouldNotify bool

	// First, update state while holding the lock
	h.mu.Lock()
	if _, ok := h.clients[client]; !ok {
		h.mu.Unlock()
		return
	}

	delete(h.clients, client)

	// Remove from player mapping, unless the player has already reconnected on a new socket
	stillCurrent := false
	if client.PlayerID != "" {
		if h.playerClients[client.PlayerID] == client {
			delete(h.playerClients, client.PlayerID)
			stillCurrent = true
		}

		if client.GameID != "" {
			if gameClients, ok := h.gameClients[client.GameID]; ok {
				delete(gameClients, client)
			}
			if stillCurrent {
				gameID = client.GameID
				playerID = client.PlayerID
				shouldNotify = true
			}
		}
	}

	client.closeSend()
	h.mu.Unlock()

	// Do database and notification AFTER releasing the lock
	if gameID != "" {
		h.server.db.SetPlayerConnected(gameID, playerID, false)
	}

	if shouldNotify {
		h.notifyGamePlayers(gameID, protocol.TypeDisconnect, protocol.DisconnectPayload{
			PlayerID: playerID,
			Reason:   "disconnected",
		})
	}
}

// handleMessage routes incoming messages.
func (h *Hub) handleMessage(client *Client, msg *protocol.Message) {
	handlers := NewHandlers(h)
	handlers.Handle(client, msg)
}

// clientsInGame returns a snapshot of the clients in a game, safe to iterate without the lock.
func (h *Hub) clientsInGame(gameID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	gameClients := h.gameClients[gameID]
	clients := make([]*Client, 0, len(gameClients))
	for client := range gameClients {
		clients = append(clients, client)
	}
	return clients
}

// gameMutex returns the lock guarding a game's persisted state.
// Hold it from loading the state until saving it; it is not reentrant, so only
// top-level entry points (message handlers, AI turns, event callbacks) take it.
func (h *Hub) gameMutex(gameID string) *sync.Mutex {
	h.mu.Lock()
	defer h.mu.Unlock()
	mu, ok := h.gameLocks[gameID]
	if !ok {
		mu = &sync.Mutex{}
		h.gameLocks[gameID] = mu
	}
	return mu
}

// lockGame locks a game's state and returns the unlock function.
func (h *Hub) lockGame(gameID string) func() {
	mu := h.gameMutex(gameID)
	mu.Lock()
	return mu.Unlock
}

// hasPendingEvent reports whether a game has an unacknowledged event of the given type.
func (h *Hub) hasPendingEvent(gameID, eventType string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, e := range h.pendingEvents {
		if e.GameID == gameID && e.Type == eventType && !e.completed {
			return true
		}
	}
	return false
}

// cachedGameMap returns the processed map for a game, if loaded.
func (h *Hub) cachedGameMap(gameID string) *maps.Map {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.gameMaps[gameID]
}

// cacheGameMap stores the processed map for a game.
func (h *Hub) cacheGameMap(gameID string, m *maps.Map) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.gameMaps[gameID] = m
}

// forgetGameMap drops a game's cached map (after the map changes or the game is deleted).
func (h *Hub) forgetGameMap(gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.gameMaps, gameID)
}

// notifyGamePlayers sends a message to all players in a game.
func (h *Hub) notifyGamePlayers(gameID string, msgType protocol.MessageType, payload interface{}) {
	clients := h.clientsInGame(gameID)

	log.Printf("Notifying %d clients in game %s with message type %s", len(clients), gameID, msgType)

	msg, err := protocol.NewMessage(msgType, payload)
	if err != nil {
		return
	}

	for _, client := range clients {
		client.Send(msg)
	}
}

// sendToPlayer sends a message to a specific player.
func (h *Hub) sendToPlayer(playerID string, msgType protocol.MessageType, payload interface{}) {
	h.mu.RLock()
	client := h.playerClients[playerID]
	h.mu.RUnlock()

	if client == nil {
		return
	}

	msg, err := protocol.NewMessage(msgType, payload)
	if err != nil {
		return
	}

	client.Send(msg)
}

// AddClientToGame adds a client to a game's client list.
func (h *Hub) AddClientToGame(client *Client, gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.GameID != "" && client.GameID != gameID {
		if previous, ok := h.gameClients[client.GameID]; ok {
			delete(previous, client)
		}
	}
	if h.gameClients[gameID] == nil {
		h.gameClients[gameID] = make(map[*Client]bool)
	}
	h.gameClients[gameID][client] = true
	client.GameID = gameID
}

// RemoveClientFromGame removes a client from a game.
func (h *Hub) RemoveClientFromGame(client *Client, gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.gameClients[gameID]; ok {
		delete(clients, client)
	}
	if client.GameID == gameID {
		client.GameID = ""
	}
}

// SetClientPlayer associates a client with a player.
func (h *Hub) SetClientPlayer(client *Client, playerID, name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client.PlayerID = playerID
	client.Name = name
	h.playerClients[playerID] = client
}

// isAuthenticated reports whether a client has a player identity, read under the hub lock.
func (h *Hub) isAuthenticated(client *Client) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return client.PlayerID != ""
}

// IsPlayerOnline checks if a player is currently connected.
func (h *Hub) IsPlayerOnline(playerID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.playerClients[playerID]
	return ok
}

// SyncTimeout is how long to wait for client acknowledgments before proceeding anyway.
const SyncTimeout = 30 * time.Second

// CreatePendingEvent creates a new event that requires client acknowledgment.
// requiredPlayers are the human player IDs that must acknowledge.
// onComplete is called when all acks are received or timeout occurs.
func (h *Hub) CreatePendingEvent(eventID, eventType, gameID string, requiredPlayers []string, onComplete func()) *PendingEvent {
	acks := make(map[string]bool)
	for _, pid := range requiredPlayers {
		acks[pid] = false
	}

	event := &PendingEvent{
		ID:           eventID,
		Type:         eventType,
		GameID:       gameID,
		RequiredAcks: acks,
		Timeout:      time.Now().Add(SyncTimeout),
		OnComplete:   onComplete,
	}

	h.mu.Lock()
	h.pendingEvents[eventID] = event
	h.mu.Unlock()

	// Start timeout goroutine
	go h.eventTimeoutWatcher(eventID)

	// If no human players need to ack, complete immediately
	if len(requiredPlayers) == 0 {
		h.completeEvent(event)
	}

	return event
}

// eventTimeoutWatcher waits for timeout and completes the event if not already done.
func (h *Hub) eventTimeoutWatcher(eventID string) {
	h.mu.RLock()
	event, exists := h.pendingEvents[eventID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	// Wait until timeout
	time.Sleep(time.Until(event.Timeout))

	h.mu.Lock()
	event, exists = h.pendingEvents[eventID]
	if exists && !event.completed {
		log.Printf("Event %s timed out, proceeding anyway", eventID)
		h.mu.Unlock()
		h.completeEvent(event)
	} else {
		h.mu.Unlock()
	}
}

// AcknowledgeEvent processes a client's acknowledgment of an event.
func (h *Hub) AcknowledgeEvent(eventID, playerID string) {
	h.mu.Lock()
	event, exists := h.pendingEvents[eventID]
	if !exists {
		h.mu.Unlock()
		log.Printf("Ack received for unknown event %s from player %s", eventID, playerID)
		return
	}

	if _, required := event.RequiredAcks[playerID]; required {
		event.RequiredAcks[playerID] = true
		log.Printf("Event %s: received ack from %s", eventID, playerID)
	}

	// Check if all acks received
	allAcked := true
	for _, acked := range event.RequiredAcks {
		if !acked {
			allAcked = false
			break
		}
	}
	h.mu.Unlock()

	if allAcked {
		h.completeEvent(event)
	}
}

// completeEvent marks an event as complete and calls its callback.
func (h *Hub) completeEvent(event *PendingEvent) {
	h.mu.Lock()
	if event.completed {
		h.mu.Unlock()
		return
	}
	event.completed = true
	delete(h.pendingEvents, event.ID)
	h.mu.Unlock()

	log.Printf("Event %s (%s) completed, proceeding", event.ID, event.Type)

	// Call the completion callback
	if event.OnComplete != nil {
		event.OnComplete()
	}
}

// GetHumanPlayersInGame returns the player IDs of connected human players in a game.
func (h *Hub) GetHumanPlayersInGame(gameID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var players []string
	if gameClients, ok := h.gameClients[gameID]; ok {
		for client := range gameClients {
			if client.PlayerID != "" {
				players = append(players, client.PlayerID)
			}
		}
	}
	return players
}

// Client represents a connected WebSocket client.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan *protocol.Message

	sendMu     sync.Mutex
	sendClosed bool

	// Tasks run one at a time on the client's own goroutine (see processInbox)
	inboxMu     sync.Mutex
	inbox       []func()
	inboxClosed bool
	inboxSignal chan struct{}

	// Identity fields. PlayerID and Name are set once at authentication; GameID is
	// only changed by tasks on this client's inbox (and under hub.mu).
	PlayerID string
	GameID   string
	Name     string
}

// maxInbox caps queued messages per client so a flooding client is dropped.
const maxInbox = 1000

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 25 * time.Second // Ping every 25s (well under most cloud timeouts)
	readTimeout    = 20 * time.Minute // Players may be idle for a while thinking about their turn
	maxMessageSize = 2 * 1024 * 1024 // 2MB to accommodate territory drawings
)

// NewClient creates a new client.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan *protocol.Message, 256),
		inboxSignal: make(chan struct{}, 1),
	}
}

// post queues a task on the client's goroutine. It never blocks; tasks posted after
// the inbox closes are dropped.
func (c *Client) post(task func()) {
	c.inboxMu.Lock()
	if c.inboxClosed {
		c.inboxMu.Unlock()
		return
	}
	if len(c.inbox) >= maxInbox {
		c.inboxMu.Unlock()
		log.Printf("Client %s inbox full, disconnecting", c.PlayerID)
		c.conn.Close(websocket.StatusPolicyViolation, "too many messages")
		return
	}
	c.inbox = append(c.inbox, task)
	c.inboxMu.Unlock()

	select {
	case c.inboxSignal <- struct{}{}:
	default:
	}
}

// closeInbox stops accepting tasks; already queued tasks still run.
func (c *Client) closeInbox() {
	c.inboxMu.Lock()
	c.inboxClosed = true
	c.inboxMu.Unlock()

	select {
	case c.inboxSignal <- struct{}{}:
	default:
	}
}

// processInbox runs queued tasks in order until the inbox is closed and drained.
func (c *Client) processInbox() {
	for {
		c.inboxMu.Lock()
		if len(c.inbox) == 0 {
			closed := c.inboxClosed
			c.inboxMu.Unlock()
			if closed {
				return
			}
			<-c.inboxSignal
			continue
		}
		task := c.inbox[0]
		c.inbox[0] = nil
		c.inbox = c.inbox[1:]
		c.inboxMu.Unlock()

		task()
	}
}

// Send queues a message to be sent to the client.
// Messages sent after the client disconnects are dropped.
func (c *Client) Send(msg *protocol.Message) {
	c.sendMu.Lock()
	if c.sendClosed {
		c.sendMu.Unlock()
		return
	}
	select {
	case c.send <- msg:
		c.sendMu.Unlock()
	default:
		c.sendMu.Unlock()
		// Channel full, client too slow - disconnect
		log.Printf("Client send channel full, disconnecting client")
		c.hub.Unregister(c)
	}
}

// closeSend closes the send channel exactly once.
func (c *Client) closeSend() {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if !c.sendClosed {
		c.sendClosed = true
		close(c.send)
	}
}

// ReadPump pumps messages from the WebSocket to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.closeInbox()
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	c.conn.SetReadLimit(maxMessageSize)

	for {
		// Use a read timeout to detect dead connections
		// This works alongside the ping/pong from WritePump
		ctx, cancel := context.WithTimeout(context.Background(), readTimeout)

		msgType, data, err := c.conn.Read(ctx)
		cancel()

		if err != nil {
			status := websocket.CloseStatus(err)
			// Only log unexpected errors (not normal closures)
			if status != websocket.StatusNormalClosure && status != websocket.StatusGoingAway {
				// Check if it's a timeout or network issue
				if ctx.Err() == context.DeadlineExceeded {
					log.Printf("WebSocket read timeout for player %s", c.PlayerID)
				} else {
					// EOF and other network errors are common, log at lower verbosity
					log.Printf("WebSocket closed for player %s: %v", c.PlayerID, err)
				}
			}
			break
		}

		// Only process text messages
		if msgType != websocket.MessageText {
			continue
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Invalid message from player %s: %v", c.PlayerID, err)
			continue
		}

		c.hub.dispatch(c, &msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case msg, ok := <-c.send:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)

			if !ok {
				// Channel closed, close connection
				c.conn.Close(websocket.StatusNormalClosure, "")
				cancel()
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				cancel()
				continue
			}

			err = c.conn.Write(ctx, websocket.MessageText, data)
			cancel()

			if err != nil {
				log.Printf("Write error for player %s: %v", c.PlayerID, err)
				return
			}

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := c.conn.Ping(ctx)
			cancel()

			if err != nil {
				log.Printf("Ping failed for player %s: %v", c.PlayerID, err)
				return
			}
		}
	}
}
