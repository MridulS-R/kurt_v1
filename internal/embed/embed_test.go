package embed

import (
	"math"
	"testing"
)

func TestCosine_identical(t *testing.T) {
	a := []float32{1, 0, 0}
	if got := Cosine(a, a); math.Abs(float64(got)-1.0) > 1e-5 {
		t.Fatalf("identical vectors: want 1.0, got %f", got)
	}
}

func TestCosine_orthogonal(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}
	if got := Cosine(a, b); math.Abs(float64(got)) > 1e-5 {
		t.Fatalf("orthogonal vectors: want 0.0, got %f", got)
	}
}

func TestCosine_opposite(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{-1, 0}
	if got := Cosine(a, b); math.Abs(float64(got)+1.0) > 1e-5 {
		t.Fatalf("opposite vectors: want -1.0, got %f", got)
	}
}

func TestCosine_zero(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	if got := Cosine(a, b); got != 0 {
		t.Fatalf("zero vector: want 0, got %f", got)
	}
}

func TestCosine_differentLengths(t *testing.T) {
	a := []float32{1, 2, 3, 4}
	b := []float32{1, 2}
	// Should not panic, uses min length
	_ = Cosine(a, b)
}

func TestCosine_similarity(t *testing.T) {
	a := []float32{3, 4}
	b := []float32{3, 4}
	got := Cosine(a, b)
	if math.Abs(float64(got)-1.0) > 1e-5 {
		t.Fatalf("same direction: want ~1.0, got %f", got)
	}
}

func TestCosine_scaled(t *testing.T) {
	a := []float32{1, 1}
	b := []float32{2, 2}
	got := Cosine(a, b)
	if math.Abs(float64(got)-1.0) > 1e-5 {
		t.Fatalf("scaled: want 1.0, got %f", got)
	}
}
