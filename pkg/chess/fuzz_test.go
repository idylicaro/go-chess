package chess

import (
	"strings"
	"testing"
)

// FuzzParseFEN checks that arbitrary input never panics and that every
// accepted FEN survives a format/parse round trip.
func FuzzParseFEN(f *testing.F) {
	for _, seed := range []string{
		StartingFEN,
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"8/8/8/8/8/8/8/K6k w - -",
		"",
		"not a fen",
		"9/8/8/8/8/8/8/8 w - - 0 1",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, fen string) {
		game, err := ParseFEN(fen)
		if err != nil {
			return
		}
		formatted := game.FEN()
		again, err := ParseFEN(formatted)
		if err != nil {
			t.Fatalf("FEN %q formatted as %q, which does not reparse: %v", fen, formatted, err)
		}
		if again.FEN() != formatted {
			t.Fatalf("FEN round trip changed %q into %q", formatted, again.FEN())
		}
	})
}

// FuzzSAN checks that ParseSAN never panics and that any move it accepts is
// legal and formats back to a SAN that parses to the same move.
func FuzzSAN(f *testing.F) {
	for _, seed := range []string{
		"e4", "Nf3", "O-O", "O-O-O", "exd5", "e8=Q+", "Qh4#", "Rad1", "R1a3",
		"Nbd7", "0-0", "e8Q", "", "xx", "Z9", "a1=K",
	} {
		f.Add(StartingFEN, seed)
	}
	f.Add("r3k2r/pPppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPpP/R3K2R w KQkq - 0 1", "bxa8=N")
	f.Fuzz(func(t *testing.T, fen, san string) {
		game, err := ParseFEN(fen)
		if err != nil {
			return
		}
		move, err := game.ParseSAN(san)
		if err != nil {
			return
		}
		formatted, err := game.SAN(move)
		if err != nil {
			t.Fatalf("ParseSAN(%q) in %q gave %v, which SAN rejects: %v", san, fen, move, err)
		}
		back, err := game.ParseSAN(formatted)
		if err != nil || back != move {
			t.Fatalf("SAN %q for %v reparsed as %v (err %v)", formatted, move, back, err)
		}
	})
}

// FuzzPGN plays a game chosen by the fuzz bytes (each byte picks a legal
// move) and checks that the exported PGN replays to the same final position.
func FuzzPGN(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{5, 12, 7, 3, 0, 9, 200, 41})
	f.Add([]byte{255, 254, 253, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	f.Fuzz(func(t *testing.T, choices []byte) {
		game := NewGame()
		for _, choice := range choices {
			moves := game.LegalMoves()
			if len(moves) == 0 {
				break
			}
			if err := game.Move(moves[int(choice)%len(moves)]); err != nil {
				t.Fatalf("legal move rejected: %v", err)
			}
		}
		pgn, err := game.PGN(PGNHeaders{White: `A "quoted" \ name`, Black: "B"})
		if err != nil {
			t.Fatalf("PGN export failed: %v", err)
		}
		replay := NewGame()
		for _, token := range pgnMovetext(pgn) {
			move, err := replay.ParseSAN(token)
			if err != nil {
				t.Fatalf("exported SAN %q does not parse: %v\n%s", token, err, pgn)
			}
			if err := replay.Move(move); err != nil {
				t.Fatalf("exported SAN %q is not playable: %v\n%s", token, err, pgn)
			}
		}
		if replay.FEN() != game.FEN() {
			t.Fatalf("PGN replay ended at %q, want %q\n%s", replay.FEN(), game.FEN(), pgn)
		}
	})
}

// pgnMovetext returns the SAN tokens of a PGN export, without tags, move
// numbers or the result.
func pgnMovetext(pgn string) []string {
	var tokens []string
	for _, line := range strings.Split(pgn, "\n") {
		if strings.HasPrefix(line, "[") {
			continue
		}
		for _, token := range strings.Fields(line) {
			if strings.HasSuffix(token, ".") || strings.Contains(token, "-") && strings.ContainsAny(token, "01*") && !strings.HasPrefix(token, "O") {
				continue
			}
			if token == "*" {
				continue
			}
			tokens = append(tokens, token)
		}
	}
	return tokens
}
