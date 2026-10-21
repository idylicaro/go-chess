package chess

// Precomputed per-square tables for the generator and attack detection, so
// the hot loops walk short slices of square indexes instead of re-checking
// board edges for every offset (#12).

// Ray directions: the first four are orthogonal (rook), the last four
// diagonal (bishop); a queen uses all eight.
const (
	firstDiagonalRay = 4
	rayCount         = 8
)

type squareTables struct {
	knight [64][]int8
	king   [64][]int8
	rays   [64][rayCount][]int8 // squares along each direction, nearest first
}

var moveTables = buildSquareTables()

// squarePositions lets a square index stand in for a *Position without
// allocating: Game.enPassantTarget points into it and is never mutated.
var squarePositions = buildSquarePositions()

func buildSquareTables() *squareTables {
	tables := &squareTables{}
	directions := append(append([]Position{}, orthogonalDirections...), diagonalDirections...)
	for square := 0; square < 64; square++ {
		tables.knight[square] = stepTargets(square, knightOffsets)
		tables.king[square] = stepTargets(square, kingOffsets)
		for ray, direction := range directions {
			tables.rays[square][ray] = rayTargets(square, direction)
		}
	}
	return tables
}

// stepTargets lists the on-board squares one offset away from square.
func stepTargets(square int, offsets []Position) []int8 {
	var targets []int8
	for _, offset := range offsets {
		x, y := square%8+offset.X, square/8+offset.Y
		if onBoard(x, y) {
			targets = append(targets, int8(squareIndex(x, y)))
		}
	}
	return targets
}

// rayTargets lists the squares from square (exclusive) to the board edge.
func rayTargets(square int, direction Position) []int8 {
	var targets []int8
	x, y := square%8+direction.X, square/8+direction.Y
	for onBoard(x, y) {
		targets = append(targets, int8(squareIndex(x, y)))
		x, y = x+direction.X, y+direction.Y
	}
	return targets
}

func buildSquarePositions() [64]Position {
	var positions [64]Position
	for square := range positions {
		positions[square] = Position{X: square % 8, Y: square / 8}
	}
	return positions
}

// orthogonalRays and diagonalRays split a square's rays by slider type.
func orthogonalRays(square int) [][]int8 {
	return moveTables.rays[square][:firstDiagonalRay]
}

func diagonalRays(square int) [][]int8 {
	return moveTables.rays[square][firstDiagonalRay:]
}
