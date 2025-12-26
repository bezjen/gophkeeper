// Package models defines data structures used by the GophKeeper client.
package models

import (
	pb "github.com/bezjen/gophkeeper/api/gophkeeper/v1"
)

// LoginPassword represents login credential data.
type LoginPassword struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// BankCard represents bank card information.
type BankCard struct {
	Number string `json:"number"`
	Holder string `json:"holder"`
	Expiry string `json:"expiry"`
}

// TextData represents plain text data.
type TextData struct {
	Text string `json:"text"`
}

// BinaryData represents binary data with size information.
type BinaryData struct {
	Data []byte `json:"data"`
	Size int64  `json:"size"`
}

// DataItem represents a complete data record with metadata.
type DataItem struct {
	ID        string            `json:"id"`
	Type      pb.DataType       `json:"type"`
	Name      string            `json:"name"`
	Content   interface{}       `json:"content,omitempty"`
	Encrypted []byte            `json:"encrypted,omitempty"`
	Metadata  map[string]string `json:"metadata"`
	Version   int64             `json:"version"`
	UpdatedAt int64             `json:"updated_at"`
	Deleted   bool              `json:"deleted"`
}
