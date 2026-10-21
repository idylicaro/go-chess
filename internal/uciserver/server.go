// Package uciserver speaks the UCI protocol for the engine, so it can play
// in GUIs and tournament managers (#42). It reads commands line by line,
// searches in the background and answers "isready" and "stop" while a
// search runs.
package uciserver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/idylicaro/go-chess/internal/search"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// Identity is what the engine answers to "uci".
type Identity struct {
	Name, Author string
}

// SearcherFactory builds a searcher with the current options. The server
// calls it at start-up and again when an option the searcher depends on
// changes, such as Hash.
type SearcherFactory func(options search.Options) *search.Searcher

// Server is one UCI session. Its methods are not safe for concurrent use;
// Run drives it from a single goroutine.
type Server struct {
	identity    Identity
	out         *lineWriter
	newSearcher SearcherFactory
	options     search.Options
	searcher    *search.Searcher
	board       *chess.SearchBoard // the position "go" searches
	running     *runningSearch     // the last search started; nil before the first
}

// runningSearch is a search in its own goroutine.
type runningSearch struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// lineWriter serializes the replies of the command loop and the search.
type lineWriter struct {
	mu  sync.Mutex
	out io.Writer
}

func (w *lineWriter) printf(format string, args ...any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.out, format+"\n", args...)
}

// New returns a server that writes its replies to out.
//
//	server := uciserver.New(identity, os.Stdout, func(o search.Options) *search.Searcher {
//		return search.New(eval.Evaluate, o)
//	})
//	err := server.Run(ctx, os.Stdin)
func New(identity Identity, out io.Writer, newSearcher SearcherFactory) *Server {
	options := search.Options{HashMB: search.DefaultHashMB}
	return &Server{
		identity:    identity,
		out:         &lineWriter{out: out},
		newSearcher: newSearcher,
		options:     options,
		searcher:    newSearcher(options),
		board:       chess.NewSearchBoard(chess.NewGame()),
	}
}

// Run reads commands from in until "quit", the end of in or the end of
// ctx, and stops any search before it returns.
func (s *Server) Run(ctx context.Context, in io.Reader) error {
	defer s.stopSearch()
	lines := bufio.NewScanner(in)
	for lines.Scan() {
		if ctx.Err() != nil || s.handle(ctx, lines.Text()) {
			return ctx.Err()
		}
	}
	return lines.Err()
}

// handle runs one command and reports whether it was "quit". Unknown
// commands are ignored, as the protocol asks.
func (s *Server) handle(ctx context.Context, line string) (quit bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	command, args := fields[0], fields[1:]
	if command == "quit" {
		return true
	}
	if run, known := commandHandlers[command]; known {
		if err := run(s, ctx, args); err != nil {
			s.out.printf("info string %v", err)
		}
	}
	return false
}

// commandHandlers are the UCI commands the server understands.
var commandHandlers = map[string]func(s *Server, ctx context.Context, args []string) error{
	"uci":        (*Server).handleUCI,
	"isready":    (*Server).handleIsReady,
	"ucinewgame": (*Server).handleNewGame,
	"setoption":  (*Server).handleSetOption,
	"position":   (*Server).handlePosition,
	"go":         (*Server).handleGo,
	"stop":       (*Server).handleStop,
	"ponderhit":  (*Server).handleStop,
}

func (s *Server) handleUCI(context.Context, []string) error {
	s.out.printf("id name %s", s.identity.Name)
	s.out.printf("id author %s", s.identity.Author)
	for _, option := range optionDeclarations {
		s.out.printf("%s", option)
	}
	s.out.printf("uciok")
	return nil
}

// handleIsReady answers at once, even during a search.
func (s *Server) handleIsReady(context.Context, []string) error {
	s.out.printf("readyok")
	return nil
}

func (s *Server) handleNewGame(context.Context, []string) error {
	s.stopSearch()
	s.searcher.Clear()
	s.board = chess.NewSearchBoard(chess.NewGame())
	return nil
}

func (s *Server) handleStop(context.Context, []string) error {
	s.stopSearch()
	return nil
}

// stopSearch ends the running search, if any, and waits for its bestmove.
func (s *Server) stopSearch() {
	if s.running == nil {
		return
	}
	s.running.cancel()
	<-s.running.done
}
