package myserver

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SignUp handles user registration
func SignUp(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		templates.ExecuteTemplate(w, "signup.html", nil)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	nickname := r.FormValue("nickname")
	ageStr := r.FormValue("age")
	gender := r.FormValue("gender")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")

	if username == "" || email == "" || password == "" {
		errorPage(w, "Username, Email and Password are required", "signup.html")
		return
	}

	// Parse age if provided
	var age int
	if ageStr != "" {
		var err error
		age, err = strconv.Atoi(ageStr)
		if err != nil {
			errorPage(w, "Age must be a valid number", "signup.html")
			return
		}
	}

	//check email
	var existingEmail string
	err := db.QueryRow("SELECT email FROM users WHERE email = ?", email).Scan(&existingEmail)
	if err == nil {
		errorPage(w, "Email already in use", "signup.html")
		return
		// if any other err expt norows
	} else if err != sql.ErrNoRows {
		log.Println("Database error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	//check username
	var existingUserName string
	err = db.QueryRow("SELECT username FROM users WHERE username = ?", username).Scan(&existingUserName)
	if err == nil {
		errorPage(w, "UserName already in use", "signup.html")
		return
	} else if err != sql.ErrNoRows {
		log.Println("Database error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Failed to hash password:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`
		INSERT INTO users (
			username, email, password, 
			nickname, age, gender, 
			first_name, last_name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		username, email, hashedPassword,
		nickname, age, gender,
		firstName, lastName)

	if err != nil {
		log.Println("Failed to insert user:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

// SignIn handles user authentication
func SignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		templates.ExecuteTemplate(w, "signin.html", nil)
		return
	}

	usernameOrEmail := r.FormValue("username")
	password := r.FormValue("password")

	if usernameOrEmail == "" || password == "" {
		errorPage(w, "Username/Email and password are required", "signin.html")
		return
	}

	var userID int
	var hashedPassword string
	err := db.QueryRow("SELECT id, password FROM users WHERE username = ? OR email = ?", usernameOrEmail, usernameOrEmail).Scan(&userID, &hashedPassword)

	if err == sql.ErrNoRows {
		errorPage(w, "Invalid username/email or password", "signin.html")
		return
	} else if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		errorPage(w, "Invalid username or password", "signin.html")
		return
	}

	// delete old sessions
	_, err = db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("Failed to clear old sessions: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// create a session
	sessionUUID, err := uuid.NewV4()
	if err != nil {
		log.Printf("Failed to generate UUID: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sessionID := sessionUUID.String()

	// expiry date 24h
	expiry := time.Now().Add(24 * time.Hour)
	_, err = db.Exec("INSERT INTO sessions (session_id, user_id, expiry) VALUES (?, ?, ?)",
		sessionID, userID, expiry)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Expires:  expiry,
		Path:     "/",
		HttpOnly: true,
	})

	http.Redirect(w, r, "/homepage", http.StatusSeeOther)
}

// Logout handles user logout
func Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		_, err := db.Exec("DELETE FROM sessions WHERE session_id = ?", cookie.Value)
		if err != nil {
			log.Printf("Failed to delete session: %v", err)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
	}

	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

// HomePage handles post creation and display
func HomePage(w http.ResponseWriter, r *http.Request) {

	//check user session
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Validate session
	var userID int
	var username string
	var expiry time.Time

	err = db.QueryRow(`SELECT user_id, expiry FROM sessions WHERE session_id = ?`, cookie.Value).Scan(&userID, &expiry)
	if err != nil || time.Now().After(expiry) {
		if err == nil {
			db.Exec("DELETE FROM sessions WHERE session_id = ?", cookie.Value)
		}
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Get username for the authenticated user
	db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)

	// if user post
	if r.Method == http.MethodPost {
		//10 * 1024 * 1024
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		content := r.FormValue("content")
		if content == "" {
			errorPage(w, "Post content cannot be empty", "homepage.html")
			return
		}

		//start a transaction
		tx, err := db.Begin()
		if err != nil {
			log.Println("Failed to begin transaction:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		var imagePath string
		file, handler, err := r.FormFile("image")
		// fmt.Println("Filename:", handler.Filename)
		// fmt.Println("Size:", handler.Size)
		// fmt.Println("Header:", handler.Header)

		if err == nil {
			defer file.Close()
			//.jpg
			ext := filepath.Ext(handler.Filename)
			//1_1743462605.jpeg
			fileName := fmt.Sprintf("%d_%d%s", userID, time.Now().Unix(), ext)
			//uploads/1_1743462749.jpg
			imagePath = filepath.Join("uploads", fileName)
			//../uploads/1_1743462749.jpg
			// destPath := filepath.Join("..", imagePath)

			//create dest
			dst, err := os.Create(imagePath)
			if err != nil {
				log.Println("Failed to create file:", err)
				http.Error(w, "Failed to upload image", http.StatusInternalServerError)
				return
			}
			defer dst.Close()

			//copy data from file to dst
			if _, err = io.Copy(dst, file); err != nil {
				log.Println("Failed to save file:", err)
				http.Error(w, "Failed to upload image", http.StatusInternalServerError)
				return
			}
		}

		var result sql.Result
		if imagePath != "" {
			result, err = tx.Exec("INSERT INTO posts (user_id, content, image_path) VALUES (?, ?, ?)",
				userID, content, imagePath)
		} else {
			result, err = tx.Exec("INSERT INTO posts (user_id, content) VALUES (?, ?)",
				userID, content)
		}

		if err != nil {
			log.Println("Failed to create post:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		postID, err := result.LastInsertId()
		if err != nil {
			log.Println("Failed to get last insert ID:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		tags := r.Form["tags"]
		if len(tags) > 0 {
			for _, tagID := range tags {
				_, err = tx.Exec("INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?)", postID, tagID)
				if err != nil {
					log.Printf("Failed to insert tag %s for post %d: %v", tagID, postID, err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}
		}

		if err = tx.Commit(); err != nil {
			log.Println("Failed to commit transaction:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/homepage", http.StatusSeeOther)
		return
	}

	//if user use filter
	tags, err := getAllTags()
	if err != nil {
		log.Println("Failed to fetch tags:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tagFilter := r.URL.Query().Get("tag")

	posts, err := getPosts(tagFilter)
	if err != nil {
		log.Println("Failed to fetch posts:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Username  string
		Posts     []Post
		Tags      []Tag
		ActiveTag string
	}{
		Username:  username,
		Posts:     posts,
		Tags:      tags,
		ActiveTag: tagFilter,
	}

	templates.ExecuteTemplate(w, "homepage.html", data)
}

func AddComment(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	var userID int
	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_id = ?", cookie.Value).Scan(&userID)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		postID := r.FormValue("post_id")
		content := r.FormValue("content")

		if content == "" {
			http.Redirect(w, r, "/homepage", http.StatusSeeOther)
			return
		}

		_, err := db.Exec("INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)",
			postID, userID, content)
		if err != nil {
			log.Printf("Failed to add comment: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/homepage", http.StatusSeeOther)
}

func AddLike(w http.ResponseWriter, r *http.Request) {
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

	if r.Method == http.MethodPost {
		r.ParseForm()
		postID := r.FormValue("post_id")

		var exists bool
		err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM likes WHERE post_id = ? AND user_id = ?)", postID, userID).Scan(&exists)
		if err != nil {
			log.Printf("Failed to check like existence: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if exists {
			_, err = db.Exec("DELETE FROM likes WHERE post_id = ? AND user_id = ?", postID, userID)
		} else {
			_, err = db.Exec("INSERT INTO likes (post_id, user_id) VALUES (?, ?)", postID, userID)
		}

		if err != nil {
			log.Printf("Failed to toggle like: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var likeCount int
		err = db.QueryRow("SELECT COUNT(*) FROM likes WHERE post_id = ?", postID).Scan(&likeCount)
		if err != nil {
			log.Printf("Failed to get like count: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%d", likeCount)))
	}
}
