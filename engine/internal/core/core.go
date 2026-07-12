// Package core defines the shared domain types used across the engine, kept
// free of any persistence, networking or platform concerns so every other
// package can depend on it without coupling.
package core

import "time"

// DeviceID identifies a device. It is derived from the device's public key, so
// it is stable and unforgeable.
type DeviceID string

// Device is a known peer or the local device itself.
type Device struct {
	ID       DeviceID
	Name     string
	Trusted  bool
	LastSeen time.Time // zero value means "never seen"
	AddedAt  time.Time
}

// SyncDirection controls how a folder synchronizes with peers.
type SyncDirection string

const (
	// SendReceive is a normal two-way folder.
	SendReceive SyncDirection = "sendreceive"
	// SendOnly publishes local changes but never applies remote ones.
	SendOnly SyncDirection = "sendonly"
	// ReceiveOnly applies remote changes but never sends local ones.
	ReceiveOnly SyncDirection = "receiveonly"
)

// Valid reports whether d is a recognized sync direction.
func (d SyncDirection) Valid() bool {
	switch d {
	case SendReceive, SendOnly, ReceiveOnly:
		return true
	default:
		return false
	}
}

// Folder is a locally shared directory that is kept in sync with peers.
type Folder struct {
	ID        string
	Label     string
	Path      string
	Direction SyncDirection
	Paused    bool
	AddedAt   time.Time
}
