package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"

	"my-platform/internal/models"
	ws "my-platform/internal/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	DB  *sqlx.DB
	Hub *ws.Hub
}

func NewWebSocketHandler(db *sqlx.DB, hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{DB: db, Hub: hub}
}

func (h *WebSocketHandler) ServerWs(c *gin.Context) {
	secretToken := c.Param("secretToken")

	var creator models.Creator
	query := `SELECT id FROM creators WHERE widget_secret_token = $1`
	err := h.DB.Get(&creator, query, secretToken)
	if err != nil {
		log.Println("Invalid WebSocket secret token:", secretToken)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade to connection:", err)
		return
	}

	client := &ws.Client{
		Hub:       h.Hub,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		CreatorID: creator.ID,
	}

	client.Hub.Register <- client

	go h.writePump(client)
	go h.readPump(client)
}

func (h *WebSocketHandler) writePump(client *ws.Client) {
	defer func() {
		client.Conn.Close()
	}()

	for message := range client.Send {
		if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}

	client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
}

func (h *WebSocketHandler) readPump(client *ws.Client) {
	defer func() {
		client.Hub.Unregister <- client
		client.Conn.Close()
	}()

	for {
		if _, _, err := client.Conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("readPump error: %v", err)
			}
			break
		}
	}
}
