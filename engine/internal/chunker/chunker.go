// Package chunker splits streams into content-defined chunks (blocks).
//
// Boundaries are chosen from the data itself using a rolling "gear" hash, so
// inserting or removing bytes only disturbs the chunks near the edit — the rest
// keep the same boundaries and therefore the same [hashing.Hash]. That property
// is what lets Syncy transfer only the blocks that actually changed instead of
// resending whole files.
//
// The chunking is fully deterministic and platform-independent: the same bytes
// always produce the same chunks, which is required for two devices to agree on
// block identities.
package chunker

import (
	"bytes"
	"errors"
	"io"
	"math/bits"

	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

// Size units for configuring the chunker.
const (
	KiB = 1 << 10
	MiB = 1 << 20
)

// Config controls how boundaries are selected.
//
// Avg must be a power of two; it sets the target average distance between
// boundaries. Min and Max clamp the resulting chunk sizes so a pathological
// input can neither produce many tiny chunks nor one enormous chunk.
type Config struct {
	Min int // minimum chunk size in bytes (>= 1)
	Avg int // target average chunk size in bytes (a power of two)
	Max int // maximum chunk size in bytes (>= Avg)
}

// DefaultConfig returns production-oriented defaults that balance delta
// granularity against per-block metadata overhead for large folders.
func DefaultConfig() Config {
	return Config{Min: 128 * KiB, Avg: 512 * KiB, Max: 2 * MiB}
}

// Validate reports whether the configuration is usable.
func (c Config) Validate() error {
	switch {
	case c.Min < 1:
		return errors.New("chunker: Min must be >= 1")
	case c.Avg < c.Min:
		return errors.New("chunker: Avg must be >= Min")
	case c.Max < c.Avg:
		return errors.New("chunker: Max must be >= Avg")
	case bits.OnesCount(uint(c.Avg)) != 1:
		return errors.New("chunker: Avg must be a power of two")
	default:
		return nil
	}
}

// Chunk describes one content-defined block within a stream.
type Chunk struct {
	Offset int64        // byte offset of the chunk within the stream
	Length int          // length of the chunk in bytes
	Hash   hashing.Hash // SHA-256 of the chunk's bytes
}

// Chunker splits streams according to a validated [Config].
type Chunker struct {
	cfg  Config
	mask uint64
}

// New returns a Chunker for the given configuration.
func New(cfg Config) (*Chunker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	// Avg is a power of two, so log2(Avg) bits in the mask make a boundary
	// occur on average once every Avg bytes.
	maskBits := bits.TrailingZeros(uint(cfg.Avg))
	return &Chunker{cfg: cfg, mask: (uint64(1) << maskBits) - 1}, nil
}

// Split reads r to completion and calls fn for each chunk, in order. It buffers
// at most Max bytes at a time, so memory use is independent of the stream size.
// If fn returns an error, Split stops and returns it.
func (c *Chunker) Split(r io.Reader, fn func(Chunk) error) error {
	buf := make([]byte, c.cfg.Max)
	var (
		offset int64
		filled int
		eof    bool
	)
	for {
		for filled < c.cfg.Max && !eof {
			n, err := r.Read(buf[filled:])
			filled += n
			switch {
			case errors.Is(err, io.EOF):
				eof = true
			case err != nil:
				return err
			}
		}
		if filled == 0 {
			return nil
		}

		b := c.boundary(buf[:filled])
		if err := fn(Chunk{
			Offset: offset,
			Length: b,
			Hash:   hashing.OfBytes(buf[:b]),
		}); err != nil {
			return err
		}
		offset += int64(b)

		// Move the unconsumed tail to the front and refill on the next pass.
		filled = copy(buf, buf[b:filled])
		if filled == 0 && eof {
			return nil
		}
	}
}

// SplitBytes returns all chunks of b. It is a convenience wrapper around Split.
func (c *Chunker) SplitBytes(b []byte) ([]Chunk, error) {
	var chunks []Chunk
	err := c.Split(bytes.NewReader(b), func(ch Chunk) error {
		chunks = append(chunks, ch)
		return nil
	})
	return chunks, err
}

// boundary returns the length of the next chunk at the start of data, honoring
// the Min/Max clamps and the content-defined cut point in between.
func (c *Chunker) boundary(data []byte) int {
	n := len(data)
	if n <= c.cfg.Min {
		return n
	}
	if n > c.cfg.Max {
		n = c.cfg.Max
	}
	var fp uint64
	for i := 0; i < n; i++ {
		fp = (fp << 1) + gear[data[i]]
		if i+1 >= c.cfg.Min && fp&c.mask == 0 {
			return i + 1
		}
	}
	return n
}

// gear is a deterministic table mapping each byte value to a 64-bit value, used
// by the rolling hash. It is generated with splitmix64 from a fixed seed so the
// boundaries are identical on every platform and build forever.
var gear [256]uint64

func init() {
	x := uint64(0x2545F4914F6CDD1D)
	for i := range gear {
		x += 0x9E3779B97F4A7C15
		z := x
		z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
		z = (z ^ (z >> 27)) * 0x94D049BB133111EB
		z ^= z >> 31
		gear[i] = z
	}
}
