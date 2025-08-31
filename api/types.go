package api

import (
	"github.com/golang-jwt/jwt/v5"
)

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
