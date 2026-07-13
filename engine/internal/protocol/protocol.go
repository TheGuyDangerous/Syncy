// Package protocol defines the DeltaSync Protocol (DSP): the message types and
// length-prefixed wire framing devices use to reconcile and transfer data.
package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/TheGuyDangerous/Syncy/engine/internal/chunker"
	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

type MessageType uint8

const (
	TypeHello MessageType = iota + 1
	TypeFolderSummary
	TypeIndexUpdate
	TypeBlockRequest
	TypeBlockData
	TypeAck
	TypePing
	TypePong
	TypeError
	TypeFriendRequest
	TypeFriendResponse
	TypeFolderListRequest
	TypeFolderListResponse
)

func (t MessageType) String() string {
	switch t {
	case TypeHello:
		return "hello"
	case TypeFolderSummary:
		return "folder-summary"
	case TypeIndexUpdate:
		return "index-update"
	case TypeBlockRequest:
		return "block-request"
	case TypeBlockData:
		return "block-data"
	case TypeAck:
		return "ack"
	case TypePing:
		return "ping"
	case TypePong:
		return "pong"
	case TypeError:
		return "error"
	case TypeFriendRequest:
		return "friend-request"
	case TypeFriendResponse:
		return "friend-response"
	case TypeFolderListRequest:
		return "folder-list-request"
	case TypeFolderListResponse:
		return "folder-list-response"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(t))
	}
}

const (
	headerSize   = 6
	MaxFrameSize = 16 << 20
)

var (
	ErrFrameTooLarge = errors.New("protocol: frame exceeds maximum size")
	ErrShortPayload  = errors.New("protocol: payload too short")
)

type Frame struct {
	Type    MessageType
	Payload []byte
}

func WriteFrame(w io.Writer, typ MessageType, payload []byte) error {
	if len(payload) > MaxFrameSize {
		return ErrFrameTooLarge
	}
	var hdr [headerSize]byte
	hdr[0] = byte(typ)
	binary.BigEndian.PutUint32(hdr[2:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func ReadFrame(r io.Reader) (Frame, error) {
	var hdr [headerSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return Frame{}, err
	}
	length := binary.BigEndian.Uint32(hdr[2:])
	if length > MaxFrameSize {
		return Frame{}, ErrFrameTooLarge
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Frame{}, err
	}
	return Frame{Type: MessageType(hdr[0]), Payload: payload}, nil
}

func WriteMessage(w io.Writer, typ MessageType, msg any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return WriteFrame(w, typ, payload)
}

func Decode(frame Frame, v any) error {
	return json.Unmarshal(frame.Payload, v)
}

type Hello struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Version    string `json:"version"`
	Protocol   int    `json:"protocol"`
}

type FolderSummary struct {
	FolderID string       `json:"folder_id"`
	Digest   hashing.Hash `json:"digest"`
	Files    int          `json:"files"`
}

type FileMeta struct {
	Path    string          `json:"path"`
	Size    int64           `json:"size"`
	ModUnix int64           `json:"mod_unix"`
	Mode    uint32          `json:"mode"`
	Hash    hashing.Hash    `json:"hash"`
	Deleted bool            `json:"deleted,omitempty"`
	Blocks  []chunker.Chunk `json:"blocks,omitempty"`
}

type IndexUpdate struct {
	FolderID string     `json:"folder_id"`
	Files    []FileMeta `json:"files"`
	Final    bool       `json:"final"`
}

type BlockRef struct {
	Offset int64        `json:"offset"`
	Length int          `json:"length"`
	Hash   hashing.Hash `json:"hash"`
}

type BlockRequest struct {
	FolderID string     `json:"folder_id"`
	Path     string     `json:"path"`
	Blocks   []BlockRef `json:"blocks"`
}

type Ack struct {
	Marker string `json:"marker"`
}

type Ping struct {
	Nonce uint64 `json:"nonce"`
}

type Pong struct {
	Nonce uint64 `json:"nonce"`
}

type ErrorMsg struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type FriendRequest struct {
	FromID    string   `json:"from_id"`
	FromName  string   `json:"from_name,omitempty"`
	Endpoints []string `json:"endpoints,omitempty"`
}

type FriendResponse struct {
	Accepted  bool     `json:"accepted"`
	Name      string   `json:"name,omitempty"`
	Endpoints []string `json:"endpoints,omitempty"`
}

type SharedFolder struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type FolderListResponse struct {
	Folders []SharedFolder `json:"folders"`
}

func WriteBlockData(w io.Writer, hash hashing.Hash, data []byte) error {
	payload := make([]byte, hashing.Size+len(data))
	copy(payload, hash[:])
	copy(payload[hashing.Size:], data)
	return WriteFrame(w, TypeBlockData, payload)
}

func ParseBlockData(payload []byte) (hashing.Hash, []byte, error) {
	if len(payload) < hashing.Size {
		return hashing.Hash{}, nil, ErrShortPayload
	}
	var h hashing.Hash
	copy(h[:], payload[:hashing.Size])
	return h, payload[hashing.Size:], nil
}
