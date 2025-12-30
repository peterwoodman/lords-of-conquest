// Package client implements the Lords of Conquest game client.
package client

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"lords-of-conquest/internal/protocol"

	"github.com/gorilla/websocket"
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
	url := "ws://" + serverAddr + "/ws"
	log.Printf("Full WebSocket URL: %s", url)
	
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
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
		c.conn.Close()
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

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Send to receive channel and callback
		select {
		case c.recvChan <- &msg:
		default:
			log.Println("Receive channel full, dropping message")
		}

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
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
