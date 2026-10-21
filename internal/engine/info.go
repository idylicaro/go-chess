package engine

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Score is an engine evaluation from the point of view of the side to move.
type Score struct {
	// Mate is true when Value counts moves to mate instead of centipawns.
	Mate  bool
	Value int
}

// String formats the score as pawns ("+0.34") or as mate distance ("#3", "#-2").
func (s Score) String() string {
	if s.Mate {
		return fmt.Sprintf("#%d", s.Value)
	}
	return fmt.Sprintf("%+.2f", float64(s.Value)/100)
}

// Info is one "info" line from the engine. Fields the line did not carry are
// zero; HasScore tells whether Score is set.
type Info struct {
	Depth    int
	MultiPV  int
	Score    Score
	HasScore bool
	Nodes    int64
	Time     time.Duration
	PV       []string // principal variation, in UCI notation
}

// ParseInfo reads an "info" line. It reports false for lines that carry
// neither a score nor a principal variation (such as "info string ...").
//
//	info, ok := ParseInfo("info depth 12 score cp 34 pv e2e4 e7e5")
func ParseInfo(line string) (Info, bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 || fields[0] != "info" {
		return Info{}, false
	}
	info := Info{MultiPV: 1}
	for i := 1; i < len(fields); i++ {
		switch fields[i] {
		case "string":
			return Info{}, false
		case "depth":
			info.Depth, i = intAfter(fields, i)
		case "multipv":
			info.MultiPV, i = intAfter(fields, i)
		case "nodes":
			var n int
			n, i = intAfter(fields, i)
			info.Nodes = int64(n)
		case "time":
			var ms int
			ms, i = intAfter(fields, i)
			info.Time = time.Duration(ms) * time.Millisecond
		case "score":
			if i+2 < len(fields) && (fields[i+1] == "cp" || fields[i+1] == "mate") {
				value, err := strconv.Atoi(fields[i+2])
				if err == nil {
					info.Score = Score{Mate: fields[i+1] == "mate", Value: value}
					info.HasScore = true
				}
				i += 2
			}
		case "pv":
			info.PV = append([]string(nil), fields[i+1:]...)
			i = len(fields)
		}
	}
	if !info.HasScore && len(info.PV) == 0 {
		return Info{}, false
	}
	return info, true
}

// intAfter parses fields[i+1] and returns the index to continue from.
func intAfter(fields []string, i int) (int, int) {
	if i+1 >= len(fields) {
		return 0, i
	}
	n, _ := strconv.Atoi(fields[i+1])
	return n, i + 1
}
