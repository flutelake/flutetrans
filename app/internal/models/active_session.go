package models

import "context"

type ConnectionStatus string

const (
	StatusConnecting   ConnectionStatus = "connecting"
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusError        ConnectionStatus = "error"
)

type ActiveSession struct {
	SessionID    string
	ProfileID    string
	Protocol     ProtocolType
	Client       any
	Status       ConnectionStatus
	CurrentPath  string
	LastActivity int64
	CancelFunc   context.CancelFunc
}
