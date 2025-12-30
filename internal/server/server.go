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
	"lords-of-conquest/internal/protocol"

	"github.com/gorilla/websocket"
)

// Server is the main game server.
type Server struct {
	db       *database.DB
	hub      *Hub
	upgrader websocket.Upgrader
	addr     string
	server   *http.Server
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
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
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

	// Start the hub
	go s.hub.Run()

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
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := NewClient(s.hub, conn)
	s.hub.Register(client)

	// Start client goroutines
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

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	server *Server

	// Registered clients
	clients map[*Client]bool

	// Clients by player ID
	playerClients map[string]*Client

	// Clients in each game
	gameClients map[string]map[*Client]bool

	// Register requests
	register chan *Client

	// Unregister requests
	unregister chan *Client

	// Inbound messages from clients
	broadcast chan *ClientMessage

	mu sync.RWMutex
}

// ClientMessage wraps a message with its source client.
type ClientMessage struct {
	Client  *Client
	Message *protocol.Message
}

// NewHub creates a new Hub.
func NewHub(server *Server) *Hub {
	return &Hub{
		server:        server,
		clients:       make(map[*Client]bool),
		playerClients: make(map[string]*Client),
		gameClients:   make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan *ClientMessage, 256),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			// Send welcome message
			h.sendWelcome(client)

		case client := <-h.unregister:
			h.handleDisconnect(client)

		case msg := <-h.broadcast:
			// Handle messages in a goroutine to avoid blocking the hub
			go h.handleMessage(msg)
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message from a client.
func (h *Hub) Broadcast(client *Client, msg *protocol.Message) {
	h.broadcast <- &ClientMessage{Client: client, Message: msg}
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
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)

	// Remove from player mapping
	if client.PlayerID != "" {
		delete(h.playerClients, client.PlayerID)

		// Update connection status in all games
		if client.GameID != "" {
			h.server.db.SetPlayerConnected(client.GameID, client.PlayerID, false)

			// Notify other players in the game
			if gameClients, ok := h.gameClients[client.GameID]; ok {
				delete(gameClients, client)
				h.notifyGamePlayers(client.GameID, protocol.TypeDisconnect, protocol.DisconnectPayload{
					PlayerID: client.PlayerID,
					Reason:   "disconnected",
				})
			}
		}
	}

	close(client.send)
}

// handleMessage routes incoming messages.
func (h *Hub) handleMessage(cm *ClientMessage) {
	handlers := NewHandlers(h)
	handlers.Handle(cm.Client, cm.Message)
}

// notifyGamePlayers sends a message to all players in a game.
func (h *Hub) notifyGamePlayers(gameID string, msgType protocol.MessageType, payload interface{}) {
	h.mu.RLock()
	clients := h.gameClients[gameID]
	h.mu.RUnlock()

	msg, err := protocol.NewMessage(msgType, payload)
	if err != nil {
		return
	}

	for client := range clients {
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

// SetClientPlayer associates a client with a player ID.
func (h *Hub) SetClientPlayer(client *Client, playerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client.PlayerID = playerID
	h.playerClients[playerID] = client
}

// Client represents a connected WebSocket client.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan *protocol.Message

	PlayerID string
	GameID   string
	Name     string
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

// NewClient creates a new client.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan *protocol.Message, 256),
	}
}

// Send queues a message to be sent to the client.
func (c *Client) Send(msg *protocol.Message) {
	select {
	case c.send <- msg:
	default:
		// Channel full, client too slow
		c.hub.Unregister(c)
	}
}

// ReadPump pumps messages from the WebSocket to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Invalid message: %v", err)
			continue
		}

		c.hub.Broadcast(c, &msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
