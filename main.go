package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Restrict in production
	},
}

var jwtSecret = []byte("your-secret-key") // Replace with secure key

var (
	connectedClients = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chat_connected_clients",
		Help: "Number of currently connected clients",
	})
	messagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_messages_sent_total",
		Help: "Total number of messages sent",
	})
)

func init() {
	prometheus.MustRegister(connectedClients, messagesSent)
}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./chat.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        room_id TEXT,
        username TEXT,
        message TEXT,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    )`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	return db
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "token",
		Value: tokenString,
		Path:  "/",
	})
	w.Write([]byte("Login successful"))
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		roomID = "default"
	}
	claims := token.Claims.(jwt.MapClaims)
	username := claims["username"].(string)
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), roomID: roomID, username: username}
	client.hub.register <- client
	go client.writePump()
	go client.readPump()
}

func main() {
	db := initDB()
	defer db.Close()
	hub := NewHub(db)
	go hub.Run()
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Server starting on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
