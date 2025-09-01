package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"todo-api-v1/api"
	"todo-api-v1/store"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	// 1. SETUP: Connect to a dedicated TEST database.
	dbSource := os.Getenv("TEST_DB_SOURCE")
	if dbSource == "" {
		log.Fatalf("FATAL: TEST_DB_SOURCE environment variable is not set")
	}

	log.Printf("Connecting to test database: %s", dbSource)

	var err error
	db, err = sql.Open("postgres", dbSource)
	if err != nil {
		log.Fatalf("FATAL: Could not connect to test database: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("FATAL: Could not ping test database: %v", err)
	}

	// 2. RUN TESTS: m.Run() executes all the other Test... functions in the file.

	//REDIS IMPLEMENTATION

	redisAddr := os.Getenv("REDIS_TEST_ADDR")
	if redisAddr == "" {
		log.Fatalf("FATAL: count not load redis address")
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if _, err = rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("FATAL: Could not connect to Redis: %v for %s", err, redisAddr)
	}
	log.Println("Redis connection successful.")

	// 3. TEARDOWN: Close the database connection after all tests are done.

	store.DB = db
	exitCode := m.Run()
	db.Close()
	rdb.Close()
	os.Exit(exitCode)
}

func clearTable() {
	// Create the table (if it doesn't exist)
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

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatalf("FATAL: Could not create test table: %v", err)
	}

	// Delete all rows from the table to ensure a clean slate
	db.Exec("DELETE FROM todos")
	// Reset the auto-incrementing ID counter
	db.Exec("ALTER SEQUENCE todos_id_seq RESTART WITH 1")

	db.Exec("DELETE FROM users")

	db.Exec("ALTER SEQUENCE users_id_seq RESTART WITH 1")
}

// setupTestData inserts sample data for tests that expect existing data
func setupTestData() {
	// Insert test todos

	// Step 1: Create a test user with id = 123
	_, err := db.Exec("INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)",
		123, "testuser", "fake-hash")
	if err != nil {
		log.Fatalf("FATAL: Could not insert test user: %v", err)
	}
	_, err = db.Exec("INSERT INTO todos (task, completed, user_id) VALUES ($1, $2, $3)", "Test Task 1", false, 123)
	if err != nil {
		log.Fatalf("FATAL: Could not insert test data: %v", err)
	}
	_, err = db.Exec("INSERT INTO todos (task, completed, user_id) VALUES ($1, $2, $3)", "Test Task 2", true, 123)
	if err != nil {
		log.Fatalf("FATAL: Could not insert test data: %v", err)
	}
}

func TestGetTodos(t *testing.T) {
	clearTable()
	setupTestData() // Add this line to insert test data

	var todos []api.Todo

	req := httptest.NewRequest(http.MethodGet, "/todos", nil)
	rr := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), userKey, 123)
	req = req.WithContext(ctx)

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
	clearTable()

	setupTestData()
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
			inputBody:          []byte(`{"task": "", "completed": false}`), // Fixed JSON syntax
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
			ctx := context.WithValue(req.Context(), userKey, 123)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			CreateTodo(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status : %d, got %d", tc.expectedStatusCode, rr.Code)
			}
			if rr.Code == http.StatusCreated {
				var todo api.Todo
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
	clearTable()
	setupTestData() // Add this line to insert test data

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

			ctx := context.WithValue(req.Context(), userKey, 123)
			req = req.WithContext(ctx)
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
	clearTable()
	setupTestData() // Add this line to insert test data

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
			ctx := context.WithValue(req.Context(), userKey, 123)
			req = req.WithContext(ctx)
			todoHandler(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status: %d got : %d", tc.expectedStatusCode, rr.Code)
			}

			if rr.Code == http.StatusNoContent {
				idStr := strings.TrimPrefix(tc.path, "/todos/")
				id, _ := strconv.Atoi(idStr)

				var todo api.Todo
				err := db.QueryRow("SELECT id FROM todos WHERE id = $1", id).Scan(&todo.ID)

				if err != sql.ErrNoRows {
					t.Errorf("expected todo with id %d to be deleted, but it still exists", id)
				}
			}
		})
	}
}

func TestUpdateTodo(t *testing.T) {
	clearTable()
	setupTestData() // Add this line to insert test data

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
			name:               "Error - Empty Task",
			path:               "/todos/2",
			inputBody:          []byte(`{"task": "", "completed": false}`),
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Error - Malformed JSON",
			path:               "/todos/2",
			inputBody:          []byte(`{"task": "bad json" "completed": false}`),
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Error - Not Found",
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
			ctx := context.WithValue(req.Context(), userKey, 123)
			req = req.WithContext(ctx)
			todoHandler(rr, req)

			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status code %d; got %d", tc.expectedStatusCode, rr.Code)
			}

			if rr.Code == http.StatusOK {
				idStr := strings.TrimPrefix(tc.path, "/todos/")
				id, _ := strconv.Atoi(idStr)

				var taskFromDB string
				var completedFromDB bool

				err := db.QueryRow("SELECT task, completed FROM todos WHERE id = $1", id).Scan(&taskFromDB, &completedFromDB)
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

func TestRegister(t *testing.T) {
	clearTable()
	setupTestData()

	testCases := []struct {
		name               string
		inputBody          []byte
		expectedStatusCode int
	}{
		{
			name:               "success - Created",
			inputBody:          []byte(`{"username": "abhishek", "password": "abhi@123"}`),
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "Failed: Bad Request",
			inputBody:          []byte(`{"username": "", "password": "a21332432"}`),
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(tc.inputBody))
			rr := httptest.NewRecorder()

			registerHandler(rr, req)
			if rr.Code != tc.expectedStatusCode {
				t.Errorf("expected status: %d got : %d", tc.expectedStatusCode, rr.Code)
			}

			// Verification: Check if the user is actually in the database
			var userCount int
			db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'abhishek'").Scan(&userCount)
			if userCount != 1 {
				t.Errorf("expected user to be created in DB, but count is %d", userCount)
			}
		})
	}

}

func TestLogin(t *testing.T) {
	clearTable()
	setupTestData()
	// Seed the database with a known user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	_, err := db.Exec("INSERT INTO users (username, password_hash) VALUES ($1, $2)", "theabhishek", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to seed user for login test: %v", err)
	}

	t.Run("Success - Login", func(t *testing.T) {
		body := []byte(`{"username": "theabhishek", "password": "password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		loginHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200 OK; got %d", rr.Code)
		}

		// Verification: Check if the response body contains a valid token
		var response map[string]string
		json.NewDecoder(rr.Body).Decode(&response)
		tokenString, ok := response["token"]
		if !ok {
			t.Fatal("response did not contain a token")
		}
		if len(tokenString) < 20 {
			t.Errorf("token seems too short: %s", tokenString)
		}
	})
	// You could add more sub-tests for wrong password, user not found, etc.
}
