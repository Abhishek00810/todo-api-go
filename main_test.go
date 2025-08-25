package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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

func TestCreateTodo(t *testing.T) {
	setupTestDB()

	defer db.Close()
	defer os.Remove("./todos_test.db")

	testCases := []struct {
		name               string
		inputBody          []byte
		expectedStatusCode int
		expectedTask       string
	}{
		{
			name:               "Success - Create Todo",
			inputBody:          []byte(`{"task": "New Task From Test", "completed": false}`),
			expectedStatusCode: http.StatusCreated,
			expectedTask:       "New Task From Test",
		},
		{
			name:               "Error - Empty task",
			inputBody:          []byte(`{"task: "", "completed":false}`),
			expectedStatusCode: http.StatusBadRequest,
			expectedTask:       "",
		},
		{
			name:               "Error - Malformed JSON",
			inputBody:          []byte(`{"task": "missing comma" "completed": false}`),
			expectedStatusCode: http.StatusBadRequest,
			expectedTask:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewReader(tc.inputBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			CreateTodo(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status : %d, got %d", tc.expectedStatusCode, rr.Code)
			}
			if rr.Code == http.StatusCreated {
				var todo Todo
				err := json.NewDecoder(rr.Body).Decode(&todo)
				if err != nil {
					t.Fatalf("could not decode response body: %v", err)
				}
				if todo.Task != tc.expectedTask {
					t.Errorf("Expected task : %s, got : %s", tc.expectedTask, todo.Task)
				}
				if todo.ID == 0 {
					t.Errorf("expected new todo to have a non-zero ID; got %d", todo.ID)
				}
			}
		})
	}

}

func TestGetTodo(t *testing.T) {
	setupTestDB()

	defer db.Close()
	defer os.Remove("todos_test.db")

	testCases := []struct {
		name               string
		path               string // The URL path we will request
		expectedStatusCode int
		expectedBody       string // A substring we expect in the response body
	}{
		{
			name:               "Success - Found",
			path:               "/todos/1",
			expectedStatusCode: http.StatusOK,
			expectedBody:       `"task":"Test Task 1"`, // We can just check for a part of the JSON
		},
		{
			name:               "Error - Not Found",
			path:               "/todos/99",
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       "Todo not found",
		},
		{
			name:               "Error - Invalid ID",
			path:               "/todos/abc",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "Invalid Todo ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			todoHandler(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status: %v, got: %v", tc.expectedStatusCode, rr.Code)
			}
			if !strings.Contains(rr.Body.String(), tc.expectedBody) {
				t.Errorf("expected body to contain '%s'; got '%s'", tc.expectedBody, rr.Body.String())
			}
		})
	}

}

func TestDeleteTodo(t *testing.T) {
	setupTestDB()

	defer db.Close()
	defer os.Remove("./todos_test.db")

	testCases := []struct {
		name               string
		path               string // The URL path to delete
		expectedStatusCode int
	}{
		{
			name:               "Success - Delete Todo",
			path:               "/todos/1",
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "Error - Not Found",
			path:               "/todos/99",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "Error - Invalid ID",
			path:               "/todos/abc",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
			rr := httptest.NewRecorder()

			todoHandler(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status: %d got : %d", tc.expectedStatusCode, rr.Code)
			}

			if rr.Code == http.StatusNoContent {
				idStr := strings.TrimPrefix(tc.path, "/todos/")
				id, _ := strconv.Atoi(idStr)

				var todo Todo
				err := db.QueryRow("SELECT id FROM todos WHERE id = ?", id).Scan(&todo.ID)

				if err != sql.ErrNoRows {
					t.Errorf("expected todo with id %d to be deleted, but it still exists", id)
				}
			}
		})
	}
}

func TestUpdateTodo(t *testing.T) {
	setupTestDB()
	defer db.Close()
	defer os.Remove("./todos_test.db")

	testCases := []struct {
		name               string
		path               string
		inputBody          []byte
		expectedStatusCode int
		expectedTask       string
		expectedCompleted  bool
	}{
		{
			name:               "Success - Update Todo",
			path:               "/todos/1",
			inputBody:          []byte(`{"task": "Updated Task 1", "completed": true}`),
			expectedStatusCode: http.StatusOK,
			expectedTask:       "Updated Task 1",
			expectedCompleted:  true,
		},
		{
			name:               "Error - Empty Task", // Renamed for clarity
			path:               "/todos/2",
			inputBody:          []byte(`{"task": "", "completed": false}`),
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Error - Malformed JSON", // Renamed for clarity
			path:               "/todos/2",
			inputBody:          []byte(`{"task": "bad json" "completed": false}`),
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Error - Not Found", // FIX: Renamed for clarity
			path:               "/todos/99",
			inputBody:          []byte(`{"task": "doesn't matter", "completed": false}`),
			expectedStatusCode: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, tc.path, bytes.NewReader(tc.inputBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			todoHandler(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status code %d; got %d", tc.expectedStatusCode, rr.Code)
			}

			if rr.Code == http.StatusOK {
				idStr := strings.TrimPrefix(tc.path, "/todos/")
				id, _ := strconv.Atoi(idStr)

				var taskFromDB string
				var completedFromDB bool

				// FIX: Use Fatalf for a more descriptive error if verification fails.
				err := db.QueryRow("SELECT task, completed FROM todos WHERE id = ?", id).Scan(&taskFromDB, &completedFromDB)
				if err != nil {
					t.Fatalf("Failed to re-fetch from DB for verification: %v", err)
				}

				if taskFromDB != tc.expectedTask {
					t.Errorf("expected task '%s'; got '%s'", tc.expectedTask, taskFromDB)
				}
				if completedFromDB != tc.expectedCompleted {
					t.Errorf("expected completed %t; got %t", tc.expectedCompleted, completedFromDB)
				}
			}
		})
	}
}
