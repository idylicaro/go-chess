package match

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/pkg/chess"
)

//go:embed openings.txt
var builtinOpenings string

// Opening is a line of moves in UCI notation from the starting position.
type Opening []string

// BuiltinOpenings returns the openings shipped with the harness.
func BuiltinOpenings() []Opening {
	openings, err := ParseOpenings(builtinOpenings)
	if err != nil {
		panic(fmt.Sprintf("built-in openings do not parse: %v", err))
	}
	return openings
}

// ParseOpenings reads one line of SAN moves per opening; blank lines and
// lines starting with # are skipped.
//
//	openings, err := ParseOpenings("e4 e5 Nf3 Nc6\nd4 d5 c4")
func ParseOpenings(text string) ([]Opening, error) {
	var openings []Opening
	for number, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		opening, err := parseOpeningLine(line)
		if err != nil {
			return nil, fmt.Errorf("openings line %d: %w", number+1, err)
		}
		openings = append(openings, opening)
	}
	if len(openings) == 0 {
		return nil, fmt.Errorf("no openings found: expected one line of SAN moves per opening, like \"e4 e5 Nf3\"")
	}
	return openings, nil
}

func parseOpeningLine(line string) (Opening, error) {
	game := chess.NewGame()
	var opening Opening
	for _, san := range strings.Fields(line) {
		move, err := game.ParseSAN(san)
		if err == nil {
			err = game.Move(move)
		}
		if err != nil {
			return nil, fmt.Errorf("move %q of %q: %w", san, line, err)
		}
		opening = append(opening, move.String())
	}
	return opening, nil
}
