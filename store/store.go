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
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatalf("FATAL: count not load redis address")
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr, // The address from our docker-compose.yml
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("FATAL: Could not connect to Redis: %v for %s", err, redisAddr)
	}
	log.Println("Redis connection successful.")

	return rdb
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
