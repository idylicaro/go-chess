// Package pgnread reads games from PGN text and replays them on a
// chess.Game. It accepts the export format that Game.PGN writes and the
// looser files found in the wild: comments, variations, NAGs and several
// games per file.
package pgnread

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Game is one game as written in the PGN text.
type Game struct {
	// Tags in file order.
	Tags []Tag
	// Moves are the SAN tokens of the main line, without numbers or comments.
	Moves []string
	// Result is the termination marker ("1-0", "0-1", "1/2-1/2" or "*"), or
	// empty if the movetext did not end with one.
	Result string
}

// Tag is one "[Name "Value"]" pair.
type Tag struct{ Name, Value string }

// Tag returns the value of the named tag, or "" if it is absent.
func (g Game) Tag(name string) string {
	for _, t := range g.Tags {
		if t.Name == name {
			return t.Value
		}
	}
	return ""
}

var (
	moveNumber = regexp.MustCompile(`^\d+\.+`)
	results    = map[string]bool{"1-0": true, "0-1": true, "1/2-1/2": true, "*": true}
)

// Parse reads every game in text. It fails on an unterminated tag, comment
// or variation, and on text that holds no game at all.
//
//	games, err := pgnread.Parse("[White \"A\"]\n\n1. e4 e5 1-0")
func Parse(text string) ([]Game, error) {
	p := parser{src: text}
	if err := p.run(); err != nil {
		return nil, err
	}
	if len(p.games) == 0 {
		return nil, fmt.Errorf("no PGN game found")
	}
	return p.games, nil
}

type parser struct {
	src     string
	pos     int
	games   []Game
	current Game
	started bool // current holds tags, moves or a result
	inMoves bool // movetext has begun for current
}

func (p *parser) run() error {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			p.pos++
		case c == '%' && p.atLineStart():
			p.skipLine()
		case c == ';':
			p.skipLine()
		case c == '{':
			if err := p.skipUntil('}', "comment"); err != nil {
				return err
			}
		case c == '(':
			if err := p.skipVariation(); err != nil {
				return err
			}
		case c == '[' && !p.inMoves || c == '[' && p.atLineStart():
			if err := p.tag(); err != nil {
				return err
			}
		default:
			p.word()
		}
	}
	p.flush()
	return nil
}

func (p *parser) atLineStart() bool {
	return p.pos == 0 || p.src[p.pos-1] == '\n'
}

func (p *parser) skipLine() {
	for p.pos < len(p.src) && p.src[p.pos] != '\n' {
		p.pos++
	}
}

func (p *parser) skipUntil(end byte, what string) error {
	i := strings.IndexByte(p.src[p.pos:], end)
	if i < 0 {
		return fmt.Errorf("unterminated %s at offset %d", what, p.pos)
	}
	p.pos += i + 1
	return nil
}

// skipVariation skips a possibly nested "( ... )", ignoring parentheses
// inside comments.
func (p *parser) skipVariation() error {
	start, depth := p.pos, 0
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				p.pos++
				return nil
			}
		case '{':
			if err := p.skipUntil('}', "comment"); err != nil {
				return err
			}
			continue
		}
		p.pos++
	}
	return fmt.Errorf("unterminated variation at offset %d", start)
}

// tag reads `[Name "Value"]`, honoring \" and \\ inside the value.
func (p *parser) tag() error {
	start := p.pos
	p.pos++ // [
	nameStart := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != ' ' && p.src[p.pos] != '"' && p.src[p.pos] != ']' {
		p.pos++
	}
	name := p.src[nameStart:p.pos]
	for p.pos < len(p.src) && p.src[p.pos] == ' ' {
		p.pos++
	}
	if p.pos >= len(p.src) || p.src[p.pos] != '"' || name == "" {
		return fmt.Errorf("malformed tag at offset %d", start)
	}
	p.pos++ // opening quote
	var value strings.Builder
	for {
		if p.pos >= len(p.src) {
			return fmt.Errorf("unterminated tag at offset %d", start)
		}
		c := p.src[p.pos]
		if c == '\\' && p.pos+1 < len(p.src) {
			value.WriteByte(p.src[p.pos+1])
			p.pos += 2
			continue
		}
		p.pos++
		if c == '"' {
			break
		}
		value.WriteByte(c)
	}
	for p.pos < len(p.src) && p.src[p.pos] == ' ' {
		p.pos++
	}
	if p.pos >= len(p.src) || p.src[p.pos] != ']' {
		return fmt.Errorf("malformed tag at offset %d: missing ]", start)
	}
	p.pos++
	if p.inMoves { // a tag after movetext starts the next game
		p.flush()
	}
	p.current.Tags = append(p.current.Tags, Tag{name, value.String()})
	p.started = true
	return nil
}

// word reads one movetext token: a move, a move number, a NAG or a result.
func (p *parser) word() {
	start := p.pos
	for p.pos < len(p.src) && !strings.ContainsRune(" \t\r\n{(;", rune(p.src[p.pos])) {
		p.pos++
	}
	token := p.src[start:p.pos]
	switch {
	case results[token]:
		p.current.Result = token
		p.started = true
		p.flush()
		return
	case strings.HasPrefix(token, "$"):
		return
	}
	token = moveNumber.ReplaceAllString(token, "")
	if token == "" || token == "..." {
		return
	}
	p.current.Moves = append(p.current.Moves, token)
	p.started, p.inMoves = true, true
}

func (p *parser) flush() {
	if p.started {
		p.games = append(p.games, p.current)
	}
	p.current, p.started, p.inMoves = Game{}, false, false
}

// Replay plays the game from its FEN tag (or the standard position) and
// returns the final position along with the moves, in order.
//
//	game, moves, err := parsed.Replay()
func (g Game) Replay() (*chess.Game, []chess.Move, error) {
	game := chess.NewGame()
	if fen := g.Tag("FEN"); fen != "" {
		var err error
		if game, err = chess.ParseFEN(fen); err != nil {
			return nil, nil, fmt.Errorf("bad FEN tag: %w", err)
		}
	}
	moves := make([]chess.Move, 0, len(g.Moves))
	for i, san := range g.Moves {
		move, err := game.ParseSAN(san)
		if err != nil {
			return nil, nil, fmt.Errorf("move %d (%s): %w", i/2+1, san, err)
		}
		if err := game.Move(move); err != nil {
			return nil, nil, fmt.Errorf("move %d (%s): %w", i/2+1, san, err)
		}
		moves = append(moves, move)
	}
	return game, moves, nil
}
