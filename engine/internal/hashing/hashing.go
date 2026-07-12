// Package hashing provides SHA-256 content identifiers used across the engine.
package hashing

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
)

const Size = sha256.Size

const (
	hexLen   = Size * 2
	shortLen = 12
)

var ErrInvalidHash = errors.New("hashing: invalid hash")

type Hash [Size]byte

func OfBytes(b []byte) Hash {
	return Hash(sha256.Sum256(b))
}

func OfString(s string) Hash {
	return Hash(sha256.Sum256([]byte(s)))
}

func OfReader(r io.Reader) (Hash, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return Hash{}, err
	}
	var out Hash
	h.Sum(out[:0])
	return out, nil
}

func OfFile(path string) (Hash, error) {
	f, err := os.Open(path)
	if err != nil {
		return Hash{}, err
	}
	defer f.Close()
	return OfReader(f)
}

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) Short() string {
	return hex.EncodeToString(h[:])[:shortLen]
}

func (h Hash) IsZero() bool {
	return h == Hash{}
}

func (h Hash) Equal(other Hash) bool {
	return h == other
}

func (h Hash) MarshalText() ([]byte, error) {
	out := make([]byte, hexLen)
	hex.Encode(out, h[:])
	return out, nil
}

func (h *Hash) UnmarshalText(text []byte) error {
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*h = parsed
	return nil
}

// Hasher computes a Hash incrementally and satisfies io.Writer.
type Hasher struct {
	h hash.Hash
}

func NewHasher() *Hasher {
	return &Hasher{h: sha256.New()}
}

func (w *Hasher) Write(p []byte) (int, error) {
	return w.h.Write(p)
}

func (w *Hasher) Sum() Hash {
	var out Hash
	w.h.Sum(out[:0])
	return out
}

func Parse(s string) (Hash, error) {
	if len(s) != hexLen {
		return Hash{}, ErrInvalidHash
	}
	var out Hash
	if _, err := hex.Decode(out[:], []byte(s)); err != nil {
		return Hash{}, ErrInvalidHash
	}
	return out, nil
}
