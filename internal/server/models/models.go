package models

import "time"

type User struct {
	ID        string
	Username  string
	Password  string
	Email     string
	CreatedAt time.Time
}

type AuthRequest struct {
	Username string
	Password string
	Email    string
}

type AuthResponse struct {
	UserId string
	Token  string
}
