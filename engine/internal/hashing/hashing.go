// Package hashing provides SHA-256 content identifiers used throughout Syncy.
//
// A [Hash] is a fixed-size, comparable value that identifies a piece of content
// (a whole file or an individual block). Because it is content-addressed,
// identical content anywhere in the system shares the same Hash, which is what
// makes block-level deduplication and delta synchronization possible.
package hashing

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

// Size is the length in bytes of a SHA-256 hash.
const Size = sha256.Size

// hexLen is the length of a Hash rendered as a hex string.
const hexLen = Size * 2

// shortLen is the number of hex characters used by [Hash.Short].
const shortLen = 12

// ErrInvalidHash is returned by [Parse] when the input is not a valid
// hex-encoded SHA-256 digest.
var ErrInvalidHash = errors.New("hashing: invalid hash")

// Hash is a SHA-256 digest used as a content identifier.
type Hash [Size]byte

// OfBytes returns the SHA-256 hash of b.
func OfBytes(b []byte) Hash {
	return Hash(sha256.Sum256(b))
}

// OfString returns the SHA-256 hash of s.
func OfString(s string) Hash {
	return Hash(sha256.Sum256([]byte(s)))
}

// OfReader returns the SHA-256 hash of all bytes read from r. It streams the
// data so it does not need to hold the whole input in memory.
func OfReader(r io.Reader) (Hash, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return Hash{}, err
	}
	var out Hash
	h.Sum(out[:0])
	return out, nil
}

// OfFile returns the SHA-256 hash of the file's contents.
func OfFile(path string) (Hash, error) {
	f, err := os.Open(path)
	if err != nil {
		return Hash{}, err
	}
	defer f.Close()
	return OfReader(f)
}

// String returns the lowercase hex encoding of the hash.
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Short returns a shortened hex prefix, convenient for logs and UI. It is not
// unique and must never be used as an identifier.
func (h Hash) Short() string {
	return hex.EncodeToString(h[:])[:shortLen]
}

// IsZero reports whether h is the zero value (no content hashed).
func (h Hash) IsZero() bool {
	return h == Hash{}
}

// Equal reports whether h and other are the same digest.
func (h Hash) Equal(other Hash) bool {
	return h == other
}

// MarshalText encodes the hash as hex, so it stores cleanly in JSON and the
// metadata database.
func (h Hash) MarshalText() ([]byte, error) {
	out := make([]byte, hexLen)
	hex.Encode(out, h[:])
	return out, nil
}

// UnmarshalText decodes a hex-encoded hash produced by MarshalText.
func (h *Hash) UnmarshalText(text []byte) error {
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*h = parsed
	return nil
}

// Parse decodes a 64-character hex string into a Hash.
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
