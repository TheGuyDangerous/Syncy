// Package core defines the shared domain types used across the engine.
package core

import "time"

type DeviceID string

type Device struct {
	ID       DeviceID  `json:"id"`
	Name     string    `json:"name"`
	Trusted  bool      `json:"trusted"`
	LastSeen time.Time `json:"last_seen"`
	AddedAt  time.Time `json:"added_at"`
}

type SyncDirection string

const (
	SendReceive SyncDirection = "sendreceive"
	SendOnly    SyncDirection = "sendonly"
	ReceiveOnly SyncDirection = "receiveonly"
)

func (d SyncDirection) Valid() bool {
	switch d {
	case SendReceive, SendOnly, ReceiveOnly:
		return true
	default:
		return false
	}
}

type Folder struct {
	ID        string        `json:"id"`
	Label     string        `json:"label"`
	Path      string        `json:"path"`
	Direction SyncDirection `json:"direction"`
	Paused    bool          `json:"paused"`
	AddedAt   time.Time     `json:"added_at"`
}

type QueuedOp struct {
	ID        int64
	DeviceID  DeviceID
	FolderID  string
	Kind      string
	Payload   string
	CreatedAt time.Time
	Attempts  int
}
