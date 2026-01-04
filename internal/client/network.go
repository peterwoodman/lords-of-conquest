// Package client implements the Lords of Conquest game client.
package client

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"lords-of-conquest/internal/protocol"

	"github.com/coder/websocket"
)

// NetworkClient handles WebSocket communication with the server.
type NetworkClient struct {
	conn     *websocket.Conn
	sendChan chan *protocol.Message
	recvChan chan *protocol.Message
	done     chan struct{}
	mu       sync.Mutex

	// Callbacks
	OnMessage    func(*protocol.Message)
	OnConnect    func()
	OnDisconnect func(error)

	connected bool
}

// NewNetworkClient creates a new network client.
func NewNetworkClient() *NetworkClient {
	return &NetworkClient{
		sendChan: make(chan *protocol.Message, 64),
		recvChan: make(chan *protocol.Message, 64),
		done:     make(chan struct{}),
	}
}

// Connect establishes a connection to the server.
func (c *NetworkClient) Connect(serverAddr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	log.Printf("Attempting to connect to: %s", serverAddr)
	
	// Determine WebSocket URL scheme based on address
	// Use wss:// for .onrender.com and other cloud hosts, ws:// for localhost
	var url string
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
	} else {
		url = "ws://" + serverAddr + "/ws"
	}
	log.Printf("Full WebSocket URL: %s", url)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		log.Printf("WebSocket dial failed: %v", err)
		return err
	}

	log.Printf("WebSocket connection established")
	c.conn = conn
	c.connected = true
	c.done = make(chan struct{})

	go c.readPump()
	go c.writePump()

	if c.OnConnect != nil {
		c.OnConnect()
	}

	return nil
}

// Disconnect closes the connection.
func (c *NetworkClient) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	c.connected = false
	close(c.done)

	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "")
		c.conn = nil
	}
}

// IsConnected returns true if connected to server.
func (c *NetworkClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
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

// RecvChan returns the channel for received messages.
func (c *NetworkClient) RecvChan() <-chan *protocol.Message {
	return c.recvChan
}

// readPump reads messages from the WebSocket.
func (c *NetworkClient) readPump() {
	defer func() {
		c.mu.Lock()
		wasConnected := c.connected
		c.connected = false
		c.mu.Unlock()

		if wasConnected && c.OnDisconnect != nil {
			c.OnDisconnect(nil)
		}
	}()

	c.conn.SetReadLimit(65536)

	for {
		select {
		case <-c.done:
			return
		default:
		}

		// Read with no timeout - rely on ping/pong to detect dead connections
		ctx := context.Background()
		msgType, data, err := c.conn.Read(ctx)
		
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

		// Handle message via callback (recvChan is not used)
		if c.OnMessage != nil {
			c.OnMessage(&msg)
		}
	}
}

// writePump writes messages to the WebSocket.
func (c *NetworkClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return

		case msg := <-c.sendChan:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				cancel()
				continue
			}

			err = c.conn.Write(ctx, websocket.MessageText, data)
			cancel()
			
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := c.conn.Ping(ctx)
			cancel()
			
			if err != nil {
				return
			}
		}
	}
}
