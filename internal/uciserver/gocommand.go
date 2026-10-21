package uciserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/idylicaro/go-chess/internal/search"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// goRequest is a parsed "go" command.
type goRequest struct {
	limits search.Limits
	// waitForStop holds "bestmove" back until "stop": UCI forbids sending
	// it on its own during "go infinite" and "go ponder".
	waitForStop bool
}

// handleGo starts a search of the current position in the background.
// Arguments it does not understand are reported and skipped: a GUI waits
// for "bestmove" after every "go", so refusing to search would hang it.
func (s *Server) handleGo(ctx context.Context, args []string) error {
	s.stopSearch()
	request, ignored := parseGo(args)
	if len(ignored) > 0 {
		s.out.printf("info string go: ignoring %q, expected depth, nodes, movetime, wtime, btime, winc, binc, movestogo, infinite or ponder", strings.Join(ignored, " "))
	}
	s.startSearch(ctx, request)
	return nil
}

func (s *Server) startSearch(ctx context.Context, request goRequest) {
	searchCtx, cancel := context.WithCancel(ctx)
	run := &runningSearch{cancel: cancel, done: make(chan struct{})}
	s.running = run
	board, searcher := s.board.Clone(), s.searcher
	go func() {
		defer close(run.done)
		result := searcher.Search(searchCtx, board, request.limits, s.reportInfo)
		if request.waitForStop {
			<-searchCtx.Done()
		}
		s.out.printf("%s", bestMoveLine(result))
	}()
}

// reportInfo prints one "info" line per completed depth.
func (s *Server) reportInfo(info search.Info) {
	s.out.printf("%s", infoLine(info))
}

// infoLine formats an iteration, for example
// "info depth 5 score cp 31 nodes 23238 nps 2148778 time 10 pv e2e4 e7e5".
func infoLine(info search.Info) string {
	milliseconds := info.Elapsed.Milliseconds()
	nps := int64(0)
	if info.Elapsed > 0 {
		nps = int64(float64(info.Nodes) / info.Elapsed.Seconds())
	}
	return fmt.Sprintf("info depth %d score %s nodes %d nps %d time %d pv %s",
		info.Depth, scoreField(info.Score), info.Nodes, nps, milliseconds, joinMoves(info.PV))
}

// scoreField is "cp <centipawns>" or "mate <moves>".
func scoreField(score int) string {
	if moves, mate := search.MateInMoves(score); mate {
		return fmt.Sprintf("mate %d", moves)
	}
	return fmt.Sprintf("cp %d", score)
}

// bestMoveLine is "bestmove <move> [ponder <reply>]"; "bestmove 0000" when
// there is no legal move.
func bestMoveLine(result search.Result) string {
	line := "bestmove " + result.Best.String()
	if len(result.PV) > 1 {
		line += " ponder " + result.PV[1].String()
	}
	return line
}

func joinMoves(moves []chess.SearchMove) string {
	names := make([]string, len(moves))
	for i, move := range moves {
		names[i] = move.String()
	}
	return strings.Join(names, " ")
}

// goSetters fill the limits from a "go" argument and its value.
var goSetters = map[string]func(request *goRequest, value int64){
	"depth":     func(r *goRequest, v int64) { r.limits.Depth = int(v) },
	"nodes":     func(r *goRequest, v int64) { r.limits.Nodes = v },
	"movetime":  func(r *goRequest, v int64) { r.limits.MoveTime = milliseconds(v) },
	"wtime":     func(r *goRequest, v int64) { r.limits.WTime = milliseconds(v) },
	"btime":     func(r *goRequest, v int64) { r.limits.BTime = milliseconds(v) },
	"winc":      func(r *goRequest, v int64) { r.limits.WInc = milliseconds(v) },
	"binc":      func(r *goRequest, v int64) { r.limits.BInc = milliseconds(v) },
	"movestogo": func(r *goRequest, v int64) { r.limits.MovesToGo = int(v) },
}

// parseGo reads the arguments of "go" and returns the ones it skipped.
// "infinite" and "ponder" search until "stop"; a negative clock (some GUIs
// send one after a flag) counts as zero.
//
//	request, ignored := parseGo([]string{"wtime", "60000", "btime", "60000"})
func parseGo(args []string) (request goRequest, ignored []string) {
	for i := 0; i < len(args); i++ {
		name := args[i]
		if name == "infinite" || name == "ponder" {
			request.limits.Infinite, request.waitForStop = true, true
			continue
		}
		value, err := goValue(args, i)
		if goSetters[name] == nil || err != nil {
			ignored = append(ignored, name)
			continue
		}
		goSetters[name](&request, max(value, 0))
		i++
	}
	return request, ignored
}

func goValue(args []string, i int) (int64, error) {
	if i+1 >= len(args) {
		return 0, fmt.Errorf("go: %s needs a number after it", args[i])
	}
	return strconv.ParseInt(args[i+1], 10, 64)
}

func milliseconds(value int64) time.Duration {
	return time.Duration(value) * time.Millisecond
}
