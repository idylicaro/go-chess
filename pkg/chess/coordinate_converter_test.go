package chess

import "testing"

func TestCoordinateConverter_ToAlgebraic(t *testing.T) {
	cc := &CoordinateConverter{}

	tests := []struct {
		name     string
		x, y     int
		expected string
		wantErr  bool
	}{
		{"bottom-left corner", 0, 7, "a1", false},
		{"bottom-right corner", 7, 7, "h1", false},
		{"top-left corner", 0, 0, "a8", false},
		{"top-right corner", 7, 0, "h8", false},
		{"center square", 4, 4, "e4", false},
		{"invalid x coordinate", -1, 4, "", true},
		{"invalid y coordinate", 4, 8, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cc.ToAlgebraic(tt.x, tt.y)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToAlgebraic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ToAlgebraic() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCoordinateConverter_FromAlgebraic(t *testing.T) {
	cc := &CoordinateConverter{}

	tests := []struct {
		name                 string
		square               string
		expectedX, expectedY int
		wantErr              bool
	}{
		{"a1", "a1", 0, 7, false},
		{"h1", "h1", 7, 7, false},
		{"a8", "a8", 0, 0, false},
		{"h8", "h8", 7, 0, false},
		{"e4", "e4", 4, 4, false},
		{"invalid format", "e", 0, 0, true},
		{"invalid file", "z4", 0, 0, true},
		{"invalid rank", "e9", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, err := cc.FromAlgebraic(tt.square)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromAlgebraic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (x != tt.expectedX || y != tt.expectedY) {
				t.Errorf("FromAlgebraic() = (%v, %v), want (%v, %v)", x, y, tt.expectedX, tt.expectedY)
			}
		})
	}
}
