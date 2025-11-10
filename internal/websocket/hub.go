package websocket

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	Send      chan []byte
	CreatorID int
}

type DonationAlert struct {
	TargetCreatorID   int    `json:"-"`
	DonorName         string `json:"donor_name"`
	AmountCents       int    `json:"amount_cents"`
	DonorMessage      string `json:"donor_message"`
	MediaType         string `json:"media_type"`
	MediaURL          string `json:"media_url"`
	MediaStartSeconds int    `json:"media_start_seconds"`
	MediaEndSeconds   int    `json:"media_end_seconds"`
}

type Hub struct {
	Clients        map[int]*Client
	Register       chan *Client
	Unregister     chan *Client
	BroadcastAlert chan DonationAlert
}

func NewHub() *Hub {
	return &Hub{
		Clients:        make(map[int]*Client),
		Register:       make(chan *Client),
		Unregister:     make(chan *Client),
		BroadcastAlert: make(chan DonationAlert),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.CreatorID] = client
			log.Printf("WebSocket Client registered for creator %d", client.CreatorID)

		case client := <-h.Unregister:
			if _, ok := h.Clients[client.CreatorID]; ok {
				delete(h.Clients, client.CreatorID)
				close(client.Send)
				log.Printf("WebSocket Client unregistered for creator %d", client.CreatorID)
			}

		case alert := <-h.BroadcastAlert:
			if client, ok := h.Clients[alert.TargetCreatorID]; ok {
				jsonData, err := json.Marshal(alert)

				if err != nil {
					log.Println("Failed to marshal donation alert:", err)
					continue
				}

				select {
				case client.Send <- jsonData:
					log.Printf("Sent alert to creator %d", client.CreatorID)
				default:
					close(client.Send)
					delete(h.Clients, client.CreatorID)
				}
			}
		}
	}
}
