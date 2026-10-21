package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// parseTimeControl reads "5+3" (5 minutes plus 3 seconds per move) or "10"
// (10 minutes, no increment).
func parseTimeControl(text string) (chess.TimeConfig, error) {
	minutesText, incrementText, hasIncrement := strings.Cut(text, "+")
	minutes, err := strconv.ParseFloat(minutesText, 64)
	if err != nil || minutes <= 0 {
		return chess.TimeConfig{}, fmt.Errorf("invalid time control %q: expected minutes plus an optional increment in seconds, like 5+3 or 10", text)
	}
	config := chess.TimeConfig{InitialTime: time.Duration(minutes * float64(time.Minute))}
	if hasIncrement {
		seconds, err := strconv.ParseFloat(incrementText, 64)
		if err != nil || seconds < 0 {
			return chess.TimeConfig{}, fmt.Errorf("invalid time control %q: the increment must be a number of seconds, like 5+3", text)
		}
		config.Increment = time.Duration(seconds * float64(time.Second))
	}
	return config, nil
}

// formatTimeControl is the PGN TimeControl value of config: "300+3".
func formatTimeControl(config chess.TimeConfig) string {
	seconds := int(config.InitialTime.Seconds())
	if config.Increment == 0 {
		return strconv.Itoa(seconds)
	}
	return fmt.Sprintf("%d+%d", seconds, int(config.Increment.Seconds()))
}
