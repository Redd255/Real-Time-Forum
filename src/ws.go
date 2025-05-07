package myserver

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var manager = ClientManager{
	clients:    make(map[*Client]bool),
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
}

func (manager *ClientManager) getClientByUserID(userID int) *Client {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	for client := range manager.clients {
		if client.id == userID {
			return client
		}
	}
	return nil
}

func (manager *ClientManager) Start() {
	for {
		select {
		case client := <-manager.register:
			manager.mutex.Lock()
			manager.clients[client] = true
			manager.mutex.Unlock()
			log.Printf("Client connected: %d", client.id)

		case client := <-manager.unregister:
			manager.mutex.Lock()
			if _, ok := manager.clients[client]; ok {
				close(client.send)
				delete(manager.clients, client)
				log.Printf("Client disconnected: %d", client.id)
			}
			manager.mutex.Unlock()
		}
	}
}

func (manager *ClientManager) SendToClient(userID int, message []byte) {
	client := manager.getClientByUserID(userID)
	if client != nil {
		client.send <- message
	}
}


func (c *Client) Read() {
	defer func() {
		manager.unregister <- c
		c.socket.Close()
	}()

	for {
		_, message, err := c.socket.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}


		recipientID := msg.RecipientID
		senderID := c.id
		content := msg.Content

		if content == "" {
			continue
		}

		result, err := db.Exec("INSERT INTO messages (sender_id, recipient_id, content) VALUES (?, ?, ?)",
			senderID, recipientID, content)
		if err != nil {
			log.Printf("Failed to save message to DB: %v", err)
			continue
		}

		messageID, _ := result.LastInsertId()

		
		var username string
		err = db.QueryRow("SELECT username FROM users WHERE id = ?", senderID).Scan(&username)
		if err != nil {
			log.Printf("Failed to get username: %v", err)
		}

		responseMsg := Message{
			ID:          int(messageID),
			SenderID:    senderID,
			RecipientID: recipientID,
			Username:    username,
			Content:     content,
			CreatedAt:   time.Now(),
			IsSent:      true,
			Type:        "message",
		}

		responseMsgBytes, _ := json.Marshal(responseMsg)
		c.send <- responseMsgBytes

		responseMsg.IsSent = false
		recipientMsgBytes, _ := json.Marshal(responseMsg)
		manager.SendToClient(recipientID, recipientMsgBytes)
	}
}

func (c *Client) Write() {
	defer func() {
		c.socket.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.socket.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}
		}
	}
}

func InitWebsocket() {
	go manager.Start()
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_id = ?", cookie.Value).Scan(&userID)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %v", err)
		return
	}

	client := &Client{
		id:     userID,
		socket: conn,
		send:   make(chan []byte, 256),
	}

	manager.register <- client

	go client.Read()
	go client.Write()

	initMsg := Message{
		Type:    "connect",
		Content: "Connected to chat server",
	}
	msgBytes, _ := json.Marshal(initMsg)
	client.send <- msgBytes
}

func LoadChatHistory(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var currentUserID int
	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_id = ?", cookie.Value).Scan(&currentUserID)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	otherUserID := r.URL.Query().Get("user_id")
	if otherUserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Mark messages as read
	_, err = db.Exec("UPDATE messages SET is_read = TRUE WHERE sender_id = ? AND recipient_id = ?", otherUserID, currentUserID)
	if err != nil {
		log.Printf("Failed to mark messages as read: %v", err)
	}

	rows, err := db.Query(`
		SELECT m.id, m.sender_id, m.recipient_id, m.content, m.created_at, u.username 
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE (m.sender_id = ? AND m.recipient_id = ?) 
		   OR (m.sender_id = ? AND m.recipient_id = ?)
		ORDER BY m.created_at ASC`,
		currentUserID, otherUserID, otherUserID, currentUserID)
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var createdAt string
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.RecipientID, &msg.Content, &createdAt, &msg.Username); err != nil {
			log.Printf("Error scanning message: %v", err)
			continue
		}
		parsedTime, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			log.Println("Invalid timestamp format:", createdAt, "error:", err)
		} else {
			msg.CreatedAt = parsedTime
		}
		

		msg.IsSent = msg.SenderID == currentUserID
		msg.Type = "message"
		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func GetUnreadMessages(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_id = ?", cookie.Value).Scan(&userID)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := db.Query(`
		SELECT sender_id, COUNT(*) as count 
		FROM messages 
		WHERE recipient_id = ? AND is_read = FALSE 
		GROUP BY sender_id`, userID)
	if err != nil {
		log.Printf("Failed to get unread counts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UnreadInfo struct {
		SenderID int `json:"sender_id"`
		Count    int `json:"count"`
	}

	var unreadCounts []UnreadInfo
	for rows.Next() {
		var info UnreadInfo
		if err := rows.Scan(&info.SenderID, &info.Count); err != nil {
			log.Printf("Error scanning unread count: %v", err)
			continue
		}
		unreadCounts = append(unreadCounts, info)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"unread_counts": unreadCounts,
	})
}
