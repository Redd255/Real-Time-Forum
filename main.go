package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type User struct {
	Name     string
	Email    string
	Password string
}

func serveForm(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("register.html"))
	tmpl.Execute(w, nil)
	fmt.Println("hh")
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	user := User{
		Name:     r.FormValue("name"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"), 
	}

	fmt.Printf("salam user: %+v\n", user)

	fmt.Fprintf(w, "Registration Successful! Welcome, %s.", user.Name)
}

func main() {
	http.HandleFunc("/", serveForm)
	http.HandleFunc("/register", handleRegister)
	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
