package protocol

import (
	"encoding/json"
	"fmt"
	"io"
)

const Version = "0.1.0"

type AgentStatus string

const (
	StatusPending AgentStatus = "pending"
	StatusActive  AgentStatus = "active"
	StatusRevoked AgentStatus = "revoked"
)

type RegisterRequest struct {
	Type     string `json:"type"`
	Code     string `json:"code"`
	UUID     string `json:"uuid"`
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
	Version  string `json:"version"`
}

type RegisterResponse struct {
	Type    string      `json:"type"`
	Status  AgentStatus `json:"status"`
	Name    string      `json:"name,omitempty"`
	Message string      `json:"message,omitempty"`
}

type PingMessage struct {
	Type string `json:"type"`
}

func WriteJSON(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

func ReadJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	return nil
}

func NewRegisterRequest(code, uuid, hostname, token string) RegisterRequest {
	return RegisterRequest{
		Type:     "register",
		Code:     code,
		UUID:     uuid,
		Hostname: hostname,
		Token:    token,
		Version:  Version,
	}
}

func NewPing() PingMessage {
	return PingMessage{Type: "ping"}
}
