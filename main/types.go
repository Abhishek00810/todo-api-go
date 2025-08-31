package main

import (
	"database/sql"

	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
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
