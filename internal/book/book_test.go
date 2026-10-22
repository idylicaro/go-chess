package book

import (
	"encoding/binary"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// specKeys are the test positions of the Polyglot format specification.
var specKeys = []struct {
	fen string
	key uint64
}{
	{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 0x463b96181691fc9c},
	{"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", 0x823c9b50fd114196},
	{"rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2", 0x0756b94461c50fb0},
	{"rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR b KQkq - 0 2", 0x662fafb965db29d4},
	{"rnbqkbnr/ppp1p1pp/8/3pPp2/8/8/PPPP1PPP/RNBQKBNR w KQkq f6 0 3", 0x22a48b5a8e47ff78},
	{"rnbqkbnr/ppp1p1pp/8/3pPp2/8/8/PPPPKPPP/RNBQ1BNR b kq - 0 3", 0x652a607ca3f242c1},
	{"rnbq1bnr/ppp1pkpp/8/3pPp2/8/8/PPPPKPPP/RNBQ1BNR w - - 0 4", 0x00fdd303c946bdd9},
	{"rnbqkbnr/p1pppppp/8/8/PpP4P/8/1P1PPPP1/RNBQKBNR b KQkq c3 0 3", 0x3c8123ea7b067637},
	{"rnbqkbnr/p1pppppp/8/8/P6P/R1p5/1P1PPPP1/1NBQKBNR b Kkq - 0 4", 0x5c3f9b829b279560},
}

func TestKey_MatchesTheSpecification(t *testing.T) {
	for _, tc := range specKeys {
		if got := Key(boardFromFEN(t, tc.fen)); got != tc.key {
			t.Errorf("Key(%q) = %016x, want %016x", tc.fen, got, tc.key)
		}
	}
}

// testEntry is a book record before encoding.
type testEntry struct {
	fen    string
	move   uint16
	weight uint16
}

// encodeMove packs a move as Polyglot does: files and rows, row 0 rank 1.
func encodeMove(fromFile, fromRow, toFile, toRow, promotion int) uint16 {
	return uint16(toFile | toRow<<3 | fromFile<<6 | fromRow<<9 | promotion<<12)
}

// encodeBook writes entries in the Polyglot layout, sorted by key.
func encodeBook(t *testing.T, entries ...testEntry) []byte {
	t.Helper()
	type record struct {
		key          uint64
		move, weight uint16
	}
	records := make([]record, len(entries))
	for i, e := range entries {
		records[i] = record{Key(boardFromFEN(t, e.fen)), e.move, e.weight}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].key < records[j].key })
	data := make([]byte, 0, len(records)*entryBytes)
	for _, r := range records {
		data = binary.BigEndian.AppendUint64(data, r.key)
		data = binary.BigEndian.AppendUint16(data, r.move)
		data = binary.BigEndian.AppendUint16(data, r.weight)
		data = binary.BigEndian.AppendUint32(data, 0)
	}
	return data
}

func TestBook_MovesAreLegalAndHeaviestFirst(t *testing.T) {
	data := encodeBook(t,
		testEntry{chess.StartingFEN, encodeMove(3, 1, 3, 3, 0), 10}, // d2d4
		testEntry{chess.StartingFEN, encodeMove(4, 1, 4, 3, 0), 30}, // e2e4
		testEntry{chess.StartingFEN, encodeMove(4, 1, 4, 5, 0), 50}, // e2e6, illegal
		testEntry{chess.StartingFEN, encodeMove(6, 0, 5, 2, 0), 0},  // g1f3, weight 0
	)
	opening, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	moves := opening.Moves(boardFromFEN(t, chess.StartingFEN))
	if len(moves) != 2 || moves[0] != (Move{"e2e4", 30}) || moves[1] != (Move{"d2d4", 10}) {
		t.Errorf("Moves = %+v, want e2e4 (30) then d2d4 (10)", moves)
	}
	if moves := opening.Moves(boardFromFEN(t, specKeys[1].fen)); len(moves) != 0 {
		t.Errorf("Moves out of book = %+v, want none", moves)
	}
}

func TestBook_PickFollowsTheWeights(t *testing.T) {
	data := encodeBook(t,
		testEntry{chess.StartingFEN, encodeMove(3, 1, 3, 3, 0), 10},
		testEntry{chess.StartingFEN, encodeMove(4, 1, 4, 3, 0), 30},
	)
	opening, _ := Parse(data)
	board := boardFromFEN(t, chess.StartingFEN)
	for roll, want := range map[int]string{0: "e2e4", 29: "e2e4", 30: "d2d4", 39: "d2d4"} {
		if got, ok := opening.Pick(board, func(int) int { return roll }); !ok || got != want {
			t.Errorf("Pick with roll %d = %q, %v, want %s", roll, got, ok, want)
		}
	}
	if _, ok := opening.Pick(boardFromFEN(t, specKeys[3].fen), func(int) int { return 0 }); ok {
		t.Error("Pick out of book reported a move")
	}
}

func TestBook_CastlingAndPromotionAreTranslated(t *testing.T) {
	castleFEN := "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1"
	promotionFEN := "4k3/P7/8/8/8/8/8/4K3 w - - 0 1"
	data := encodeBook(t,
		testEntry{castleFEN, encodeMove(4, 0, 7, 0, 0), 5},    // e1h1 = O-O
		testEntry{castleFEN, encodeMove(4, 0, 0, 0, 0), 5},    // e1a1 = O-O-O
		testEntry{promotionFEN, encodeMove(0, 6, 0, 7, 1), 5}, // a7a8n
	)
	opening, _ := Parse(data)
	var castles []string
	for _, move := range opening.Moves(boardFromFEN(t, castleFEN)) {
		castles = append(castles, move.UCI)
	}
	sort.Strings(castles)
	if strings.Join(castles, " ") != "e1c1 e1g1" {
		t.Errorf("castling moves = %v, want e1c1 and e1g1", castles)
	}
	if moves := opening.Moves(boardFromFEN(t, promotionFEN)); len(moves) != 1 || moves[0].UCI != "a7a8n" {
		t.Errorf("promotion = %+v, want a7a8n", moves)
	}
}

func TestParse_RejectsATruncatedOrUnsortedBook(t *testing.T) {
	if _, err := Parse(make([]byte, 17)); err == nil {
		t.Error("Parse of 17 bytes: want an error")
	}
	sorted := encodeBook(t, testEntry{chess.StartingFEN, 1, 1}, testEntry{specKeys[1].fen, 1, 1})
	unsorted := append(append([]byte(nil), sorted[entryBytes:]...), sorted[:entryBytes]...)
	if _, err := Parse(unsorted); err == nil {
		t.Error("Parse of an unsorted book: want an error")
	}
}

// TestOpen_RealBook reads a real book when GO_CHESS_BOOK names one, such
// as gm2001.bin (see docs/labs/3.0-C-polyglot-book.md).
func TestOpen_RealBook(t *testing.T) {
	path := os.Getenv("GO_CHESS_BOOK")
	if path == "" {
		t.Skip("set GO_CHESS_BOOK to a Polyglot book to run")
	}
	opening, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if moves := opening.Moves(boardFromFEN(t, chess.StartingFEN)); len(moves) == 0 {
		t.Errorf("a %d-entry book has no move in the starting position", opening.Size())
	}
}

func boardFromFEN(t *testing.T, fen string) *chess.SearchBoard {
	t.Helper()
	game, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}
	return chess.NewSearchBoard(game)
}
