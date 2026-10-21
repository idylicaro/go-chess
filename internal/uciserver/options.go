package uciserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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

// handleSetOption reads "setoption name <name> [value <value>]"; names may
// have spaces and are matched without regard to case, as UCI asks.
func (s *Server) handleSetOption(_ context.Context, args []string) error {
	s.stopSearch()
	name, value, err := parseSetOption(args)
	if err != nil {
		return err
	}
	key := strings.ToLower(name)
	switch {
	case key == "hash":
		return s.setHash(value)
	case key == "threads":
		return s.setThreads(value)
	case key == "clear hash":
		s.searcher.Clear()
		return nil
	case pruningSwitches[key] != nil:
		return s.setPruning(pruningSwitches[key], value)
	}
	return fmt.Errorf("setoption: unknown option %q, expected Hash, Clear Hash, Threads, NullMove, LMR or Futility", name)
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
