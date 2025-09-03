package store

import (
	"context"
	"database/sql"
	"log"
	"os"
	"todo-api-v1/api"

	"github.com/redis/go-redis/v9"
)

var DB *sql.DB // Global variable in store package

func InitDB() *sql.DB {
	dbSource := os.Getenv("DB_SOURCE")
	if dbSource == "" {
		log.Fatal("FATAL: DB_SOURCE environment variable is not set.")
	}
	var err error
	db, err := sql.Open("postgres", dbSource)
	if err != nil {
		log.Fatalf("FATAL: Could not connect to the database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("FATAL: Could not ping the database: %v", err)
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
		log.Fatalf("FATAL: Could not create tables: %v", err)
	}

	log.Println("Database connection successful and table created.")

	DB = db
	return db
}

func InitRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatalf("FATAL: count not load redis address")
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Could not parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("FATAL: Could not connect to Redis: %v for %s", err, redisURL)
	}
	log.Println("Redis connection successful.")

	return client
}

func GetUserTodos(ctx context.Context, userID interface{}) ([]api.Todo, error) {
	rows, err := DB.QueryContext(
		ctx,
		"SELECT id, task, completed FROM todos WHERE user_id = $1",
		userID,
	)
	if err != nil {
		return nil, err // Return raw error - handler will decide status code
	}
	defer rows.Close()

	var todos []api.Todo

	for rows.Next() {
		var t api.Todo
		err = rows.Scan(&t.ID, &t.Task, &t.Completed)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return todos, nil
}

func CreateUserTodo(ctx context.Context, userID interface{}, task string, completed bool) (int, error) {
	var newID int
	err := DB.QueryRowContext(ctx, `INSERT INTO todos (task, completed, user_id) VALUES ($1, $2, $3) RETURNING id`, task, completed, userID).Scan(&newID)
	if err != nil {
		return 0, err
	}
	return newID, nil
}

func GetUserTodo(ctx context.Context, userID interface{}, id int) (api.Todo, error) {
	var Todo api.Todo
	err := DB.QueryRowContext(ctx, "SELECT id, task, completed FROM todos WHERE id = $1 and user_id = $2", id, userID).Scan(&Todo.ID, &Todo.Task, &Todo.Completed)

	if err != nil {
		return api.Todo{}, err
	}
	return Todo, err
}

func DeleteUserTodo(ctx context.Context, todoID int, userID interface{}) (int64, error) {
	res, err := DB.ExecContext(ctx, "DELETE FROM todos WHERE id = $1 AND user_id = $2", todoID, userID)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func UpdateUserTodo(ctx context.Context, task string, completed bool, id int, userID interface{}) (int64, error) {
	res, err := DB.ExecContext(ctx, "UPDATE todos SET task = $1, completed = $2 WHERE id = $3 and user_id = $4", task, completed, id, userID)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil

}

func GetUserByUsername(ctx context.Context, username string) (*api.User, error) {
	var user api.User
	sqlStatement := "SELECT id, password_hash FROM users WHERE username = $1"
	err := DB.QueryRowContext(ctx, sqlStatement, username).Scan(&user.ID, &user.PasswordHash)

	if err != nil {
		return nil, err // Return raw error - handler decides if it's "not found" or "server error"
	}

	return &user, nil
}

func CreateUser(ctx context.Context, username, passwordHash string) (int, error) {
	var newUserID int
	sqlStatement := `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`

	err := DB.QueryRowContext(ctx, sqlStatement, username, passwordHash).Scan(&newUserID)
	if err != nil {
		return 0, err // Return raw error - handler decides if it's duplicate username or server error
	}

	return newUserID, nil
}
