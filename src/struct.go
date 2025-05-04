package myserver

import (
	"database/sql"
	"html/template"
)

var (
	db        *sql.DB
	templates = template.Must(template.ParseFiles(
		"./templates/signin.html",
		"./templates/signup.html",
		"./templates/homepage.html",
	))
)

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