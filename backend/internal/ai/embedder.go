package ai

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"unicode"
)

const stubDims = 64

// Embedder produces vector embeddings for text batches.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Model() string
	VectorDims() int
}

// StubEmbedder returns deterministic pseudo-embeddings from text hashes.
// Used when EMBED_API_KEY is empty and in unit tests (never calls an external API).
type StubEmbedder struct {
	Dims int
	// FailWith, if set, makes every Embed call return this error.
	FailWith error
}

// NewStubEmbedder returns a StubEmbedder with default dimensions.
func NewStubEmbedder() *StubEmbedder {
	return &StubEmbedder{Dims: stubDims}
}

// Embed implements Embedder.
func (s *StubEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.FailWith != nil {
		return nil, s.FailWith
	}
	dims := s.Dims
	if dims <= 0 {
		dims = stubDims
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = stubVector(t, dims)
	}
	return out, nil
}

// Model identifies this embedder for storage/versioning.
func (s *StubEmbedder) Model() string { return "stub/hash-v1" }

// VectorDims is the vector width produced by StubEmbedder.
func (s *StubEmbedder) VectorDims() int {
	if s.Dims <= 0 {
		return stubDims
	}
	return s.Dims
}

func stubVector(text string, dims int) []float32 {
	v := make([]float32, dims)
	add := func(key string, weight float64) {
		h := sha256.Sum256([]byte(key))
		for i := 0; i < 4; i++ {
			idx := int(binary.BigEndian.Uint32(h[i*4:(i+1)*4]) % uint32(dims))
			v[idx] += float32(weight)
		}
	}

	lower := strings.ToLower(strings.TrimSpace(text))
	var b strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			add("tok:"+b.String(), 1)
			b.Reset()
		}
	}
	if b.Len() > 0 {
		add("tok:"+b.String(), 1)
	}
	// Light full-text component so distinct sentences still differ.
	add("full:"+lower, 0.15)

	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	norm = math.Sqrt(norm)
	if norm < 1e-12 {
		return v
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / norm)
	}
	return v
}

// Ensure StubEmbedder satisfies Embedder.
var _ Embedder = (*StubEmbedder)(nil)

// ErrEmbedderUnavailable is returned when no API key is configured and stub is not used.
var ErrEmbedderUnavailable = fmt.Errorf("embedder unavailable: missing API key")
