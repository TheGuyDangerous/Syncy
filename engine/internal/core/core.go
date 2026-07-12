// Package core defines the shared domain types used across the engine.
package core

import "time"

type DeviceID string

type Device struct {
	ID       DeviceID
	Name     string
	Trusted  bool
	LastSeen time.Time
	AddedAt  time.Time
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
	ID        string
	Label     string
	Path      string
	Direction SyncDirection
	Paused    bool
	AddedAt   time.Time
}
