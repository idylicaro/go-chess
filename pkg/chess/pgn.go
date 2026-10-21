package chess

import (
	"fmt"
	"idylicaro/go-chess/internal/common"
	"strings"
)

// pgnLineWidth is the export-format line limit from the PGN standard.
const pgnLineWidth = 80

// PGNHeaders are the Seven Tag Roster values supplied by the caller; the
// Result tag is derived from the game. Empty values export as "?"
// ("????.??.??" for Date), as the PGN standard prescribes for unknowns.
type PGNHeaders struct {
	Event, Site, Date, Round, White, Black string
}

// PGN exports the game in PGN export format: the Seven Tag Roster, plus
// SetUp/FEN when the game did not start from the initial position, and the
// movetext in SAN with the result (#9).
//
//	pgn, err := game.PGN(PGNHeaders{White: "Alice", Black: "Bob"})
func (g *Game) PGN(headers PGNHeaders) (string, error) {
	start, err := ParseFEN(g.initialFEN)
	if err != nil {
		return "", fmt.Errorf("cannot replay game for PGN: %w", err)
	}
	sanMoves, err := g.sanHistory(start)
	if err != nil {
		return "", err
	}
	result := g.Result().String()
	movetext := wrapPGNTokens(append(numberedMoveTokens(start, sanMoves), result), pgnLineWidth)
	return g.pgnTagSection(headers, result) + "\n" + movetext + "\n", nil
}

// sanHistory replays the move history from start, formatting each move in
// the position it was played in.
func (g *Game) sanHistory(start *Game) ([]string, error) {
	replay := start
	sanMoves := make([]string, 0, len(g.moveHistory))
	for _, move := range g.moveHistory {
		san, err := replay.SAN(move)
		if err != nil {
			return nil, fmt.Errorf("cannot replay game for PGN: %w", err)
		}
		sanMoves = append(sanMoves, san)
		replay = replay.childAfter(move)
	}
	return sanMoves, nil
}

// numberedMoveTokens adds move numbers: "1." before White's moves, and "N..."
// when the game starts with Black to move. A number and its move form one
// token ("8. Nc3") so line wrapping never separates them.
func numberedMoveTokens(start *Game, sanMoves []string) []string {
	moveNumber, turn := start.fullMoveNumber, start.currentTurn
	var tokens []string
	for i, san := range sanMoves {
		token := san
		if turn == common.White {
			token = fmt.Sprintf("%d. %s", moveNumber, san)
		} else if i == 0 {
			token = fmt.Sprintf("%d... %s", moveNumber, san)
		}
		tokens = append(tokens, token)
		if turn == common.Black {
			moveNumber++
		}
		turn = opposite(turn)
	}
	return tokens
}

func (g *Game) pgnTagSection(headers PGNHeaders, result string) string {
	tags := [][2]string{
		{"Event", orUnknown(headers.Event, "?")},
		{"Site", orUnknown(headers.Site, "?")},
		{"Date", orUnknown(headers.Date, "????.??.??")},
		{"Round", orUnknown(headers.Round, "?")},
		{"White", orUnknown(headers.White, "?")},
		{"Black", orUnknown(headers.Black, "?")},
		{"Result", result},
	}
	if g.initialFEN != StartingFEN {
		tags = append(tags, [2]string{"SetUp", "1"}, [2]string{"FEN", g.initialFEN})
	}
	var section strings.Builder
	for _, tag := range tags {
		fmt.Fprintf(&section, "[%s \"%s\"]\n", tag[0], escapePGNTagValue(tag[1]))
	}
	return section.String()
}

func orUnknown(value, unknown string) string {
	if value == "" {
		return unknown
	}
	return value
}

// escapePGNTagValue escapes backslashes and quotes, as the PGN standard requires.
func escapePGNTagValue(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
}

// wrapPGNTokens joins tokens with spaces, breaking lines before width.
func wrapPGNTokens(tokens []string, width int) string {
	var lines []string
	line := ""
	for _, token := range tokens {
		if line != "" && len(line)+1+len(token) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += token
	}
	return strings.Join(append(lines, line), "\n")
}
