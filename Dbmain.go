package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq" // The SQLite driver
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var rdb *redis.Client // Add this new global variable for the Redis client

var jwtKey []byte

type contextKey string

// We create a constant of our new type to use as the key.
const userKey contextKey = "userID"

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

type Todo struct {
	ID        int    `json:"id"`
	Task      string `json:"task"`
	Completed bool   `json:"completed"`
}

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func registerHandler(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)

	defer cancel()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&creds)

	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if creds.Password == "" || creds.Username == "" {
		http.Error(w, "Username and Passwrod are mandatory", http.StatusBadRequest)
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)

	if err != nil {
		http.Error(w, "Failed to encrypt the password", http.StatusInternalServerError)
		return
	}

	var newUserId int
	sqlStatement := `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING ID`

	err = db.QueryRowContext(ctx, sqlStatement, creds.Username, string(hashPassword)).Scan(&newUserId)

	if err != nil {
		// This could be a real DB error, or a "username already exists" error
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	fmt.Fprintf(w, "user created successfully with ID: %d", newUserId)

}

func todoHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")

	if idStr == "" {
		switch r.Method {
		case http.MethodGet:
			GetTodos(w, r)

		case http.MethodPost:
			CreateTodo(w, r)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		return
	} else {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid Todo ID", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			getTodo(w, r, id)
		case http.MethodPut:
			updateTodo(w, r, id)
		case http.MethodDelete:
			DeleteTodo(w, r, id)
		}
	}

}

func GetTodos(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userKey)
	if userID == nil {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)

	defer cancel()

	rows, err := db.QueryContext(
		ctx,
		"SELECT id, task, completed FROM todos WHERE user_id = $1",
		userID,
	)

	if err != nil {
		// If the timeout was exceeded, the error will be context.DeadlineExceeded
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else {
			log.Printf("ERROR: Database query failed: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}
	defer rows.Close()

	var todos []Todo

	for rows.Next() {
		var t Todo
		err = rows.Scan(&t.ID, &t.Task, &t.Completed)
		if err != nil {
			http.Error(w, "Error while scanning values", http.StatusInternalServerError)
		}
		todos = append(todos, t)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(todos)
	if err != nil {
		http.Error(w, "Values are not getting turned right", http.StatusInternalServerError)
	}
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userKey)
	if userID == nil {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	log.Println("UserKey has been there: ", userID)

	defer cancel()
	var NewTodo Todo
	err := json.NewDecoder(r.Body).Decode(&NewTodo)
	if err != nil {
		http.Error(w, "Values are not following rules", http.StatusBadRequest)
		return
	}

	if NewTodo.Task == "" {
		http.Error(w, "The 'task' field is required", http.StatusBadRequest)
		return
	}

	var newID int
	err = db.QueryRowContext(ctx, `INSERT INTO todos (task, completed, user_id) VALUES ($1, $2, $3) RETURNING id`, NewTodo.Task, NewTodo.Completed, userID).Scan(&newID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else {
			http.Error(w, "invalid query: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	NewTodo.ID = int(newID)
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(NewTodo)

}

func getTodo(w http.ResponseWriter, r *http.Request, id int) {
	userID := r.Context().Value(userKey)
	if userID == nil {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	var t Todo

	defer cancel()
	// 1. Use QueryRow for a single result. We also specify the columns explicitly.
	// The .Scan() is chained directly onto the QueryRow call.

	cacheKey := fmt.Sprintf("Todo: %d", id)

	val, err := rdb.Get(ctx, cacheKey).Result()

	if err == nil {
		log.Println("CACHE HIT for key:", cacheKey) // this is the most important line when it comes if the cache is available

		err := json.Unmarshal([]byte(val), &t)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(t)
			return
		}
	}

	//otherwise we will move into this piece of code

	log.Printf("Cache missing for %d", id)

	err = db.QueryRowContext(ctx, "SELECT id, task, completed FROM todos WHERE id = $1 and user_id = $2", id, userID).Scan(&t.ID, &t.Task, &t.Completed)

	//set the key as example: id:45
	if err != nil {
		// 2. This is the key part: Check if the error is specifically "no rows were found".
		if err == sql.ErrNoRows {
			// This is a client error (they asked for an ID that doesn't exist), so we send a 404.
			http.Error(w, "Todo not found", http.StatusNotFound)
		} else {
			// Any other error is a real server problem, so we send a 500.
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return // CRITICAL: Return after handling any error.
	}

	//BEFORE SENDING THE DATA WE WILL SAVE THIS INTO CACHE

	cacheData, err := json.Marshal(t)

	if err != nil {
		log.Printf("Not able to save the cache as marshalling failed")
	} else {
		err = rdb.Set(ctx, cacheKey, cacheData, 5*time.Minute).Err()
		if err != nil {
			log.Printf("ERROR: Failed to set the cache.")
		}
	}

	// 3. If there were no errors, we found the todo. Send the successful response.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	userID := r.Context().Value(userKey)
	if userID == nil {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)

	defer cancel()
	res, err := db.ExecContext(ctx, "DELETE FROM todos where id = $1 and user_id = $2", id, userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	cacheKey := fmt.Sprintf("Todo: %d", id)
	err = rdb.Del(ctx, cacheKey).Err()

	if err != nil {
		log.Printf("WARN: Failed to delete the cache key, %s,  %v", cacheKey, err)
	} else {
		log.Printf("successfully delete the cache key")
	}

	w.WriteHeader(http.StatusNoContent)
}

func updateTodo(w http.ResponseWriter, r *http.Request, id int) {
	userID := r.Context().Value(userKey)
	if userID == nil {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)

	defer cancel()
	var updateTodo Todo
	err := json.NewDecoder(r.Body).Decode(&updateTodo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if updateTodo.Task == "" {
		http.Error(w, "The 'task' field is required", http.StatusBadRequest)
		return
	}
	updateTodo.ID = id
	res, err := db.ExecContext(ctx, "UPDATE todos SET task = $1, completed = $2 WHERE id = $3 and user_id = $4", updateTodo.Task, updateTodo.Completed, id, userID)

	if err != nil {
		http.Error(w, "Not able to update the DB", http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	//after updating successfully and SENDING RESPONSE REQUEST WE WILL USE SETUP CACHE TO DELETE THE EXISTING K-V PAIR

	cacheKey := fmt.Sprintf("Todo: %d", id)
	err = rdb.Del(ctx, cacheKey).Err()

	if err != nil {
		log.Printf("WARN: Failed to delete the cache key, %s,  %v", cacheKey, err)
	} else {
		log.Printf("successfully delete the cache key")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateTodo)

}

func setUpDB() (*sql.DB, error) {
	dbSource := os.Getenv("DB_SOURCE")
	if dbSource == "" {
		log.Fatal("FATAL: DB_SOURCE environment variable is not set.")
	}

	var err error
	db, err = sql.Open("postgres", dbSource)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	// FIX: Using correct PostgreSQL syntax
	// In your initDB() function in main.go
	createTableSQL := `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS todos (
    id SERIAL PRIMARY KEY,
    task TEXT NOT NULL,
    completed BOOLEAN NOT NULL,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);`

	if _, err = db.Exec(createTableSQL); err != nil {
		return nil, err
	}

	log.Println("Database connection successful and table created.")

	//REDIS IMPLEMENTATION
	redisAddr := os.Getenv("REDIS_ADDR")

	if redisAddr == "" {
		log.Fatalf("FATAL: count not load redis address")
	}
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr, // The address from our docker-compose.yml
	})

	if _, err = rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("FATAL: Could not connect to Redis: %v for %s", err, redisAddr)
	}
	log.Println("Redis connection successful.")

	return db, nil
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)

	defer cancel()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	//fetch the user from the database

	var user User
	sqlStatement := "SELECT id, password_hash FROM users WHERE username = $1"
	err = db.QueryRowContext(ctx, sqlStatement, creds.Username).Scan(&user.ID, &user.PasswordHash)

	if err != nil {
		if err == sql.ErrNoRows {
			// If the user doesn't exist, return an unauthorized error
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password))

	if err != nil {
		http.Error(w, "Invalid username and password", http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(15 * time.Minute)
	claims := &Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)

	if err != nil {
		http.Error(w, "Failed to create token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})

}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHandler := r.Header.Get("Authorization")
		if authHandler == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHandler, "Bearer ")

		if tokenString == authHandler {
			//not valid token parsing
			http.Error(w, "Invalid token body", http.StatusUnauthorized)
			return
		}

		// now to validate
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))

	}
}

func main() {
	var err error

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET not set in environment")
	}
	jwtKey = []byte(secret)

	db, err = setUpDB()
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database: %v", err)
	}

	defer db.Close()

	log.Println("Database initialized and table created successfully.")

	http.HandleFunc("/todos/", authMiddleware(todoHandler))

	http.HandleFunc("/register", registerHandler)

	http.HandleFunc("/login", loginHandler)

	fmt.Println("Server listening to port 8080")

	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		log.Fatalf("FATAL: Server failed to start: %v", err)
	}

}

//Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjozLCJleHAiOjE3NTY2NDM3OTB9.p7UMgeZqP8uXBwXF6xamcum9cLBmvD-SDoE6Y-Nfiag
