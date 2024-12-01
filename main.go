package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Book struct{
	ID int `json:"id"`
	Name string `json:"name"`
	Author string `json:"author"`
	Year int `json:"year"`
}

var db *sql.DB
// var books []Book
// var nextId int=1


func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./books.db") // Connect to SQLite database
	if err != nil {
		log.Fatal("Failed to open the database:", err)
	}

	// Create the books table if it doesn't exist
	query := `CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		author TEXT NOT NULL,
		year INTEGER
	);`
	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

var jwtKey = []byte("secret_key") // Use a secret key to sign the JWT
var users = map[string]string{
	"user1": "password123", // Sample hardcoded user: username -> password
}
// UserClaims struct to store claims in the JWT
type UserClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// Create JWT token for authentication
func generateJWT(username string) (string, error) {
	claims := &UserClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(), // Token expires in 24 hours
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// Login handler to authenticate and return a JWT token
func login(w http.ResponseWriter, r *http.Request) {
	var credentials map[string]string
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	username, password := credentials["username"], credentials["password"]

	// Check if the username and password are correct
	if storedPassword, ok := users[username]; ok && storedPassword == password {
		// Generate JWT token
		token, err := generateJWT(username)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}
		// Send the token as a response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
		})
		return
	}

	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}
// Middleware to validate JWT token and protect routes
func isAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Missing authorization token", http.StatusUnauthorized)
			return
		}

		// Parse and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Pass the request to the next handler if token is valid
		next.ServeHTTP(w, r)
	})
}


// var catalogue map[string]string = make(map[string]string)

func addBook(w http.ResponseWriter, r *http.Request){
	var book Book

	err := json.NewDecoder(r.Body).Decode(&book)
	if err!=nil{
		http.Error(w, "Invalid input format", http.StatusBadRequest)
		return
	}

	// book.ID=nextId
	// nextId++
	// books=append(books,book)

	query := `INSERT INTO books (name, author, year) VALUES (?, ?, ?)`

	result, err2 := db.Exec(query, book.Name, book.Author, book.Year)
	if err2 != nil {
		http.Error(w, "Failed to add book", http.StatusInternalServerError)
		return
	}

	id,_ := result.LastInsertId()
	book.ID = int(id)

	w.Header().Set("Content-type","application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Book added successfully",
		"id": strconv.Itoa(book.ID),
	})
}


func getBooks(w http.ResponseWriter, r *http.Request){
	
	
	rows, err := db.Query("SELECT id, name, author, year FROM books")

	if err != nil {
		http.Error(w, "Failed to fetch books", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var books_array []Book
	var count int=0

	for rows.Next(){
		var book Book

		err2 := rows.Scan(&book.ID,&book.Name,&book.Author,&book.Year)
		if err2!=nil{
			http.Error(w,"Failed to parse books", http.StatusInternalServerError)
			return
		}
		books_array=append(books_array,book)
		count++
	}

	if count==0{
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string {"message":"No books to show! Please add some books first!"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books_array)
}

func getBookbyID(w http.ResponseWriter, r *http.Request){
	idStr := mux.Vars(r)["id"]
	id_int, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	// for _,book := range books{

	// 	if id_int==book.ID{
	// 		w.Header().Set("Content-Type", "application/json")
	// 		json.NewEncoder(w).Encode(book)
	// 		return
	// 	}
	// }
	// http.Error(w, "Book not found", http.StatusNotFound)


	var book Book
	err = db.QueryRow("SELECT id, name, author, year FROM books WHERE id = ?", id_int).Scan(&book.ID, &book.Name, &book.Author, &book.Year)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}


func updateBook(w http.ResponseWriter, r *http.Request){
	idStr := mux.Vars(r)["id"]
	id_int, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	var updatedBook Book
	err2 := json.NewDecoder(r.Body).Decode(&updatedBook)
	if err2!=nil{
		http.Error(w,"Invalid Input", http.StatusBadRequest)
		return
	}

	// for i,book := range books{
	// 	if book.ID==id_int{
	// 		books[i].Name = updatedBook.Name
	// 		books[i].Author = updatedBook.Author
	// 		books[i].Year = updatedBook.Year
	// 		w.Header().Set("Content-Type", "application/json")
	// 		json.NewEncoder(w).Encode(map[string]string{"message": "Book updated successfully"})
	// 		return
	// 	}
	// }

	// http.Error(w, "Book not found", http.StatusNotFound)


	query := `UPDATE books SET name = ?, author = ?, year = ? WHERE id = ?`
	_, err = db.Exec(query, updatedBook.Name, updatedBook.Author, updatedBook.Year, id_int)

	if err != nil {
		http.Error(w, "Failed to update book", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Book updated successfully"})

}

func deleteBook(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id_int, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	// for i, book := range books {
	// 	if book.ID == id {
	// 		books = append(books[:i], books[i+1:]...) // Remove book from slice
	// 		w.Header().Set("Content-Type", "application/json")
	// 		json.NewEncoder(w).Encode(map[string]string{"message": "Book deleted successfully"})
	// 		return
	// 	}
	// }

	// http.Error(w, "Book not found", http.StatusNotFound)
	query := `DELETE FROM books WHERE id = ?`
	result, err := db.Exec(query, id_int)
	if err != nil {
		http.Error(w, "Failed to delete book", http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected==0{
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Book doesn't exist"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Book deleted successfully"})
}


func main(){

	initDB()
	defer db.Close()


	router := mux.NewRouter()

	// Public route for login
	router.HandleFunc("/login", login).Methods("POST")

	// Protected routes for book management
	router.Handle("/books", isAuthenticated(http.HandlerFunc(getBooks))).Methods("GET")
	router.Handle("/books/{id}", isAuthenticated(http.HandlerFunc(getBookbyID))).Methods("GET")
	router.Handle("/books", isAuthenticated(http.HandlerFunc(addBook))).Methods("POST")
	router.Handle("/books/{id}", isAuthenticated(http.HandlerFunc(deleteBook))).Methods("DELETE")
	// router.Handle()
	fmt.Println("Starting the server at localhost :8000")

	log.Fatal(http.ListenAndServe(":8000",router))

}