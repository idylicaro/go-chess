package uciserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Hash option bounds, in megabytes.
const (
	minHashMB = 1
	maxHashMB = 4096
)

// optionDeclarations are sent in reply to "uci".
var optionDeclarations = []string{
	fmt.Sprintf("option name Hash type spin default 16 min %d max %d", minHashMB, maxHashMB),
	"option name Clear Hash type button",
}

// handleSetOption reads "setoption name <name> [value <value>]"; names may
// have spaces and are matched without regard to case, as UCI asks.
func (s *Server) handleSetOption(_ context.Context, args []string) error {
	s.stopSearch()
	name, value, err := parseSetOption(args)
	if err != nil {
		return err
	}
	switch strings.ToLower(name) {
	case "hash":
		return s.setHash(value)
	case "clear hash":
		s.searcher.Clear()
		return nil
	}
	return fmt.Errorf("setoption: unknown option %q, expected Hash or Clear Hash", name)
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
