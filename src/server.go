package server

import (
	"database/sql"
	"net/http"
	"text/template"
)

type User struct {
	ID       int
	Name     string
	Email    string
	Password string
}

type Post struct {
	ID      int
	Content string
	Author  string
}

var tmpl = template.Must(template.ParseGlob("templates/*.html"))
var db *sql.DB

func SendDB(database *sql.DB) {
	db = database
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := db.Query("SELECT content, author FROM posts")
		if err != nil {
			http.Error(w, "Error fetching posts", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var posts []Post
		for rows.Next() {
			var post Post
			if err := rows.Scan(&post.Content, &post.Author); err != nil {
				http.Error(w, "Error reading posts", http.StatusInternalServerError)
				return
			}
			posts = append(posts, post)
		}

		tmpl.ExecuteTemplate(w, "index.html", posts)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		content := r.FormValue("content")
		author := r.FormValue("author")

		_, err := db.Exec("INSERT INTO posts (content, author) VALUES (?, ?)", content, author)
		if err != nil {
			http.Error(w, "Failed to save post", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl.ExecuteTemplate(w, "register.html", nil)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		user := User{
			Name:     r.FormValue("name"),
			Email:    r.FormValue("email"),
			Password: r.FormValue("password"),
		}

		_, err := db.Exec("INSERT INTO users (name, email, password) VALUES (?, ?, ?)", user.Name, user.Email, user.Password)
		if err != nil {
			http.Error(w, "Error registering user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl.ExecuteTemplate(w, "login.html", nil)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		user := User{
			Name:     r.FormValue("name"),
			Password: r.FormValue("password"),
		}

		row := db.QueryRow(`SELECT id, name, email FROM users WHERE name = ? AND password = ?`,
			user.Name, user.Password)

		var id int
		var name, email string
		err := row.Scan(&id, &name, &email)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

	}
}
