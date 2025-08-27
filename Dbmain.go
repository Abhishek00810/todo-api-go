package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq" // The SQLite driver
)

var db *sql.DB

type Todo struct {
	ID        int    `json:"id"`
	Task      string `json:"task"`
	Completed bool   `json:"completed"`
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
	rows, err := db.Query("SELECT id, task, completed FROM todos")

	if err != nil {
		http.Error(w, "Values are not getting fetched by the DB", http.StatusInternalServerError)
		return
	}

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
	err = db.QueryRow(`INSERT INTO todos (task, completed) VALUES ($1, $2) RETURNING id`, NewTodo.Task, NewTodo.Completed).Scan(&newID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	NewTodo.ID = int(newID)
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(NewTodo)

}

func getTodo(w http.ResponseWriter, r *http.Request, id int) {
	var t Todo

	// 1. Use QueryRow for a single result. We also specify the columns explicitly.
	// The .Scan() is chained directly onto the QueryRow call.
	err := db.QueryRow("SELECT id, task, completed FROM todos WHERE id = $1", id).Scan(&t.ID, &t.Task, &t.Completed)
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

	// 3. If there were no errors, we found the todo. Send the successful response.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	res, err := db.Exec("DELETE FROM todos where id = $1", id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func updateTodo(w http.ResponseWriter, r *http.Request, id int) {
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
	res, err := db.Exec("UPDATE todos SET task = $1, completed = $2 WHERE id = $3", updateTodo.Task, updateTodo.Completed, id)

	if err != nil {
		http.Error(w, "Not able to update the DB", http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
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
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS todos (
        id SERIAL PRIMARY KEY,
        task TEXT NOT NULL,
        completed BOOLEAN NOT NULL
    );`

	if _, err = db.Exec(createTableSQL); err != nil {
		return nil, err
	}

	log.Println("Database connection successful and table created.")
	return db, nil
}

func main() {
	var err error

	db, err = setUpDB()
	if err != nil {
		log.Fatalf("FATAL: Could not initialize database: %v", err)
	}

	defer db.Close()

	log.Println("Database initialized and table created successfully.")

	http.HandleFunc("/todos/", todoHandler)

	fmt.Println("Server listening to port 8080")

	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		log.Fatalf("FATAL: Server failed to start: %v", err)
	}

}
