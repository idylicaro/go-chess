// Package engine is a client for UCI chess engines such as Stockfish. It
// starts the engine as a child process, or talks over any pair of streams,
// and runs one search at a time.
package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// handshakeTimeout bounds the wait for "uciok" and "readyok".
const handshakeTimeout = 10 * time.Second

// stopGrace is how long Search waits for "bestmove" after a "stop".
const stopGrace = 2 * time.Second

// Limits bound a search. With no field set the search is infinite until the
// context ends. Zero fields are omitted from the "go" command.
type Limits struct {
	Depth    int
	MoveTime time.Duration
	// Clocks for time-managed play; the engine splits the time itself.
	WTime, BTime, WInc, BInc time.Duration
}

func (l Limits) command() string {
	parts := []string{"go"}
	if l.Depth > 0 {
		parts = append(parts, "depth", fmt.Sprint(l.Depth))
	}
	if l.MoveTime > 0 {
		parts = append(parts, "movetime", fmt.Sprint(l.MoveTime.Milliseconds()))
	}
	for _, c := range []struct {
		name  string
		value time.Duration
		set   bool
	}{
		{"wtime", l.WTime, l.WTime > 0 || l.BTime > 0},
		{"btime", l.BTime, l.WTime > 0 || l.BTime > 0},
		{"winc", l.WInc, l.WInc > 0},
		{"binc", l.BInc, l.BInc > 0},
	} {
		if c.set {
			parts = append(parts, c.name, fmt.Sprint(max(c.value, 0).Milliseconds()))
		}
	}
	if len(parts) == 1 {
		parts = append(parts, "infinite")
	}
	return strings.Join(parts, " ")
}

// Result is the engine's answer to a search.
type Result struct {
	Best   string // best move in UCI notation; "(none)" when there is no legal move
	Ponder string
}

// Engine is a running UCI engine. Its methods are safe to call from several
// goroutines, but searches are serialized.
type Engine struct {
	// Name is the engine's "id name", empty if it did not send one.
	Name string

	in    io.WriteCloser
	lines chan string
	wait  func() error
	kill  func()

	mu     sync.Mutex // serializes commands and searches
	closed bool
}

// Start runs the engine at path (looked up in PATH if it has no slash) and
// completes the UCI handshake.
//
//	e, err := engine.Start("stockfish")
func Start(path string, args ...string) (*Engine, error) {
	cmd := exec.Command(path, args...)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot start engine %q: %w", path, err)
	}
	e, err := New(in, out, cmd.Wait, func() { _ = cmd.Process.Kill() })
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("engine %q: %w", path, err)
	}
	return e, nil
}

// New talks UCI over in (commands) and out (replies) and completes the
// handshake. wait blocks until the engine has exited and kill forces it to;
// either may be nil.
func New(in io.WriteCloser, out io.Reader, wait func() error, kill func()) (*Engine, error) {
	e := &Engine{in: in, lines: make(chan string, 256), wait: wait, kill: kill}
	go func() {
		defer close(e.lines)
		scanner := bufio.NewScanner(out)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			e.lines <- strings.TrimSpace(scanner.Text())
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), handshakeTimeout)
	defer cancel()
	if err := e.send("uci"); err != nil {
		return nil, err
	}
	if err := e.readUntil(ctx, "uciok", func(line string) {
		if name, ok := strings.CutPrefix(line, "id name "); ok {
			e.Name = name
		}
	}); err != nil {
		return nil, fmt.Errorf("no uciok (is it a UCI engine?): %w", err)
	}
	if err := e.ready(ctx); err != nil {
		return nil, err
	}
	return e, nil
}

// SetOption sets a UCI option, for example "Skill Level" to "5".
func (e *Engine) SetOption(name, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.send(fmt.Sprintf("setoption name %s value %s", name, value))
}

// NewGame tells the engine the next search belongs to a different game.
func (e *Engine) NewGame() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.send("ucinewgame"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), handshakeTimeout)
	defer cancel()
	return e.ready(ctx)
}

// Search analyzes the position fen within limits. onInfo, if not nil, gets
// every info line that carries a score or a variation. When ctx ends first,
// the search is stopped and ctx.Err() is returned.
//
//	res, err := e.Search(ctx, game.FEN(), engine.Limits{MoveTime: time.Second}, nil)
func (e *Engine) Search(ctx context.Context, fen string, limits Limits, onInfo func(Info)) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.send("position fen " + fen); err != nil {
		return Result{}, err
	}
	if err := e.send(limits.command()); err != nil {
		return Result{}, err
	}

	var result Result
	handle := func(line string) {
		if info, ok := ParseInfo(line); ok && onInfo != nil {
			onInfo(info)
		}
	}
	err := e.readUntil(ctx, "bestmove", func(line string) {
		if rest, ok := strings.CutPrefix(line, "bestmove"); ok {
			fields := strings.Fields(rest)
			if len(fields) > 0 {
				result.Best = fields[0]
			}
			if len(fields) > 2 && fields[1] == "ponder" {
				result.Ponder = fields[2]
			}
			return
		}
		handle(line)
	})
	if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
		_ = e.send("stop")
		drain, cancel := context.WithTimeout(context.Background(), stopGrace)
		defer cancel()
		_ = e.readUntil(drain, "bestmove", nil)
		return Result{}, ctx.Err()
	}
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

// Close quits the engine, killing it if it does not exit promptly.
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	_ = e.send("quit")
	_ = e.in.Close()
	e.mu.Unlock()

	if e.wait == nil {
		return nil
	}
	exited := make(chan error, 1)
	go func() { exited <- e.wait() }()
	select {
	case <-exited:
	case <-time.After(stopGrace):
		if e.kill != nil {
			e.kill()
		}
		<-exited
	}
	return nil
}

func (e *Engine) send(command string) error {
	_, err := fmt.Fprintln(e.in, command)
	if err != nil {
		return fmt.Errorf("engine is not running: %w", err)
	}
	return nil
}

func (e *Engine) ready(ctx context.Context) error {
	if err := e.send("isready"); err != nil {
		return err
	}
	return e.readUntil(ctx, "readyok", nil)
}

// readUntil feeds every line to visit until one starts with prefix (that
// line included), the stream ends or ctx is done.
func (e *Engine) readUntil(ctx context.Context, prefix string, visit func(string)) error {
	for {
		select {
		case line, ok := <-e.lines:
			if !ok {
				return errors.New("engine exited")
			}
			if visit != nil {
				visit(line)
			}
			if line == prefix || strings.HasPrefix(line, prefix+" ") {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
