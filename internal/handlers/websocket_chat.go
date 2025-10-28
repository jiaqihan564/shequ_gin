package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	// WebSocket configuration
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 4096 // 4KB, support ~1000+ Chinese characters
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// TODO: In production, check origin properly
			return true
		},
	}
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type string      `json:"type"` // message, online_count, heartbeat, system
	Data interface{} `json:"data"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub      *ConnectionHub
	conn     *websocket.Conn
	send     chan []byte
	userID   uint
	username string
	nickname string
	avatar   string
}

// ConnectionHub manages all active WebSocket connections
type ConnectionHub struct {
	clients    map[uint]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	chatRepo   *services.ChatRepository
	userRepo   *services.UserRepository
	logger     utils.Logger
}

var (
	globalHub *ConnectionHub
	hubOnce   sync.Once
)

// InitConnectionHub initializes the global connection hub
func InitConnectionHub(chatRepo *services.ChatRepository, userRepo *services.UserRepository) {
	hubOnce.Do(func() {
		globalHub = &ConnectionHub{
			clients:    make(map[uint]*Client),
			broadcast:  make(chan []byte, 256),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			chatRepo:   chatRepo,
			userRepo:   userRepo,
			logger:     utils.GetLogger(),
		}
		go globalHub.run()
	})
}

// run starts the hub's main loop
func (h *ConnectionHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// If user already has a connection, close the old one
			if oldClient, exists := h.clients[client.userID]; exists {
				close(oldClient.send)
				delete(h.clients, client.userID)
				h.logger.Info("Replaced old connection", "userID", client.userID)
			}
			h.clients[client.userID] = client
			h.mu.Unlock()

			h.logger.Info("Client connected", "userID", client.userID, "username", client.username)
			h.broadcastOnlineCount()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, exists := h.clients[client.userID]; exists {
				delete(h.clients, client.userID)
				close(client.send)
			}
			h.mu.Unlock()

			h.logger.Info("Client disconnected", "userID", client.userID)
			h.broadcastOnlineCount()

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, skip
					h.logger.Warn("Client send buffer full", "userID", client.userID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// broadcastOnlineCount sends the current online count to all clients
func (h *ConnectionHub) broadcastOnlineCount() {
	h.mu.RLock()
	count := len(h.clients)
	h.mu.RUnlock()

	msg := WSMessage{
		Type: "online_count",
		Data: map[string]int{"count": count},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to marshal online count", "error", err.Error())
		return
	}

	h.broadcast <- data
}

// GetOnlineCount returns the current online count (O(1))
func (h *ConnectionHub) GetOnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetOnlineUsers returns the list of online users
func (h *ConnectionHub) GetOnlineUsers() []map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]map[string]interface{}, 0, len(h.clients))
	for _, client := range h.clients {
		users = append(users, map[string]interface{}{
			"user_id":  client.userID,
			"username": client.username,
			"nickname": client.nickname,
			"avatar":   client.avatar,
		})
	}
	return users
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Error("WebSocket read error", "error", err.Error(), "userID", c.userID)
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(messageBytes, &wsMsg); err != nil {
			c.hub.logger.Error("Failed to unmarshal message", "error", err.Error(), "userID", c.userID)
			continue
		}

		switch wsMsg.Type {
		case "heartbeat":
			// Heartbeat - do nothing, just reset read deadline
			// Don't save heartbeat to database

		case "message":
			// Chat message - save to database and broadcast
			dataMap, ok := wsMsg.Data.(map[string]interface{})
			if !ok {
				c.hub.logger.Error("Invalid message data format", "userID", c.userID)
				continue
			}

			content, ok := dataMap["content"].(string)
			if !ok || content == "" {
				c.hub.logger.Error("Empty message content", "userID", c.userID)
				continue
			}

			// Get IP address from connection (not available in WebSocket, use empty for now)
			ipAddress := ""

			// Save message to database
			message, err := c.hub.chatRepo.SendMessage(c.userID, c.username, c.nickname, c.avatar, content, ipAddress)
			if err != nil {
				c.hub.logger.Error("Failed to save message", "error", err.Error(), "userID", c.userID)
				continue
			}

			// Broadcast message to all clients
			broadcastMsg := WSMessage{
				Type: "message",
				Data: message,
			}

			data, err := json.Marshal(broadcastMsg)
			if err != nil {
				c.hub.logger.Error("Failed to marshal broadcast message", "error", err.Error())
				continue
			}

			c.hub.broadcast <- data
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

// HandleWebSocket handles WebSocket connection requests
func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	// User is already authenticated by AuthMiddleware
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Error("Failed to get user ID from context", "error", err.Error())
		utils.ErrorResponse(c, 401, "Unauthorized")
		return
	}

	// Get user info
	user, err := h.userRepo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user info", "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "Failed to get user info")
		return
	}

	// Get user profile
	profile, _ := h.userRepo.GetUserProfile(c.Request.Context(), userID)
	nickname := user.Username
	avatar := ""
	if profile != nil {
		if profile.Nickname != "" {
			nickname = profile.Nickname
		}
		avatar = profile.AvatarURL
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to WebSocket", "error", err.Error(), "userID", userID)
		return
	}

	// Create client
	client := &Client{
		hub:      globalHub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		username: user.Username,
		nickname: nickname,
		avatar:   avatar,
	}

	// Register client
	globalHub.register <- client

	// Start read and write pumps
	go client.writePump()
	go client.readPump()
}

// GetOnlineCountWS returns online count from WebSocket hub (HTTP fallback)
func (h *ChatHandler) GetOnlineCountWS(c *gin.Context) {
	if globalHub == nil {
		// Fallback to database method
		h.GetOnlineCount(c)
		return
	}

	count := globalHub.GetOnlineCount()
	utils.SuccessResponse(c, 200, "Success", models.OnlineCountResponse{
		Count: count,
	})
}

// GetOnlineUsersWS returns online users from WebSocket hub
func (h *ChatHandler) GetOnlineUsersWS(c *gin.Context) {
	if globalHub == nil {
		utils.ErrorResponse(c, 500, "WebSocket hub not initialized")
		return
	}

	users := globalHub.GetOnlineUsers()
	utils.SuccessResponse(c, 200, "Success", gin.H{
		"users": users,
		"count": len(users),
	})
}
