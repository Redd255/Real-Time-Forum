package myserver

import (
	"database/sql"
	"html/template"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ----------DB--------------
var db *sql.DB

// ---------HOMEPAGE-----------
var templates = template.Must(template.ParseFiles(
	"./templates/signin.html",
	"./templates/signup.html",
	"./templates/homepage.html",
	"./templates/chat.html",
))

type Post struct {
	ID        int
	Username  string
	Content   string
	ImagePath string
	Likes     int
	Comments  []Comment
	Tags      []string
}

type Comment struct {
	ID       int
	Username string
	Content  string
	Likes    int
}

type Tag struct {
	ID   int
	Name string
}

// --------------- ws - chat -------------
type ClientManager struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.Mutex
}

type Client struct {
	id     int
	socket *websocket.Conn
	send   chan []byte
}

type Message struct {
	ID          int       `json:"id,omitempty"`
	SenderID    int       `json:"sender_id"`
	RecipientID int       `json:"recipient_id"`
	Username    string    `json:"username,omitempty"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	IsSent      bool      `json:"is_sent,omitempty"`
	Type        string    `json:"type"` 
}

type Contact struct {
	ID       int
	Username string
	Initials string
	Unread   int
}
