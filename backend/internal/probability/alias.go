// Package probability implements the Alias Method for O(1) weighted random sampling.
// Based on the Vose/Javascript alias method for efficient discrete distribution sampling.
package probability

import (
	"crypto/rand"
	"encoding/binary"
	"math"
)

// AliasTable is a precomputed lookup table for O(1) weighted random sampling.
// Build once with NewAliasTable, then call Next() repeatedly.
type AliasTable struct {
	prob  []float64
	alias []int
	rng   struct {
		buf [8]byte
	}
}

// NewAliasTable builds an Alias Table from a slice of non-negative weights.
// Weights do not need to sum to 1 — they are normalized internally.
// Returns nil if weights is empty or all zeros.
func NewAliasTable(weights []float64) *AliasTable {
	n := len(weights)
	if n == 0 {
		return nil
	}

	// Normalize weights
	total := 0.0
	for _, w := range weights {
		total += w
	}
	if total <= 0 {
		return nil
	}

	// Convert to scaled probabilities (n * p_i)
	prob := make([]float64, n)
	scaled := make([]float64, n)
	alias := make([]int, n)

	for i := 0; i < n; i++ {
		scaled[i] = float64(n) * (weights[i] / total)
		prob[i] = scaled[i]
		alias[i] = -1
	}

	// Two-worklist algorithm (Vose)
	var small, large []int
	for i := 0; i < n; i++ {
		if scaled[i] < 1.0 {
			small = append(small, i)
		} else {
			large = append(large, i)
		}
	}

	for len(small) > 0 && len(large) > 0 {
		l := small[len(small)-1]
		small = small[:len(small)-1]
		g := large[len(large)-1]
		large = large[:len(large)-1]

		prob[l] = scaled[l]
		alias[l] = g

		scaled[g] = scaled[g] - (1.0 - scaled[l])
		if scaled[g] < 1.0 {
			small = append(small, g)
		} else {
			large = append(large, g)
		}
	}

	for len(large) > 0 {
		g := large[len(large)-1]
		large = large[:len(large)-1]
		prob[g] = 1.0
	}
	for len(small) > 0 {
		l := small[len(small)-1]
		small = small[:len(small)-1]
		prob[l] = 1.0
	}

	return &AliasTable{prob: prob, alias: alias}
}

// Next returns the index of the next sampled item.
// Uses crypto/rand for cryptographic-grade randomness.
func (t *AliasTable) Next() int {
	if t == nil || len(t.prob) == 0 {
		return -1
	}

	// Generate a random 64-bit unsigned integer
	_, err := rand.Read(t.rng.buf[:])
	if err != nil {
		return 0 // fallback — should never happen on modern systems
	}
	r := binary.LittleEndian.Uint64(t.rng.buf[:])

	n := uint64(len(t.prob))
	i := int(r % n)          // uniform column selection
	p := t.prob[i]           // threshold for this column
	limit := float64(r>>32) / float64(math.MaxUint32>>32) // fraction in [0,1)

	if limit < p {
		return i
	}
	return t.alias[i]
}

// Len returns the number of items in the table.
func (t *AliasTable) Len() int {
	if t == nil {
		return 0
	}
	return len(t.prob)
}
