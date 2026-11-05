// Package client implements the Lords of Conquest game client.
package client

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"lords-of-conquest/internal/protocol"

	"github.com/coder/websocket"
)

// How often to ping the health endpoint to keep cloud instances alive
const healthPingInterval = 5 * time.Minute

// NetworkEvent is a received message, or a notification that the connection was lost.
type NetworkEvent struct {
	Message      *protocol.Message
	Disconnected bool
}

// connection is one WebSocket session; its pumps exit when done is closed.
type connection struct {
	ws        *websocket.Conn
	done      chan struct{}
	closeOnce sync.Once
}

func (cn *connection) close() {
	cn.closeOnce.Do(func() {
		close(cn.done)
		cn.ws.Close(websocket.StatusNormalClosure, "")
	})
}

// NetworkClient handles WebSocket communication with the server.
// Received messages are delivered on Events() for the game loop to process.
type NetworkClient struct {
	current  *connection
	sendChan chan *protocol.Message
	events   chan NetworkEvent
	mu       sync.Mutex
}

// NewNetworkClient creates a new network client.
func NewNetworkClient() *NetworkClient {
	return &NetworkClient{
		sendChan: make(chan *protocol.Message, 64),
		events:   make(chan NetworkEvent, 256),
	}
}

// Connect establishes a connection to the server, replacing any existing one.
func (c *NetworkClient) Connect(serverAddr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.current != nil {
		c.current.close()
		c.current = nil
	}

	log.Printf("Attempting to connect to: %s", serverAddr)

	// Determine WebSocket URL scheme based on address
	// Use wss:// for .onrender.com and other cloud hosts, ws:// for localhost
	var url, healthURL string
	var isCloud bool
	if strings.Contains(serverAddr, ".onrender.com") ||
		strings.Contains(serverAddr, ".herokuapp.com") ||
		strings.Contains(serverAddr, ".fly.dev") ||
		strings.HasPrefix(serverAddr, "wss://") {
		// Cloud deployment - use secure WebSocket without port
		host := strings.TrimPrefix(serverAddr, "wss://")
		// Remove any port if specified (cloud providers use standard 443)
		if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
			host = host[:colonIdx]
		}
		url = "wss://" + host + "/ws"
		healthURL = "https://" + host + "/health"
		isCloud = true
	} else {
		url = "ws://" + serverAddr + "/ws"
		healthURL = "http://" + serverAddr + "/health"
		isCloud = false
	}
	log.Printf("Full WebSocket URL: %s", url)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ws, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		CompressionMode:      websocket.CompressionContextTakeover,
		CompressionThreshold: 512,
	})
	if err != nil {
		log.Printf("WebSocket dial failed: %v", err)
		return err
	}

	log.Printf("WebSocket connection established")

	// Drop anything queued for the previous connection
	for drained := false; !drained; {
		select {
		case <-c.sendChan:
		default:
			drained = true
		}
	}

	cn := &connection{ws: ws, done: make(chan struct{})}
	c.current = cn

	go c.readPump(cn)
	go c.writePump(cn)

	// Start health pings for cloud deployments to prevent instance spin-down
	if isCloud {
		go healthPingLoop(healthURL, cn.done)
	}

	return nil
}

// Disconnect closes the connection without raising a disconnect event.
func (c *NetworkClient) Disconnect() {
	c.mu.Lock()
	cn := c.current
	c.current = nil
	c.mu.Unlock()

	if cn != nil {
		cn.close()
	}
}

// IsConnected returns true if connected to server.
func (c *NetworkClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current != nil
}

// Send queues a message to be sent to the server.
func (c *NetworkClient) Send(msg *protocol.Message) {
	select {
	case c.sendChan <- msg:
	default:
		log.Println("Send channel full, dropping message")
	}
}

// SendPayload creates and sends a message with the given type and payload.
func (c *NetworkClient) SendPayload(msgType protocol.MessageType, payload interface{}) error {
	msg, err := protocol.NewMessage(msgType, payload)
	if err != nil {
		return err
	}
	c.Send(msg)
	return nil
}

// Events returns the channel of received messages and disconnect notifications.
func (c *NetworkClient) Events() <-chan NetworkEvent {
	return c.events
}

// readPump reads messages from the WebSocket.
func (c *NetworkClient) readPump(cn *connection) {
	defer func() {
		cn.close()

		c.mu.Lock()
		wasCurrent := c.current == cn
		if wasCurrent {
			c.current = nil
		}
		c.mu.Unlock()

		if wasCurrent {
			c.events <- NetworkEvent{Disconnected: true}
		}
	}()

	cn.ws.SetReadLimit(2 * 1024 * 1024) // 2MB to accommodate territory drawings

	for {
		// Read with no timeout - rely on ping/pong to detect dead connections
		msgType, data, err := cn.ws.Read(context.Background())
		if err != nil {
			status := websocket.CloseStatus(err)
			if status != websocket.StatusNormalClosure && status != websocket.StatusGoingAway {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		// Only process text messages
		if msgType != websocket.MessageText {
			continue
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		select {
		case c.events <- NetworkEvent{Message: &msg}:
		case <-cn.done:
			return
		}
	}
}

// writePump writes messages to the WebSocket. A failed write closes the connection.
func (c *NetworkClient) writePump(cn *connection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cn.done:
			return

		case msg := <-c.sendChan:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				cancel()
				continue
			}

			err = cn.ws.Write(ctx, websocket.MessageText, data)
			cancel()

			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				cn.close()
				return
			}

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := cn.ws.Ping(ctx)
			cancel()

			if err != nil {
				cn.close()
				return
			}
		}
	}
}

// healthPingLoop periodically pings the server's health endpoint via HTTP.
// This keeps cloud instances (like Render free tier) from spinning down
// due to inactivity, since WebSocket ping/pong doesn't count as HTTP traffic.
func healthPingLoop(healthURL string, done <-chan struct{}) {
	ticker := time.NewTicker(healthPingInterval)
	defer ticker.Stop()

	// Do an initial ping right away
	pingHealth(healthURL)

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			pingHealth(healthURL)
		}
	}
}

// pingHealth makes an HTTP GET request to the health endpoint.
func pingHealth(healthURL string) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		log.Printf("Health ping failed: %v", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("Health ping OK")
	} else {
		log.Printf("Health ping returned status: %d", resp.StatusCode)
	}
}
