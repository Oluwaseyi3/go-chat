package main

import (
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	roomID   string
	username string
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()
	messages, err := c.hub.loadRecentMessages(c.roomID)
	if err != nil {
		log.Printf("Error loading recent messages: %v", err)
	} else {
		for _, msg := range messages {
			select {
			case c.send <- []byte(msg):
			default:
				log.Println("Failed to send recent message to client")
			}
		}
	}
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			return
		}
		msg := c.username + ": " + string(message)
		if err := c.hub.saveMessage(c.roomID, c.username, string(message)); err != nil {
			log.Printf("Error saving message: %v", err)
		}
		c.hub.rooms[c.roomID].broadcast <- []byte(msg)
	}
}

func (c *Client) writePump() {
	defer func() {
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()
	for message := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error writing message: %v", err)
			return
		}
	}
}
