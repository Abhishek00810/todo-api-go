package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupTestDB() {
	os.Remove("./todos_test.db")

	var err error
	db, err = sql.Open("sqlite3", "todos_test.db")

	if err != nil {
		log.Fatalf("Could not connect to testing database, : %v", err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS todos(id PRIMARY KEY, task TEXT NOT NULL, completed bool NOT NULL)`

	_, err = db.Exec(createTableSQL)

	if err != nil {
		log.Fatalf("FATAL: Could not create test table: %v", err)
	}

	_, err = db.Exec(`INSERT INTO todos (id, task, completed) VALUES (1, 'Test Task 1', false), (2, 'Test Task 2', true)`)
	if err != nil {
		log.Fatalf("FATAL: Could not insert the values, %v", err)
	}
}

func TestGetTodos(t *testing.T) {
	setupTestDB()
	defer db.Close()
	defer os.Remove("./todos_test.db")
	var todos []Todo

	req := httptest.NewRequest(http.MethodGet, "/todos", nil)
	rr := httptest.NewRecorder()

	GetTodos(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status: %d, got: %d", http.StatusOK, rr.Code)
	}

	err := json.NewDecoder(rr.Body).Decode(&todos)

	if err != nil {
		t.Fatalf("Value is not getting decoded, %v", err)
	}

	if len(todos) != 2 {
		t.Errorf("expected 2 todos; got %d", len(todos))
	}
}
