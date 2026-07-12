package chunker

import (
	"io"
	"testing"

	"github.com/TheGuyDangerous/Syncy/engine/internal/hashing"
)

func pseudoRandom(n int, seed uint64) []byte {
	out := make([]byte, n)
	x := seed
	for i := range out {
		x += 0x9E3779B97F4A7C15
		z := x
		z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
		z = (z ^ (z >> 27)) * 0x94D049BB133111EB
		z ^= z >> 31
		out[i] = byte(z)
	}
	return out
}

func testChunker(t *testing.T) *Chunker {
	t.Helper()
	c, err := New(Config{Min: 2 * KiB, Avg: 8 * KiB, Max: 64 * KiB})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"default", DefaultConfig(), false},
		{"min zero", Config{Min: 0, Avg: 8, Max: 16}, true},
		{"avg below min", Config{Min: 16, Avg: 8, Max: 32}, true},
		{"max below avg", Config{Min: 2, Avg: 8, Max: 4}, true},
		{"avg not power of two", Config{Min: 2, Avg: 6, Max: 16}, true},
		{"valid small", Config{Min: 2, Avg: 8, Max: 16}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestChunksCoverInputContiguously(t *testing.T) {
	c := testChunker(t)
	data := pseudoRandom(1*MiB, 1)

	chunks, err := c.SplitBytes(data)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	var offset int64
	for i, ch := range chunks {
		if ch.Offset != offset {
			t.Fatalf("chunk %d offset = %d, want %d (gap or overlap)", i, ch.Offset, offset)
		}
		if ch.Length <= 0 {
			t.Fatalf("chunk %d has non-positive length %d", i, ch.Length)
		}
		want := hashing.OfBytes(data[offset : offset+int64(ch.Length)])
		if ch.Hash != want {
			t.Fatalf("chunk %d hash mismatch", i)
		}
		offset += int64(ch.Length)
	}
	if offset != int64(len(data)) {
		t.Fatalf("chunks cover %d bytes, want %d", offset, len(data))
	}
}

func TestChunkSizeBounds(t *testing.T) {
	c := testChunker(t)
	data := pseudoRandom(1*MiB, 2)

	chunks, err := c.SplitBytes(data)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}
	for i, ch := range chunks {
		if ch.Length > c.cfg.Max {
			t.Errorf("chunk %d length %d exceeds Max %d", i, ch.Length, c.cfg.Max)
		}
		// Every chunk except the last must be at least Min bytes.
		if i < len(chunks)-1 && ch.Length < c.cfg.Min {
			t.Errorf("non-final chunk %d length %d below Min %d", i, ch.Length, c.cfg.Min)
		}
	}
}

func TestDeterministic(t *testing.T) {
	c := testChunker(t)
	data := pseudoRandom(512*KiB, 3)

	first, err := c.SplitBytes(data)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}
	second, err := c.SplitBytes(data)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("chunk counts differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("chunk %d differs between runs", i)
		}
	}
}

func TestStreamingMatchesSplitBytes(t *testing.T) {
	c := testChunker(t)
	data := pseudoRandom(700*KiB, 4)

	viaBytes, err := c.SplitBytes(data)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}

	var viaStream []Chunk
	if err := c.Split(&drip{data: data, max: 7}, func(ch Chunk) error {
		viaStream = append(viaStream, ch)
		return nil
	}); err != nil {
		t.Fatalf("Split: %v", err)
	}

	if len(viaBytes) != len(viaStream) {
		t.Fatalf("chunk counts differ: bytes=%d stream=%d", len(viaBytes), len(viaStream))
	}
	for i := range viaBytes {
		if viaBytes[i] != viaStream[i] {
			t.Fatalf("chunk %d differs between SplitBytes and streaming", i)
		}
	}
}

type drip struct {
	data []byte
	pos  int
	max  int
}

func (d *drip) Read(p []byte) (int, error) {
	if d.pos >= len(d.data) {
		return 0, io.EOF
	}
	n := copy(p[:min(len(p), d.max)], d.data[d.pos:])
	d.pos += n
	return n, nil
}

func TestShiftResistance(t *testing.T) {
	c := testChunker(t)
	data := pseudoRandom(1*MiB, 5)

	original, err := c.SplitBytes(data)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}

	shifted, err := c.SplitBytes(append(pseudoRandom(100, 99), data...))
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}

	origHashes := make(map[hashing.Hash]struct{}, len(original))
	for _, ch := range original {
		origHashes[ch.Hash] = struct{}{}
	}
	shared := 0
	for _, ch := range shifted {
		if _, ok := origHashes[ch.Hash]; ok {
			shared++
		}
	}

	ratio := float64(shared) / float64(len(original))
	if ratio < 0.7 {
		t.Errorf("only %.0f%% of chunks were shared after a small prepend; "+
			"content-defined chunking should preserve most (shared=%d, total=%d)",
			ratio*100, shared, len(original))
	}
}

func TestEmptyInput(t *testing.T) {
	c := testChunker(t)
	chunks, err := c.SplitBytes(nil)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("empty input should yield 0 chunks, got %d", len(chunks))
	}
}

func TestSmallInputSingleChunk(t *testing.T) {
	c := testChunker(t)
	data := pseudoRandom(c.cfg.Min-1, 6) // smaller than Min
	chunks, err := c.SplitBytes(data)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("sub-Min input should yield exactly 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Length != len(data) {
		t.Errorf("single chunk length = %d, want %d", chunks[0].Length, len(data))
	}
	if chunks[0].Hash != hashing.OfBytes(data) {
		t.Error("single chunk hash mismatch")
	}
}

func TestForcedMaxBoundary(t *testing.T) {
	c := testChunker(t)
	data := make([]byte, 5*c.cfg.Max)
	chunks, err := c.SplitBytes(data)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}
	for i := 0; i < len(chunks)-1; i++ {
		if chunks[i].Length != c.cfg.Max {
			t.Errorf("chunk %d length = %d, want forced Max %d", i, chunks[i].Length, c.cfg.Max)
		}
	}
}

func BenchmarkSplit8MiB(b *testing.B) {
	c, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	data := pseudoRandom(8*MiB, 7)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.SplitBytes(data); err != nil {
			b.Fatal(err)
		}
	}
}
