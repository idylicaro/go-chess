package uciserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/idylicaro/go-chess/internal/book"
	"github.com/idylicaro/go-chess/internal/search"
)

// Hash option bounds, in megabytes, and Threads bounds.
const (
	minHashMB  = 1
	maxHashMB  = 4096
	maxThreads = 256
)

// optionDeclarations are sent in reply to "uci".
var optionDeclarations = []string{
	fmt.Sprintf("option name Hash type spin default 16 min %d max %d", minHashMB, maxHashMB),
	"option name Clear Hash type button",
	fmt.Sprintf("option name Threads type spin default 1 min 1 max %d", maxThreads),
	"option name OwnBook type check default false",
	"option name BookFile type string default <empty>",
	"option name NullMove type check default true",
	"option name LMR type check default true",
	"option name Futility type check default true",
}

// pruningSwitches map the check options to the technique they turn on or
// off, so lab #44 can match the engine against itself without one.
var pruningSwitches = map[string]func(pruning *search.Pruning) *bool{
	"nullmove": func(p *search.Pruning) *bool { return &p.NullMove },
	"lmr":      func(p *search.Pruning) *bool { return &p.LateMoveReductions },
	"futility": func(p *search.Pruning) *bool { return &p.Futility },
}

// optionSetters apply the options other than the pruning switches, keyed
// by lower-case name.
var optionSetters = map[string]func(s *Server, value string) error{
	"hash":       (*Server).setHash,
	"clear hash": func(s *Server, _ string) error { s.searcher.Clear(); return nil },
	"threads":    (*Server).setThreads,
	"ownbook":    (*Server).setOwnBook,
	"bookfile":   (*Server).setBookFile,
}

// handleSetOption reads "setoption name <name> [value <value>]"; names may
// have spaces and are matched without regard to case, as UCI asks.
func (s *Server) handleSetOption(_ context.Context, args []string) error {
	s.stopSearch()
	name, value, err := parseSetOption(args)
	if err != nil {
		return err
	}
	key := strings.ToLower(name)
	if set, known := optionSetters[key]; known {
		return set(s, value)
	}
	if field, known := pruningSwitches[key]; known {
		return s.setPruning(field, value)
	}
	return fmt.Errorf("setoption: unknown option %q, expected Hash, Clear Hash, Threads, OwnBook, BookFile, NullMove, LMR or Futility", name)
}

func parseSetOption(args []string) (name, value string, err error) {
	if len(args) < 2 || args[0] != "name" {
		return "", "", fmt.Errorf("setoption: got %q, expected \"name <name> [value <value>]\"", strings.Join(args, " "))
	}
	rest := strings.Join(args[1:], " ")
	name, value, _ = strings.Cut(rest, " value ")
	return strings.TrimSpace(name), strings.TrimSpace(value), nil
}

// setHash resizes the transposition table, which empties it.
func (s *Server) setHash(value string) error {
	megabytes, err := strconv.Atoi(value)
	if err != nil || megabytes < minHashMB || megabytes > maxHashMB {
		return fmt.Errorf("setoption: Hash %q, expected megabytes between %d and %d", value, minHashMB, maxHashMB)
	}
	s.options.HashMB = megabytes
	s.searcher = s.newSearcher(s.options)
	return nil
}

// setThreads sets how many goroutines search together (Lazy SMP, #45).
func (s *Server) setThreads(value string) error {
	threads, err := strconv.Atoi(value)
	if err != nil || threads < 1 || threads > maxThreads {
		return fmt.Errorf("setoption: Threads %q, expected a number between 1 and %d", value, maxThreads)
	}
	s.options.Threads = threads
	s.searcher = s.newSearcher(s.options)
	return nil
}

func (s *Server) setOwnBook(value string) error {
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("setoption: OwnBook %q, expected true or false", value)
	}
	s.ownBook = enabled
	return nil
}

// setBookFile loads a Polyglot book (lab #46); "<empty>" or "" unloads it.
func (s *Server) setBookFile(path string) error {
	if path == "" || path == "<empty>" {
		s.openingBook = nil
		return nil
	}
	opening, err := book.Open(path)
	if err != nil {
		return fmt.Errorf("setoption: BookFile: %w", err)
	}
	s.openingBook = opening
	return nil
}

// setPruning turns a technique on or off; it rebuilds the searcher, which
// empties the transposition table.
func (s *Server) setPruning(field func(pruning *search.Pruning) *bool, value string) error {
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("setoption: %q is not a check value, expected true or false", value)
	}
	*field(&s.options.Pruning) = enabled
	s.searcher = s.newSearcher(s.options)
	return nil
}
