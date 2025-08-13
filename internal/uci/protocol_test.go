package uci

import (
	common "idylicaro/go-chess/internal/common"
	"testing"
)

func TestParseCoordinate(t *testing.T) {
	var tests = []struct {
		input         string
		expectedX     int
		expectedY     int
		expectedError bool
	}{
		{"a1", 0, 7, false},  // Bottom left corner
		{"h8", 7, 0, false},  // Top right corner
		{"e2", 4, 6, false},  // Example you already had
		{"d4", 3, 4, false},  // Center of the board
		{"a9", 0, 0, true},   // Invalid coordinate (row out of bounds)
		{"i1", 0, 0, true},   // Invalid coordinate (column out of bounds)
		{"a", 0, 0, true},    // Incorrect string length
		{"a1b2", 0, 0, true}, // Incorrect string length
	}

	// 2. Iterate over the table
	for _, tt := range tests {
		// tt.input is the current test case
		actualX, actualY, err := parseCoordinate(tt.input)

		// 3. Compare results
		// Check if the expected error occurred
		if (err != nil) != tt.expectedError {
			t.Errorf("parseCoordinate(%q): expected error %v, got %v", tt.input, tt.expectedError, err)
			continue
		}

		// If error was expected, no need to check coordinates
		if tt.expectedError {
			continue
		}

		// Check if coordinates are correct
		if actualX != tt.expectedX {
			t.Errorf("parseCoordinate(%q): expected X: %d, got X: %d", tt.input, tt.expectedX, actualX)
		}
		if actualY != tt.expectedY {
			t.Errorf("parseCoordinate(%q): expected Y: %d, got Y: %d", tt.input, tt.expectedY, actualY)
		}
	}
}

func TestParseMove(t *testing.T) {
	var tests = []struct {
		input         string
		expectedMove  Move
		expectedError bool
	}{
		// Valid move scenarios (4 characters)
		{"e2e4", Move{FromX: 4, FromY: 6, ToX: 4, ToY: 4}, false},
		{"b1c3", Move{FromX: 1, FromY: 7, ToX: 2, ToY: 5}, false},

		// Pawn promotion scenarios (5 characters)
		{"a7a8q", Move{FromX: 0, FromY: 1, ToX: 0, ToY: 0, Promotion: common.Queen}, false},
		{"h2h1r", Move{FromX: 7, FromY: 6, ToX: 7, ToY: 7, Promotion: common.Rook}, false},

		// Error scenarios
		{"e2", Move{}, true},    // Incorrect length
		{"e2e4e", Move{}, true}, // Incorrect length
		{"i2e4", Move{}, true},  // Invalid coordinate (column)
		{"e2e9", Move{}, true},  // Invalid coordinate (row)
		{"a7a8p", Move{}, true}, // Invalid promotion character
	}

	// 2. Iterate over the table
	for _, tt := range tests {
		// tt.input is the current test case
		actualMove, err := ParseMove(tt.input)

		// 3. Compare results
		// Check if the expected error occurred
		if (err != nil) != tt.expectedError {
			t.Errorf("ParseMove(%q): expected error %v, got %v", tt.input, tt.expectedError, err)
			continue
		}

		// If error was expected, no need to check coordinates
		if tt.expectedError {
			continue
		}

		// Check if coordinates are correct
		if actualMove != tt.expectedMove {
			t.Errorf("ParseMove(%q): expected move: %+v, got move: %+v", tt.input, tt.expectedMove, actualMove)
		}
	}
}
