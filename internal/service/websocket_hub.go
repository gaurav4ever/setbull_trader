package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"setbull_trader/pkg/log"

	"github.com/gorilla/websocket"
)

// WebSocketHub manages WebSocket connections for real-time updates
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan []byte
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	log.WebSocketInfo("start", "WebSocket hub started", map[string]interface{}{
		"initial_clients": len(h.clients),
	})

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			clientCount := len(h.clients)
			h.mu.Unlock()

			log.WebSocketInfo("client_register", "Client registered", map[string]interface{}{
				"total_clients": clientCount,
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				clientCount := len(h.clients)
				h.mu.Unlock()

				log.WebSocketInfo("client_unregister", "Client unregistered", map[string]interface{}{
					"total_clients": clientCount,
				})
			} else {
				h.mu.Unlock()
				log.BBWWarn("websocket", "client_not_found", "Attempted to unregister non-existent client", map[string]interface{}{
					"total_clients": len(h.clients),
				})
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			clientCount := len(h.clients)
			successCount := 0
			failedCount := 0

			for client := range h.clients {
				select {
				case client.send <- message:
					successCount++
				default:
					close(client.send)
					delete(h.clients, client)
					failedCount++
				}
			}
			h.mu.RUnlock()

			log.WebSocketInfo("broadcast", "Message broadcasted", map[string]interface{}{
				"total_clients": clientCount,
				"success_count": successCount,
				"failed_count":  failedCount,
				"message_size":  len(message),
			})
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(message []byte) {
	log.WebSocketInfo("broadcast_request", "Broadcast request received", map[string]interface{}{
		"message_size": len(message),
	})
	h.broadcast <- message
}

// BroadcastJSON sends a JSON message to all connected clients
func (h *WebSocketHub) BroadcastJSON(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.WebSocketError("json_marshal", "Failed to marshal JSON for broadcast", err, map[string]interface{}{
			"data_type": fmt.Sprintf("%T", data),
		})
		return err
	}

	log.WebSocketInfo("broadcast_json", "JSON broadcast prepared", map[string]interface{}{
		"message_size": len(jsonData),
		"data_type":    fmt.Sprintf("%T", data),
	})

	h.Broadcast(jsonData)
	return nil
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.WebSocketInfo("connection_attempt", "WebSocket connection attempt", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
	})

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WebSocketError("upgrade_failed", "WebSocket upgrade failed", err, map[string]interface{}{
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		})
		return
	}

	client := &WebSocketClient{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	log.WebSocketInfo("connection_established", "WebSocket connection established", map[string]interface{}{
		"remote_addr":   r.RemoteAddr,
		"user_agent":    r.UserAgent(),
		"total_clients": h.GetClientCount(),
	})

	go client.writePump()
	go client.readPump()
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(60 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	log.WebSocketInfo("write_pump_start", "Write pump started", map[string]interface{}{
		"remote_addr": c.conn.RemoteAddr().String(),
	})

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				log.WebSocketInfo("write_pump_close", "Write pump closing - channel closed", map[string]interface{}{
					"remote_addr": c.conn.RemoteAddr().String(),
				})
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.WebSocketError("write_pump_error", "Failed to get next writer", err, map[string]interface{}{
					"remote_addr":  c.conn.RemoteAddr().String(),
					"message_size": len(message),
				})
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				log.WebSocketError("write_pump_close_error", "Failed to close writer", err, map[string]interface{}{
					"remote_addr": c.conn.RemoteAddr().String(),
				})
				return
			}

			log.WebSocketInfo("message_sent", "Message sent successfully", map[string]interface{}{
				"remote_addr":  c.conn.RemoteAddr().String(),
				"message_size": len(message),
			})

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.WebSocketError("ping_failed", "Failed to send ping", err, map[string]interface{}{
					"remote_addr": c.conn.RemoteAddr().String(),
				})
				return
			}

			log.WebSocketInfo("ping_sent", "Ping sent successfully", map[string]interface{}{
				"remote_addr": c.conn.RemoteAddr().String(),
			})
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		log.WebSocketInfo("pong_received", "Pong received", map[string]interface{}{
			"remote_addr": c.conn.RemoteAddr().String(),
		})
		return nil
	})

	log.WebSocketInfo("read_pump_start", "Read pump started", map[string]interface{}{
		"remote_addr": c.conn.RemoteAddr().String(),
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WebSocketError("read_pump_error", "Unexpected close error", err, map[string]interface{}{
					"remote_addr": c.conn.RemoteAddr().String(),
				})
			} else {
				log.WebSocketInfo("read_pump_close", "Read pump closing - normal close", map[string]interface{}{
					"remote_addr": c.conn.RemoteAddr().String(),
				})
			}
			break
		}

		log.WebSocketInfo("message_received", "Message received from client", map[string]interface{}{
			"remote_addr":  c.conn.RemoteAddr().String(),
			"message_size": len(message),
		})
	}
}
