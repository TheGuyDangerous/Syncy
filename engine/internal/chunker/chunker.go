// Package chunker splits streams into deterministic content-defined blocks.
package chunker

import (
	"bytes"
	"errors"
	"io"
	"math/bits"

	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

const (
	KiB = 1 << 10
	MiB = 1 << 20
)

type Config struct {
	Min int
	Avg int
	Max int
}

func DefaultConfig() Config {
	return Config{Min: 128 * KiB, Avg: 512 * KiB, Max: 2 * MiB}
}

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

type Chunk struct {
	Offset int64
	Length int
	Hash   hashing.Hash
}

type Chunker struct {
	cfg  Config
	mask uint64
}

func New(cfg Config) (*Chunker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	maskBits := bits.TrailingZeros(uint(cfg.Avg))
	return &Chunker{cfg: cfg, mask: (uint64(1) << maskBits) - 1}, nil
}

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

		filled = copy(buf, buf[b:filled])
		if filled == 0 && eof {
			return nil
		}
	}
}

func (c *Chunker) SplitBytes(b []byte) ([]Chunk, error) {
	var chunks []Chunk
	err := c.Split(bytes.NewReader(b), func(ch Chunk) error {
		chunks = append(chunks, ch)
		return nil
	})
	return chunks, err
}

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
