// Package book reads opening books in the Polyglot format, so the engine
// can play known opening moves without searching (lab #46).
package book

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// entryBytes is the size of one book entry: key, move, weight and learn.
const entryBytes = 16

// entry is one book record; the file is sorted by key.
type entry struct {
	key    uint64
	move   uint16
	weight uint16
}

// Book is an opening book loaded in memory.
type Book struct {
	entries []entry
}

// Move is a book move in UCI notation with its weight (how often or how
// well it was played; higher is better).
type Move struct {
	UCI    string
	Weight int
}

// Open reads the Polyglot book at path.
//
//	opening, err := book.Open("books/gm2001.bin")
func Open(path string) (*Book, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse reads a book from its bytes.
func Parse(data []byte) (*Book, error) {
	if len(data)%entryBytes != 0 {
		return nil, fmt.Errorf("book of %d bytes: expected a multiple of %d (Polyglot entries)", len(data), entryBytes)
	}
	entries := make([]entry, len(data)/entryBytes)
	for i := range entries {
		record := data[i*entryBytes:]
		entries[i] = entry{key: binary.BigEndian.Uint64(record), move: binary.BigEndian.Uint16(record[8:]), weight: binary.BigEndian.Uint16(record[10:])}
	}
	if !sort.SliceIsSorted(entries, func(i, j int) bool { return entries[i].key < entries[j].key }) {
		return nil, fmt.Errorf("book entries are not sorted by key, as the Polyglot format requires")
	}
	return &Book{entries: entries}, nil
}

// Size is the number of entries.
func (b *Book) Size() int {
	return len(b.entries)
}

// Moves returns the book's legal moves for board, heaviest first.
func (b *Book) Moves(board *chess.SearchBoard) []Move {
	key := Key(board)
	first := sort.Search(len(b.entries), func(i int) bool { return b.entries[i].key >= key })
	var moves []Move
	for i := first; i < len(b.entries) && b.entries[i].key == key; i++ {
		notation := decodeMove(board, b.entries[i].move)
		if b.entries[i].weight > 0 && isLegal(board, notation) {
			moves = append(moves, Move{UCI: notation, Weight: int(b.entries[i].weight)})
		}
	}
	sort.SliceStable(moves, func(i, j int) bool { return moves[i].Weight > moves[j].Weight })
	return moves
}

// Pick chooses a book move at random in proportion to the weights;
// randomInt(n) must return a number in [0, n). ok is false out of book.
//
//	move, ok := opening.Pick(board, rand.IntN)
func (b *Book) Pick(board *chess.SearchBoard, randomInt func(n int) int) (string, bool) {
	moves := b.Moves(board)
	total := 0
	for _, move := range moves {
		total += move.Weight
	}
	if total == 0 {
		return "", false
	}
	roll := randomInt(total)
	for _, move := range moves {
		if roll < move.Weight {
			return move.UCI, true
		}
		roll -= move.Weight
	}
	return moves[len(moves)-1].UCI, true
}

func isLegal(board *chess.SearchBoard, notation string) bool {
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	for _, move := range board.GenerateMoves(buffer[:0]) {
		if move.String() != notation {
			continue
		}
		undo, legal := board.MakeMove(move)
		if legal {
			board.UnmakeMove(move, undo)
		}
		return legal
	}
	return false
}
