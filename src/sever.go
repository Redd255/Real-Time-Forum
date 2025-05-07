package myserver

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// initialise database
func InitHandlers(database *sql.DB) {
	db = database
}

func FilterByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("id")
	// fmt.Println(r.URL.RawQuery)
	http.Redirect(w, r, "/homepage?tag="+tag, http.StatusSeeOther)
}

func getAllTags() ([]Tag, error) {
	rows, err := db.Query("SELECT id, name FROM tags ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}
func getPosts(tagFilter string) ([]Post, error) {
	var rows *sql.Rows
	var err error

	if tagFilter != "" {
		// DISTINCT avoid selection same post everytime if it have multp tags
		rows, err = db.Query(`
            SELECT DISTINCT posts.id, users.username, posts.content, posts.image_path,
            (SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) AS likesj
            FROM posts 
            JOIN users ON posts.user_id = users.id 
            JOIN post_tags ON posts.id = post_tags.post_id
            JOIN tags ON post_tags.tag_id = tags.id
            WHERE tags.id = ?
            ORDER BY posts.id DESC`, tagFilter)

	} else {
		rows, err = db.Query(`
            SELECT posts.id, users.username, posts.content, posts.image_path,
            (SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) AS likes
            FROM posts 
            JOIN users ON posts.user_id = users.id 
            ORDER BY posts.id DESC`)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		var imagePath sql.NullString
		if err := rows.Scan(&post.ID, &post.Username, &post.Content, &imagePath, &post.Likes); err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}
		if imagePath.Valid {
			post.ImagePath = imagePath.String
		}

		// Get tags for post
		tagRows, err := db.Query(`
			SELECT tags.name 
			FROM post_tags 
			JOIN tags ON post_tags.tag_id = tags.id 
			WHERE post_tags.post_id = ?`, post.ID)
		if err != nil {
			log.Printf("Failed to get tags for post %d: %v", post.ID, err)
			continue
		}
		defer tagRows.Close()

		for tagRows.Next() {
			var tagName string
			if err := tagRows.Scan(&tagName); err != nil {
				log.Printf("Error scanning tag: %v", err)
				continue
			}
			post.Tags = append(post.Tags, tagName)
		}

		// Get comments for post
		commentRows, err := db.Query(`
			SELECT comments.id, users.username, comments.content,
			(SELECT COUNT(*) FROM comment_likes WHERE comment_likes.comment_id = comments.id) AS likes
			FROM comments 
			JOIN users ON comments.user_id = users.id 
			WHERE comments.post_id = ? 
			ORDER BY comments.created_at ASC`, post.ID)
		if err != nil {
			log.Printf("Failed to get comments: %v", err)
			continue
		}
		defer commentRows.Close()

		for commentRows.Next() {
			var comment Comment
			if err := commentRows.Scan(&comment.ID, &comment.Username, &comment.Content, &comment.Likes); err != nil {
				log.Printf("Error scanning comment: %v", err)
				continue
			}
			post.Comments = append(post.Comments, comment)
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func GetAllConn(w http.ResponseWriter, currentUserID int) []Contact {
	// Get all other users with unread counts
	rows, err := db.Query(`
        SELECT u.id, u.username, 
               (SELECT COUNT(*) FROM messages m WHERE m.sender_id = u.id AND m.recipient_id = ? AND m.is_read = FALSE) as unread_count
        FROM users u
        WHERE u.id != ?
        ORDER BY u.username`, currentUserID, currentUserID)
	if err != nil {
		log.Printf("Failed to fetch users: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var contact Contact
		if err := rows.Scan(&contact.ID, &contact.Username, &contact.Unread); err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}
		// avatar
		initials := ""
		words := strings.Fields(contact.Username)
		for _, word := range words {
			if len(word) > 0 {
				initials += string(word[0])
			}
			if len(initials) >= 2 {
				break
			}
		}
		contact.Initials = strings.ToUpper(initials)
		contacts = append(contacts, contact)
	}
	return contacts
}
