package main

import (
	"database/sql"
	"log"
)

type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	db         *sql.DB
}

type Room struct {
	clients   map[*Client]bool
	broadcast chan []byte
}

func NewHub(db *sql.DB) *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		db:         db,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			roomID := client.roomID
			if _, ok := h.rooms[roomID]; !ok {
				h.rooms[roomID] = &Room{
					clients:   make(map[*Client]bool),
					broadcast: make(chan []byte),
				}
				go h.runRoom(roomID)
			}
			h.rooms[roomID].clients[client] = true
			connectedClients.Inc()
			log.Printf("Client %s joined room %s. Total clients: %d", client.username, roomID, len(h.rooms[roomID].clients))
		case client := <-h.unregister:
			if room, ok := h.rooms[client.roomID]; ok {
				if _, ok := room.clients[client]; ok {
					close(client.send)
					delete(room.clients, client)
					connectedClients.Dec()
					log.Printf("Client %s left room %s. Total clients: %d", client.username, client.roomID, len(room.clients))
					if len(room.clients) == 0 {
						delete(h.rooms, client.roomID)
					}
				}
			}
		}
	}
}

func (h *Hub) runRoom(roomID string) {
	room := h.rooms[roomID]
	for message := range room.broadcast {
		messagesSent.Inc()
		for client := range room.clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(room.clients, client)
				connectedClients.Dec()
			}
		}
	}
}

func (h *Hub) saveMessage(roomID, username, message string) error {
	_, err := h.db.Exec("INSERT INTO messages (room_id, username, message) VALUES (?, ?, ?)", roomID, username, message)
	return err
}

func (h *Hub) loadRecentMessages(roomID string) ([]string, error) {
	rows, err := h.db.Query("SELECT username, message FROM messages WHERE room_id = ? ORDER BY timestamp DESC LIMIT 50", roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []string
	for rows.Next() {
		var username, msg string
		if err := rows.Scan(&username, &msg); err != nil {
			return nil, err
		}
		messages = append([]string{username + ": " + msg}, messages...)
	}
	return messages, nil
}
