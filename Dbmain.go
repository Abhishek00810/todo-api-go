package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3" // The SQLite driver
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
		}

		return
	} else {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Put valid URL", http.StatusMethodNotAllowed)
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

	res, err := db.Exec("INSERT INTO todos (task, completed) VALUES (? , ?)", NewTodo.Task, NewTodo.Completed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	NewTodo.ID = int(id)
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(NewTodo)

}

func getTodo(w http.ResponseWriter, _ *http.Request, id int) {
	row, err := db.Query("SELECT * FROM todos where id = ?", id)

	if err != nil {
		http.Error(w, "Error fetching data", http.StatusInternalServerError)
	}

	var todo Todo
	for row.Next() {
		err = row.Scan(&todo.ID, &todo.Task, &todo.Completed)
		if err != nil {
			http.Error(w, "Error while scanning values", http.StatusInternalServerError)
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(todo)
	if err != nil {
		http.Error(w, "Values are not getting turned right", http.StatusInternalServerError)
	}

}

func DeleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	res, err := db.Exec("DELETE FROM todos where id =?", id)

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
	res, err := db.Exec("UPDATE todos SET task = ?, completed = ? WHERE id = ?", updateTodo.Task, updateTodo.Completed, id)

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

func setUpDB() {
	var err error
	db, err = sql.Open("sqlite3", "./todos.db")
	if err != nil {
		log.Fatalf("Fatal: Count not connect to the database: %v", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS todos(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task text NOT NULL,
			completed BOOLEAN NOT NULL
		)
	`

	_, err = db.Exec(createTableSQL)

	if err != nil {
		log.Fatalf("FATAL: Count not create table %v", err)
	}

	log.Println("Database initialized and table created successfully.")
}

func main() {
	var err error

	setUpDB()
	db, err = sql.Open("sqlite3", "./todos.db")
	if err != nil {
		log.Fatalf("Fatal: Count not connect to the database: %v", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS todos(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task text NOT NULL,
			completed BOOLEAN NOT NULL
		)
	`

	_, err = db.Exec(createTableSQL)

	if err != nil {
		log.Fatalf("FATAL: Count not create table %v", err)
	}

	log.Println("Database initialized and table created successfully.")

	http.HandleFunc("/todos/", todoHandler)

	fmt.Println("Server listening to port 8080")

	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		log.Fatalf("FATAL: Server failed to start: %v", err)
	}

}
