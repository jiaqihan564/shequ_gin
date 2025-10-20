package handlers

import (
	"context"
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

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
	chatRepo    *services.ChatRepository
	userRepo    *services.UserRepository
	logger      utils.Logger
	upgrader    websocket.Upgrader
	clients     map[uint]*Client
	clientsLock sync.RWMutex
	broadcast   chan *models.ChatMessage
	register    chan *Client
	unregister  chan *Client
}

// Client WebSocket客户端
type Client struct {
	ID       uint
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Handler  *WebSocketHandler
}

// NewWebSocketHandler 创建WebSocket处理器
func NewWebSocketHandler(chatRepo *services.ChatRepository, userRepo *services.UserRepository) *WebSocketHandler {
	handler := &WebSocketHandler{
		chatRepo: chatRepo,
		userRepo: userRepo,
		logger:   utils.GetLogger(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 生产环境应该检查Origin
			},
		},
		clients:    make(map[uint]*Client),
		broadcast:  make(chan *models.ChatMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	// 启动Hub
	go handler.run()

	return handler
}

// run WebSocket Hub主循环
func (h *WebSocketHandler) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.clientsLock.Lock()
			h.clients[client.ID] = client
			h.clientsLock.Unlock()
			h.logger.Info("WebSocket客户端已连接",
				"userID", client.ID,
				"username", client.Username,
				"totalClients", len(h.clients))

			// 发送在线用户数更新
			h.broadcastOnlineCount()

		case client := <-h.unregister:
			h.clientsLock.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.clientsLock.Unlock()
			h.logger.Info("WebSocket客户端已断开",
				"userID", client.ID,
				"username", client.Username,
				"totalClients", len(h.clients))

			// 更新在线用户数
			h.broadcastOnlineCount()

		case message := <-h.broadcast:
			// 广播消息给所有在线客户端
			h.clientsLock.RLock()
			messageJSON, _ := json.Marshal(map[string]interface{}{
				"type": "message",
				"data": message,
			})
			for _, client := range h.clients {
				select {
				case client.Send <- messageJSON:
				default:
					// 发送失败，关闭连接
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.clientsLock.RUnlock()

		case <-ticker.C:
			// 定期清理断开的连接
			h.cleanupDeadConnections()
		}
	}
}

// broadcastOnlineCount 广播在线用户数
func (h *WebSocketHandler) broadcastOnlineCount() {
	h.clientsLock.RLock()
	count := len(h.clients)
	h.clientsLock.RUnlock()

	message := map[string]interface{}{
		"type": "online_count",
		"data": map[string]int{
			"count": count,
		},
	}

	messageJSON, _ := json.Marshal(message)

	h.clientsLock.RLock()
	for _, client := range h.clients {
		select {
		case client.Send <- messageJSON:
		default:
		}
	}
	h.clientsLock.RUnlock()
}

// cleanupDeadConnections 清理断开的连接
func (h *WebSocketHandler) cleanupDeadConnections() {
	h.clientsLock.Lock()
	defer h.clientsLock.Unlock()

	for userID, client := range h.clients {
		// 发送ping消息检测连接
		if err := client.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			h.logger.Debug("检测到断开的连接", "userID", userID)
			close(client.Send)
			delete(h.clients, userID)
		}
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 获取用户ID
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	// 获取用户信息
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := h.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		utils.ErrorResponse(c, 500, "获取用户信息失败")
		return
	}

	// 升级HTTP连接到WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket升级失败", "error", err.Error())
		return
	}

	// 创建客户端
	client := &Client{
		ID:       userID,
		Username: user.Username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Handler:  h,
	}

	// 注册客户端
	h.register <- client

	// 更新在线状态
	_ = h.chatRepo.UpdateOnlineUser(userID, user.Username)

	// 启动读写goroutine
	go client.writePump()
	go client.readPump()
}

// readPump 从WebSocket读取消息
func (c *Client) readPump() {
	defer func() {
		c.Handler.unregister <- c
		c.Conn.Close()
	}()

	// 设置读取超时和消息大小限制
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetReadLimit(4096) // 4KB消息大小限制
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageData, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Handler.logger.Error("WebSocket读取错误", "error", err.Error())
			}
			break
		}

		// 解析消息
		var req models.SendMessageRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			c.Handler.logger.Warn("解析消息失败", "error", err.Error())
			continue
		}

		// 获取用户扩展信息
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		profile, err := c.Handler.userRepo.GetUserProfile(ctx, c.ID)
		nickname := ""
		avatar := ""
		if err == nil && profile != nil {
			nickname = profile.Nickname
			avatar = profile.AvatarURL
		}
		cancel()

		// 发送消息到数据库
		ipAddress := c.Conn.RemoteAddr().String()
		message, err := c.Handler.chatRepo.SendMessage(
			c.ID, c.Username, nickname, avatar, req.Content, ipAddress)
		if err != nil {
			c.Handler.logger.Error("保存消息失败", "error", err.Error())
			continue
		}

		// 广播消息给所有客户端
		c.Handler.broadcast <- message

		// 重置读取超时
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}
}

// writePump 向WebSocket写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 写入消息
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Handler.logger.Error("WebSocket写入错误", "error", err.Error())
				return
			}

		case <-ticker.C:
			// 发送ping保持连接
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetOnlineUsersWS 获取在线用户列表（WebSocket）
func (h *WebSocketHandler) GetOnlineUsersWS(c *gin.Context) {
	h.clientsLock.RLock()
	defer h.clientsLock.RUnlock()

	users := make([]map[string]interface{}, 0, len(h.clients))
	for userID, client := range h.clients {
		users = append(users, map[string]interface{}{
			"user_id":  userID,
			"username": client.Username,
		})
	}

	utils.SuccessResponse(c, 200, "获取成功", gin.H{
		"count": len(h.clients),
		"users": users,
	})
}
