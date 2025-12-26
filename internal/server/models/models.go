// Package models defines data structures used by the GophKeeper server.
package models

import "time"

// User represents a GophKeeper user account.
type User struct {
	ID        string
	Username  string
	Password  string
	Email     string
	CreatedAt time.Time
}

// AuthRequest represents authentication request data.
type AuthRequest struct {
	Username string
	Password string
	Email    string
}

// AuthResponse represents authentication response data.
type AuthResponse struct {
	UserId string
	Token  string
}
