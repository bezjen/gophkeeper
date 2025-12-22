package client

import (
	pb "github.com/bezjen/gophkeeper/pkg/proto"
)

type LoginPassword struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type BankCard struct {
	Number string `json:"number"`
	Holder string `json:"holder"`
	Expiry string `json:"expiry"`
}

type TextData struct {
	Text string `json:"text"`
}

type BinaryData struct {
	Data []byte `json:"data"`
	Size int64  `json:"size"`
}

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
