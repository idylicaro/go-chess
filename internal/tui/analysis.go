package tui

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/idylicaro/go-chess/internal/engine"
	"github.com/idylicaro/go-chess/internal/render"
	"github.com/idylicaro/go-chess/pkg/chess"
)

// analysisDepths are the depths searched one after the other. Each search
// ends in a message, which is how the interface hears from the engine while
// it is still deepening; a single endless search would only speak at the end.
var analysisDepths = []int{6, 10, 14, 18, 22}

// analysisMsg is the engine's last word at one depth of a position.
type analysisMsg struct {
	gen  int
	step int
	info engine.Info
	ok   bool // info carries a score
	err  error
}

// analyzing says whether the engine is analyzing instead of playing.
func (m Model) analyzing() bool { return m.cfg.Analyze && m.cfg.Engine != nil }

// syncAnalysis starts a new analysis when the position is not the one being
// analyzed, whatever changed it: a move, a take-back or returning to the game.
func (m Model) syncAnalysis() (Model, tea.Cmd) {
	if !m.analyzing() || m.screen != screenGame {
		return m, nil
	}
	fen := m.game.FEN()
	if fen == m.analysisFEN {
		return m, nil
	}
	m = m.stopSearch()
	m.analysisFEN, m.hasAnalysis = fen, false
	if m.game.IsOver() {
		return m, nil
	}
	return m.analysisStep(0)
}

// analysisStep searches the position to analysisDepths[step], in a command.
func (m Model) analysisStep(step int) (Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(m.baseContext())
	m.cancelSearch = cancel
	gen, fen, eng := m.searchGen, m.analysisFEN, m.cfg.Engine
	depth := analysisDepths[step]
	return m, func() tea.Msg {
		var last engine.Info
		var seen bool
		_, err := eng.Search(ctx, fen, engine.Limits{Depth: depth}, func(info engine.Info) {
			if info.HasScore && info.MultiPV <= 1 {
				last, seen = info, true
			}
		})
		return analysisMsg{gen: gen, step: step, info: last, ok: seen, err: err}
	}
}

func (m Model) updateAnalysis(msg analysisMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.searchGen {
		return m, nil // an analysis of a position the player has left
	}
	if msg.err != nil {
		m.cancelSearch = nil
		if !errors.Is(msg.err, context.Canceled) {
			m.message = fmt.Sprintf("The engine failed: %v", msg.err)
		}
		return m, nil
	}
	if msg.ok {
		m.analysis, m.hasAnalysis = msg.info, true
	}
	if msg.step+1 < len(analysisDepths) {
		return m.analysisStep(msg.step + 1)
	}
	m.cancelSearch = nil
	return m, nil
}

// analysisPanel is the engine's view of the position for the side panel, or
// nil outside analysis mode.
func (m Model) analysisPanel() *render.Analysis {
	if !m.analyzing() || m.game.IsOver() {
		return nil
	}
	if !m.hasAnalysis {
		return &render.Analysis{Pending: true}
	}
	eval := render.Eval{Mate: m.analysis.Score.Mate, Value: m.analysis.Score.Value}
	if m.game.GetCurrentTurn() == chess.Black { // engines score for the side to move
		eval.Value = -eval.Value
	}
	return &render.Analysis{
		Eval:  eval,
		Depth: m.analysis.Depth,
		Line:  engine.Line(m.analysisFEN, m.analysis.PV),
		Style: m.cfg.Style,
	}
}
