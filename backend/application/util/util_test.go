package util

import (
	"math"
	"testing"
)

// Test CumulativeToDiffs function
func TestCumulativeToDiffs(t *testing.T) {
	tests := []struct {
		name     string
		input    []int64
		expected []int64
	}{
		{"empty slice", []int64{}, nil},
		{"single element", []int64{5}, []int64{5}},
		{"multiple elements", []int64{5, 10, 15, 20}, []int64{5, 5, 5, 5}},
		{"non linear", []int64{3, 6, 10, 15}, []int64{3, 3, 4, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CumulativeToDiffs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Index %d: expected %d, got %d", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// Helper to compare float64 with tolerance
func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

// Test StandardDeviation function
func TestStandardDeviation(t *testing.T) {
	tests := []struct {
		name     string
		input    []int64
		expected float64
	}{
		{"empty slice", []int64{}, 0},
		{"single element", []int64{42}, 0},
		{"two elements equal", []int64{10, 10}, 0},
		{"simple values", []int64{10, 12, 23, 23, 16, 23, 21, 16}, 4.898979485566356},
		{"increasing sequence", []int64{1, 2, 3, 4, 5}, 1.414213562},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StandardDeviation(tt.input)
			if !almostEqual(result, tt.expected, 1e-9) {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// Test FindMinIgnoringFirst function
func TestFindMinIgnoringFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    []int64
		expected int64
	}{
		{"empty slice", []int64{}, 0},
		{"single element", []int64{10}, 0},
		{"two elements, second min", []int64{20, 10}, 10},
		{"multiple elements", []int64{20, 15, 8, 12, 30}, 8},
		{"all equal after first", []int64{5, 7, 7, 7}, 7},
		{"only two elements equal", []int64{3, 3}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindMinIgnoringFirst(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}
