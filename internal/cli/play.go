package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/idylicaro/go-chess/internal/moveinput"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// maxSkillLevel is the top of the "Skill Level" scale that UCI engines such
// as Stockfish use.
const maxSkillLevel = 20

// playOptions are the flags of the play command.
type playOptions struct {
	fen        string
	timeText   string
	enginePath string
	level      int
	color      string
	pgnOut     string
	style      *styleFlags
}

func runPlay(e env, args []string) error {
	fs := newFlagSet(e, "play", "[flags]", "Play a game in the terminal. Moves are typed in SAN (Nf3, exd5, O-O) or UCI (e2e4).\nType \"help\" during the game for the other commands.")
	var opts playOptions
	fs.StringVar(&opts.fen, "fen", chess.StartingFEN, "position to start from")
	fs.StringVar(&opts.timeText, "time", "", "time control as minutes+increment seconds, like 5+3 (default: untimed)")
	fs.StringVar(&opts.enginePath, "engine", "", "UCI engine to play against, like stockfish (default: two players at one keyboard)")
	fs.IntVar(&opts.level, "level", -1, "engine strength from 0 to 20 (needs --engine)")
	fs.StringVar(&opts.color, "color", "white", "your side against the engine: white or black (needs --engine)")
	fs.StringVar(&opts.pgnOut, "pgn-out", "", "write the game as PGN to this file when it ends, or to standard output with -")
	opts.style = addStyleFlags(fs)
	rest, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usagef("unexpected argument %q", rest[0])
	}
	s, err := newSession(e, opts)
	if err != nil {
		return err
	}
	defer s.close()
	s.pgnOut = opts.pgnOut
	err = s.loop()
	if exportErr := s.exportPGN(); exportErr != nil && err == nil {
		err = exportErr
	}
	return err
}

// newSession builds the game the flags describe and, with --engine, starts
// the engine.
func newSession(e env, opts playOptions) (*session, error) {
	if opts.enginePath == "" && (opts.level >= 0 || opts.color != "white") {
		return nil, usagef("--level and --color only apply when playing against an engine; add --engine")
	}
	if opts.level > maxSkillLevel {
		return nil, usagef("invalid --level %d: expected 0 to %d", opts.level, maxSkillLevel)
	}
	if opts.color != "white" && opts.color != "black" {
		return nil, usagef("invalid --color %q: expected white or black", opts.color)
	}

	s := &session{env: e, style: e.style(opts.style)}
	var err error
	if opts.timeText != "" {
		config, err := parseTimeControl(opts.timeText)
		if err != nil {
			return nil, usageError{err.Error()}
		}
		s.timeControl = &config
		if s.game, err = chess.ParseFENTimed(opts.fen, config, chess.SystemClock{}); err != nil {
			return nil, err
		}
		if err := s.game.StartTimer(); err != nil {
			return nil, err
		}
	} else if s.game, err = chess.ParseFEN(opts.fen); err != nil {
		return nil, err
	}

	if opts.enginePath != "" {
		humanColor := chess.White
		if opts.color == "black" {
			humanColor = chess.Black
			s.flipped = true
		}
		if err := s.startEngine(opts.enginePath, opts.level, humanColor); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// loop plays the game until it ends, the player leaves or the input runs out.
func (s *session) loop() error {
	scanner := bufio.NewScanner(s.in)
	fmt.Fprintln(s.out, "Starting Go Chess Engine...")
	for {
		if s.engineToMove() {
			if err := s.engineMove(); err != nil {
				return err
			}
		}
		s.show()
		if s.game.IsOver() {
			fmt.Fprintf(s.out, "Game over! Result: %s\n", s.game.Result())
			return nil
		}

		fmt.Fprintf(s.out, "> Enter a move (SAN or UCI) or a command, \"help\" lists them (%s): ", s.game.GetCurrentTurn())
		if !scanner.Scan() {
			return nil
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "exit" || input == "quit" {
			fmt.Fprintln(s.out, "Exiting game.")
			return nil
		}
		if s.command(input) {
			continue
		}
		move, err := moveinput.Parse(s.game, input)
		if err == nil {
			err = s.play(move)
		}
		if err != nil {
			fmt.Fprintf(s.out, "Error: %v\n", err)
		}
	}
}
