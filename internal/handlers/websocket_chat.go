package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// createUpgrader creates a WebSocket upgrader with proper origin checking
func createUpgrader(allowedOrigins []string, cfg *config.WebSocketConfig, logger utils.Logger) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Same-origin requests (no Origin header)
			if origin == "" {
				return true
			}

			// Check against allowed origins
			for _, allowed := range allowedOrigins {
				if allowed == "*" || allowed == origin {
					return true
				}
			}

			logger.Warn("WebSocket origin not allowed", "origin", origin, "allowed", allowedOrigins)
			return false
		},
	}
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type string      `json:"type"` // message, online_count, heartbeat, system
	Data interface{} `json:"data"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub             *ConnectionHub
	conn            *websocket.Conn
	send            chan []byte
	userID          uint
	username        string
	nickname        string
	avatar          string
	ipAddress       string     // Client IP address
	closeOnce       sync.Once  // Ensures connection is closed only once
	channelClosed   bool       // Track if send channel is closed
	lastMessageTime time.Time  // Last message timestamp for rate limiting
	messageCount    int        // Message count in current time window
	mu              sync.Mutex // Protects rate limiting fields and channelClosed
}

// close safely closes the WebSocket connection exactly once
func (c *Client) close() {
	c.closeOnce.Do(func() {
		c.conn.Close()
	})
}

// closeSendChannel safely closes the send channel, preventing panic from double-close
func (c *Client) closeSendChannel() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.channelClosed {
		close(c.send)
		c.channelClosed = true
	}
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
	config     *config.WebSocketConfig
}

var (
	globalHub *ConnectionHub
	hubOnce   sync.Once
)

// InitConnectionHub initializes the global connection hub
func InitConnectionHub(chatRepo *services.ChatRepository, userRepo *services.UserRepository, cfg *config.Config) {
	hubOnce.Do(func() {
		globalHub = &ConnectionHub{
			clients:    make(map[uint]*Client),
			broadcast:  make(chan []byte, cfg.WebSocket.BroadcastBufferSize),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			chatRepo:   chatRepo,
			userRepo:   userRepo,
			logger:     utils.GetLogger(),
			config:     &cfg.WebSocket,
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
			var oldClient *Client
			// Check if user already has a connection
			if existing, exists := h.clients[client.userID]; exists {
				oldClient = existing
				// Remove from map immediately to prevent broadcast attempting to send
				delete(h.clients, client.userID)
				h.logger.Info("Replacing old connection", "userID", client.userID)
			}
			// Add new client to map
			h.clients[client.userID] = client
			h.mu.Unlock()

			// Close old connection outside the lock (if exists)
			if oldClient != nil {
				oldClient.closeSendChannel() // 使用安全的关闭方法，防止panic
				oldClient.close()
				h.logger.Info("Old connection closed", "userID", client.userID)
			}

			h.logger.Info("Client connected", "userID", client.userID, "username", client.username)
			h.broadcastOnlineCount()

		case client := <-h.unregister:
			h.mu.Lock()
			var shouldBroadcast bool
			var onlineCount int

			// Only close channel if this client is still the current connection
			// Prevents closing already-closed channels when old connections disconnect
			if currentClient, exists := h.clients[client.userID]; exists && currentClient == client {
				delete(h.clients, client.userID)
				shouldBroadcast = true
				onlineCount = len(h.clients) // 在锁内读取准确人数
			}
			h.mu.Unlock()

			if shouldBroadcast {
				client.closeSendChannel() // 使用安全的关闭方法，防止panic
				h.logger.Info("Client disconnected", "userID", client.userID, "onlineCount", onlineCount)
				h.broadcastOnlineCountValue(onlineCount)
			}

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

	h.broadcastOnlineCountValue(count)
}

// broadcastOnlineCountValue sends a specific online count to all clients
func (h *ConnectionHub) broadcastOnlineCountValue(count int) {
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
		c.close()
	}()

	c.conn.SetReadLimit(int64(c.hub.config.MaxMessageSize))
	c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.hub.config.PongWait) * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.hub.config.PongWait) * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
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
			// Heartbeat - respond to client to acknowledge receipt
			// Don't save heartbeat to database
			heartbeatResp := WSMessage{
				Type: "heartbeat",
				Data: map[string]interface{}{"timestamp": time.Now().Unix()},
			}
			if respData, err := json.Marshal(heartbeatResp); err == nil {
				select {
				case c.send <- respData:
				default:
					c.hub.logger.Warn("Heartbeat response buffer full", "userID", c.userID)
				}
			}

		case "message":
			// Chat message - save to database and broadcast
			dataMap, ok := wsMsg.Data.(map[string]interface{})
			if !ok {
				c.hub.logger.Error("Invalid message data format", "userID", c.userID)
				continue
			}

			content, ok := dataMap["content"].(string)
			if !ok {
				c.hub.logger.Error("Invalid message content type", "userID", c.userID)
				continue
			}

			// Trim whitespace
			content = strings.TrimSpace(content)

			// Validate content is not empty after trimming
			if len(content) == 0 {
				c.hub.logger.Warn("Empty message after trim", "userID", c.userID)
				continue
			}

			// Validate message length (count characters, not bytes)
			messageLen := utf8.RuneCountInString(content)
			if messageLen > c.hub.config.MaxMessageLength {
				c.hub.logger.Warn("Message too long (characters)", "userID", c.userID, "length", messageLen, "max", c.hub.config.MaxMessageLength)
				continue
			}

			// Additional validation: check byte size to ensure it fits within MaxMessageSize
			// Note: MaxMessageSize (4096 bytes) includes JSON structure, MaxMessageLength (500 chars) is content only
			contentBytes := len(content)
			// Reserve 600 bytes for JSON structure overhead
			if contentBytes > c.hub.config.MaxMessageSize-600 {
				c.hub.logger.Warn("Message too long (bytes)", "userID", c.userID, "bytes", contentBytes, "max", c.hub.config.MaxMessageSize-600)
				continue
			}

			// Rate limiting: check messages per second
			c.mu.Lock()
			now := time.Now()
			if now.Sub(c.lastMessageTime) < time.Second {
				c.messageCount++
				if c.messageCount > c.hub.config.MaxMessagesPerSecond {
					c.mu.Unlock()
					c.hub.logger.Warn("Rate limit exceeded", "userID", c.userID, "count", c.messageCount)
					continue
				}
			} else {
				c.messageCount = 1
				c.lastMessageTime = now
			}
			c.mu.Unlock()

			// Save message to database
			message, err := c.hub.chatRepo.SendMessage(c.userID, c.username, c.nickname, c.avatar, content, c.ipAddress)
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

		default:
			// Unknown message type
			c.hub.logger.Warn("Unknown message type", "type", wsMsg.Type, "userID", c.userID)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(time.Duration(c.hub.config.PingPeriod) * time.Second)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.hub.config.WriteWait) * time.Second))
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
			c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.hub.config.WriteWait) * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// HandleWebSocket handles WebSocket connection requests
func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	// Check if WebSocket hub is initialized
	if globalHub == nil {
		h.logger.Error("WebSocket hub not initialized")
		utils.ErrorResponse(c, 500, "Chat service unavailable")
		return
	}

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

	// Get client IP address before upgrade
	clientIP := c.ClientIP()

	// Create upgrader with CORS origin checking
	upgrader := createUpgrader(h.config.CORS.AllowOrigins, &h.config.WebSocket, h.logger)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to WebSocket", "error", err.Error(), "userID", userID)
		return
	}

	// Create client
	client := &Client{
		hub:             globalHub,
		conn:            conn,
		send:            make(chan []byte, globalHub.config.ClientSendBufferSize),
		userID:          userID,
		username:        user.Username,
		nickname:        nickname,
		avatar:          avatar,
		ipAddress:       clientIP,
		lastMessageTime: time.Now(),
		messageCount:    0,
	}

	// Register client
	globalHub.register <- client

	// Start write pump in background
	go client.writePump()

	// Start read pump as main goroutine (blocks until connection closes)
	client.readPump()
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
