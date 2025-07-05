package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub_NewHub(t *testing.T) {
	hub := NewHub(nil)
	assert.NotNil(t, hub.rooms)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
}

func TestHub_RegisterClient(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	client := &Client{roomID: "test", username: "testuser", send: make(chan []byte)}
	hub.register <- client
	time.Sleep(10 * time.Millisecond) // Allow goroutine to process
	assert.Contains(t, hub.rooms, "test")
	assert.Contains(t, hub.rooms["test"].clients, client)
}

func TestHub_UnregisterClient(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	client := &Client{roomID: "test", username: "testuser", send: make(chan []byte)}
	hub.register <- client
	time.Sleep(10 * time.Millisecond) // Allow registration
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond) // Allow unregistration
	// Room should be deleted since it has no clients
	assert.NotContains(t, hub.rooms, "test")
}
