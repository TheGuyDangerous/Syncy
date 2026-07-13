package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/TheGuyDangerous/Syncy/engine/internal/chunker"
	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	payload := []byte("some payload bytes")
	if err := WriteFrame(&buf, TypeHello, payload); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	frame, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if frame.Type != TypeHello {
		t.Errorf("type = %v, want %v", frame.Type, TypeHello)
	}
	if !bytes.Equal(frame.Payload, payload) {
		t.Errorf("payload = %q, want %q", frame.Payload, payload)
	}
}

func TestEmptyPayloadFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, TypePing, nil); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	frame, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if frame.Type != TypePing || len(frame.Payload) != 0 {
		t.Errorf("unexpected frame %+v", frame)
	}
}

func TestWriteFrameTooLarge(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, TypeIndexUpdate, make([]byte, MaxFrameSize+1)); !errors.Is(err, ErrFrameTooLarge) {
		t.Errorf("error = %v, want ErrFrameTooLarge", err)
	}
}

func TestReadFrameRejectsOversizeLength(t *testing.T) {
	var hdr [headerSize]byte
	hdr[0] = byte(TypeIndexUpdate)
	binary.BigEndian.PutUint32(hdr[2:], MaxFrameSize+1)
	if _, err := ReadFrame(bytes.NewReader(hdr[:])); !errors.Is(err, ErrFrameTooLarge) {
		t.Errorf("error = %v, want ErrFrameTooLarge", err)
	}
}

func TestReadFrameTruncatedPayload(t *testing.T) {
	var hdr [headerSize]byte
	hdr[0] = byte(TypeHello)
	binary.BigEndian.PutUint32(hdr[2:], 100)
	input := append(hdr[:], []byte("too short")...)
	if _, err := ReadFrame(bytes.NewReader(input)); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("error = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReadFrameEmpty(t *testing.T) {
	if _, err := ReadFrame(bytes.NewReader(nil)); !errors.Is(err, io.EOF) {
		t.Errorf("error = %v, want io.EOF", err)
	}
}

func TestMessageRoundTrip(t *testing.T) {
	h1 := hashing.OfString("block-1")
	h2 := hashing.OfString("file")
	cases := []struct {
		typ MessageType
		msg any
		out any
	}{
		{TypeHello, &Hello{DeviceID: "abc", DeviceName: "laptop", Version: "0.1.0", Protocol: 1}, &Hello{}},
		{TypeFolderSummary, &FolderSummary{FolderID: "photos", Digest: h2, Files: 42}, &FolderSummary{}},
		{TypeIndexUpdate, &IndexUpdate{
			FolderID: "photos",
			Files: []FileMeta{{
				Path: "a/b.jpg", Size: 1234, ModUnix: 1700000000, Mode: 0o644, Hash: h2,
				Blocks: []chunker.Chunk{{Offset: 0, Length: 1234, Hash: h1}},
			}},
			Final: true,
		}, &IndexUpdate{}},
		{TypeBlockRequest, &BlockRequest{FolderID: "photos", Path: "a/b.jpg", Blocks: []BlockRef{{Offset: 0, Length: 1234, Hash: h1}}}, &BlockRequest{}},
		{TypeAck, &Ack{Marker: "m-7"}, &Ack{}},
		{TypePing, &Ping{Nonce: 99}, &Ping{}},
		{TypePong, &Pong{Nonce: 99}, &Pong{}},
		{TypeError, &ErrorMsg{Code: "conflict", Message: "both sides changed"}, &ErrorMsg{}},
		{TypeFriendRequest, &FriendRequest{FromID: "abc", FromName: "laptop", Endpoints: []string{"192.168.1.4:22067", "203.0.113.9:22067"}}, &FriendRequest{}},
		{TypeFriendResponse, &FriendResponse{Accepted: true, Name: "desktop", Endpoints: []string{"192.168.1.5:22067"}}, &FriendResponse{}},
	}
	for _, tc := range cases {
		t.Run(tc.typ.String(), func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteMessage(&buf, tc.typ, tc.msg); err != nil {
				t.Fatalf("WriteMessage: %v", err)
			}
			frame, err := ReadFrame(&buf)
			if err != nil {
				t.Fatalf("ReadFrame: %v", err)
			}
			if frame.Type != tc.typ {
				t.Fatalf("type = %v, want %v", frame.Type, tc.typ)
			}
			if err := Decode(frame, tc.out); err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if !reflect.DeepEqual(tc.out, tc.msg) {
				t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", tc.out, tc.msg)
			}
		})
	}
}

func TestBlockDataRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	hash := hashing.OfString("payload")
	data := []byte("the actual block bytes")
	if err := WriteBlockData(&buf, hash, data); err != nil {
		t.Fatalf("WriteBlockData: %v", err)
	}
	frame, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if frame.Type != TypeBlockData {
		t.Fatalf("type = %v, want block-data", frame.Type)
	}
	gotHash, gotData, err := ParseBlockData(frame.Payload)
	if err != nil {
		t.Fatalf("ParseBlockData: %v", err)
	}
	if gotHash != hash {
		t.Error("hash mismatch")
	}
	if !bytes.Equal(gotData, data) {
		t.Errorf("data = %q, want %q", gotData, data)
	}
}

func TestParseBlockDataShort(t *testing.T) {
	if _, _, err := ParseBlockData([]byte{1, 2, 3}); !errors.Is(err, ErrShortPayload) {
		t.Errorf("error = %v, want ErrShortPayload", err)
	}
}

func TestMessageTypeString(t *testing.T) {
	if TypeBlockData.String() != "block-data" {
		t.Errorf("unexpected string: %q", TypeBlockData.String())
	}
	if MessageType(200).String() != "unknown(200)" {
		t.Errorf("unexpected string: %q", MessageType(200).String())
	}
}
