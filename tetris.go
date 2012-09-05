package main

import (
	"fmt"
	"github.com/nsf/termbox-go"
	"math/rand"
	"time"
)

const (
	width      = 10
	height     = 18
	initialX   = 4
	piecesFile = "./pieces.txt"
)

type Vector struct {
	x, y int
}

func (first Vector) plus(second Vector) Vector {
	return Vector{first.x + second.x, first.y + second.y}
}
func (first Vector) equals(second Vector) bool {
	return first.x == second.x && first.y == second.y
}

// A particular rotational instance of a piece.
type PieceInstance []Vector

type Piece struct {
	rotations       []PieceInstance
	currentRotation int
	color           termbox.Attribute
}

func (p *Piece) instance() PieceInstance {
	return p.rotations[p.currentRotation]
}

func (p *Piece) rotate() {
	p.currentRotation = (p.currentRotation + 1) % len(p.rotations)
}

func (p *Piece) unrotate() {
	p.currentRotation = (p.currentRotation - 1) % len(p.rotations)
	if p.currentRotation < 0 {
		p.currentRotation += len(p.rotations)
	}
}

func TetrisPieces() []Piece {
	return []Piece{Piece{[]PieceInstance{[]Vector{Vector{0, 0}, Vector{1, 0}, Vector{0, 1}, Vector{1, 1}}},
		0, termbox.ColorYellow},
		Piece{[]PieceInstance{[]Vector{Vector{0, 0}, Vector{1, 0}, Vector{1, 1}, Vector{2, 1}},
			[]Vector{Vector{1, 0}, Vector{0, 1}, Vector{1, 1}, Vector{0, 2}},
		}, 0, termbox.ColorRed},
		Piece{[]PieceInstance{[]Vector{Vector{1, 0}, Vector{2, 0}, Vector{0, 1}, Vector{1, 1}},
			[]Vector{Vector{0, 0}, Vector{0, 1}, Vector{1, 1}, Vector{1, 2}},
		}, 0, termbox.ColorGreen},
		Piece{[]PieceInstance{[]Vector{Vector{1, 0}, Vector{0, 1}, Vector{1, 1}, Vector{2, 1}},
			[]Vector{Vector{0, 0}, Vector{0, 1}, Vector{1, 1}, Vector{0, 2}},
			[]Vector{Vector{0, 0}, Vector{1, 0}, Vector{2, 0}, Vector{1, 1}},
			[]Vector{Vector{1, 0}, Vector{0, 1}, Vector{1, 1}, Vector{1, 2}},
		}, 0, termbox.ColorMagenta},
		Piece{[]PieceInstance{[]Vector{Vector{1, 0}, Vector{1, 1}, Vector{1, 2}, Vector{2, 2}},
			[]Vector{Vector{0, 1}, Vector{1, 1}, Vector{2, 1}, Vector{0, 2}},
			[]Vector{Vector{0, 0}, Vector{1, 0}, Vector{1, 1}, Vector{1, 2}},
			[]Vector{Vector{2, 0}, Vector{0, 1}, Vector{1, 1}, Vector{2, 1}},
		}, 0, termbox.ColorWhite},
		Piece{[]PieceInstance{[]Vector{Vector{1, 0}, Vector{1, 1}, Vector{1, 2}, Vector{0, 2}},
			[]Vector{Vector{0, 1}, Vector{1, 1}, Vector{2, 1}, Vector{0, 0}},
			[]Vector{Vector{1, 0}, Vector{2, 0}, Vector{1, 1}, Vector{1, 2}},
			[]Vector{Vector{0, 1}, Vector{1, 1}, Vector{2, 1}, Vector{2, 2}},
		}, 0, termbox.ColorBlue},
		Piece{[]PieceInstance{[]Vector{Vector{1, 0}, Vector{1, 1}, Vector{1, 2}, Vector{1, 3}},
			[]Vector{Vector{0, 1}, Vector{1, 1}, Vector{2, 1}, Vector{3, 1}},
		}, 0, termbox.ColorCyan},
	}
}

// A map from a point on a board to the color of that cell.
type ColorMap map[Vector]termbox.Attribute

func (cm ColorMap) contains(v Vector) bool {
	_, ok := cm[v]
	return ok
}

type Direction int

const (
	Up Direction = iota + 1
	Down
	Left
	Right
)

type Board struct {
	cells           ColorMap
	currentPiece    *Piece
	currentPosition Vector
}

func NewBoard() *Board {
	board := new(Board)
	board.cells = make(ColorMap)
	return board
}

type Game struct {
	board           *Board
	nextPiece       *Piece
	pieces          []Piece
	over            bool
	dropDelayMillis int
	ticker          *time.Ticker
}

func NewGame() *Game {
	game := new(Game)
	game.pieces = TetrisPieces()
	game.board = NewBoard()
	game.board.currentPiece = game.GeneratePiece()
	game.board.currentPosition = Vector{initialX, 0}
	game.nextPiece = game.GeneratePiece()
	game.over = false
	// Start off the delay at 3/4 of a second.
	game.dropDelayMillis = 750
	game.startTicker()
	return game
}

func (game *Game) startTicker() {
	game.ticker = time.NewTicker(time.Duration(game.dropDelayMillis) * time.Millisecond)
}

func (game *Game) stopTicker() {
	game.ticker.Stop()
}

type GameEvent int

const (
	MoveLeft GameEvent = iota
	MoveRight
	MoveDown
	Rotate
	QuickDrop
	Quit
	// An event that doesn't cause a change to game state but causes a full redraw; e.g., a window resize.
	Redraw
)

func (game *Game) Start() {
	game.Draw(true)

	eventQueue := make(chan GameEvent, 100)
	go func() {
		for {
			eventQueue <- waitForUserEvent()
		}
	}()
gameLoop:
	for {
		fullRedraw := false
		var event GameEvent
		select {
		case event = <-eventQueue:
		case <-game.ticker.C:
			event = MoveDown
		}
		switch event {
		case MoveLeft:
			game.Move(Left)
		case MoveRight:
			game.Move(Right)
		case MoveDown:
			game.Move(Down)
		case QuickDrop:
			game.QuickDrop()
		case Rotate:
			game.Rotate()
		case Quit:
			break gameLoop
		case Redraw:
			fullRedraw = true
		}
		if game.over {
			break gameLoop
		}
		game.Draw(fullRedraw)
	}
}

func waitForTick(ticker *time.Ticker) GameEvent {
	<-ticker.C
	return MoveDown
}

func waitForUserEvent() GameEvent {
	switch event := termbox.PollEvent(); event.Type {
	// Movement: arrow keys or vim controls (h, j, k, l)
	// Exit: 'q' or ctrl-c.
	case termbox.EventKey:
		if event.Ch == 0 { // A special key combo was pressed
			switch event.Key {
			case termbox.KeyCtrlC:
				return Quit
			case termbox.KeyArrowLeft:
				return MoveLeft
			case termbox.KeyArrowUp:
				return Rotate
			case termbox.KeyArrowRight:
				return MoveRight
			case termbox.KeyArrowDown:
				return MoveDown
			case termbox.KeySpace:
				return QuickDrop
			}
		} else {
			switch event.Ch {
			case 'q':
				return Quit
			case 'h':
				return MoveLeft
			case 'k':
				return Rotate
			case 'l':
				return MoveRight
			case 'j':
				return MoveDown
			}
		}
	case termbox.EventResize:
		return Redraw
	case termbox.EventError:
		panic(event.Err)
	}
	return Redraw // Should never be reached
}

func (game *Game) GeneratePiece() *Piece {
	return &game.pieces[rand.Intn(len(game.pieces))]
}

func (board *Board) currentPieceInCollision() bool {
	for _, point := range board.currentPiece.instance() {
		attemptedPoint := point.plus(board.currentPosition)
		if attemptedPoint.x < 0 || attemptedPoint.x >= width ||
			attemptedPoint.y < 0 || attemptedPoint.y >= height ||
			board.cells.contains(attemptedPoint) {
			return true
		}
	}
	return false
}

func (board *Board) moveIfPossible(translation Vector) bool {
	position := board.currentPosition
	board.currentPosition = board.currentPosition.plus(translation)
	if board.currentPieceInCollision() {
		board.currentPosition = position
		return false
	}
	return true
}

func (board *Board) mergeCurrentPiece() {
	for _, point := range board.currentPiece.instance() {
		board.cells[point.plus(board.currentPosition)] = board.currentPiece.color
	}
}

// Check whether a horizontal row is complete.
func (board *Board) rowComplete(y int) bool {
	for x := 0; x < width; x++ {
		if !board.cells.contains(Vector{x, y}) {
			return false
		}
	}
	return true
}

// Clear a single row and move every above cell down.
func (board *Board) collapseRow(rowY int) {
	for y := rowY - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			if color, ok := board.cells[Vector{x, y}]; ok {
				board.cells[Vector{x, y + 1}] = color
			} else {
				delete(board.cells, Vector{x, y + 1})
			}
		}
	}
	// Clear the top row completely
	for x := 0; x < width; x++ {
		delete(board.cells, Vector{x, 0})
	}
}

// Clear any complete rows and move the above blocks down.
func (board *Board) clearRows() {
	y := height - 1
	for y >= 0 {
		for board.rowComplete(y) {
			board.collapseRow(y)
		}
		y -= 1
	}
}

// Anchor the current piece to the board, clears lines, and generate a new piece. Sets the 'game over' state
// if the new piece overlaps existing pieces.
func (game *Game) anchor() {
	game.board.mergeCurrentPiece()
	game.board.clearRows()

	game.board.currentPiece = game.nextPiece
	game.board.currentPosition = Vector{initialX, 0}
	game.nextPiece = game.GeneratePiece()
	game.nextPiece.currentRotation = 0

	if game.board.currentPieceInCollision() {
		game.over = true
	}
}

// Attempt to move.
func (game *Game) Move(where Direction) {
	translation := Vector{0, 0}
	switch where {
	case Down:
		translation = Vector{0, 1}
	case Left:
		translation = Vector{-1, 0}
	case Right:
		translation = Vector{1, 0}
	}
	// Attempt to make the move.
	moved := game.board.moveIfPossible(translation)

	// Perform anchoring if we tried to move down but we were unsuccessful.
	if where == Down && !moved {
		game.anchor()
	}
}

// Drop the piece all the way and anchor it.
func (game *Game) QuickDrop() {
	// Move down as far as possible
	for game.board.moveIfPossible(Vector{0, 1}) {
	}
	game.anchor()
}

func (game *Game) Rotate() {
	game.board.currentPiece.rotate()
	if game.board.currentPieceInCollision() {
		game.board.currentPiece.unrotate()
	}
}

func (board *Board) CellColor(position Vector) termbox.Attribute {
	if color, ok := board.cells[position]; ok {
		return color
	}
	if board.currentPiece == nil {
		return termbox.ColorDefault
	}
	for _, point := range board.currentPiece.instance() {
		if point.plus(board.currentPosition).equals(position) {
			return board.currentPiece.color
		}
	}
	return termbox.ColorDefault
}

// Print a message in white text.
func printString(x, y int, message string) {
	for i, ch := range message {
		termbox.SetCell(x+i, y, ch, termbox.ColorWhite, termbox.ColorDefault)
	}
}

// Print a message vertically in white text.
func printStringVertical(x, y int, message string) {
	for i, ch := range message {
		termbox.SetCell(x, y+i, ch, termbox.ColorWhite, termbox.ColorDefault)
	}
}

// Print a box-drawing border character.
func printBorderCharacter(x, y int, ch rune) {
	termbox.SetCell(x, y, ch, termbox.ColorBlue, termbox.ColorDefault)
}

/*

  +---------------------------------------+
  |                 header                |
  +-----------------------+---------------+
	|                       |               |
	|                       |   preview     |
	|                       |               |
  |                       |               |
	|        board          +---------------+
	|   (width x height)    |               |
	|                       |               |
	|                       |    score      |
	|                       |               |
	|                       |               |
	+-----------------------+---------------+
	|                                       |
	|             instructions              |
	|                                       |
	+---------------------------------------+

*/
func (game *Game) Draw(fullRedraw bool) {

	headerHeight := 5
	previewHeight := 6
	/*scoreHeight := height - previewHeight - 1*/
	sidebarWidth := 20
	instructionsHeight := 10

	// The internal cells (the board cells) are treated as pairs, so to keep them on even x coordinates we'll
	// put an empty column on the left side.
	totalHeight := headerHeight + height + instructionsHeight + 2
	totalWidth := (width * 2) + sidebarWidth + 1

	// We don't need to redraw the static stuff termbox's buffer every time we move a piece.
	// See http://en.wikipedia.org/wiki/Box-drawing_character for unicode characters.
	if fullRedraw {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		// Print the borders.
		for x := 2; x < totalWidth+2; x++ {
			printBorderCharacter(x, 0, '─')
			printBorderCharacter(x, headerHeight+1, '─')
			printBorderCharacter(x, headerHeight+height+2, '─')
			printBorderCharacter(x, totalHeight+1, '─')
		}
		for x := width+2; x < totalWidth+2; x++ {
			printBorderCharacter(x, headerHeight+previewHeight+2, '─')
		}
		for y := 1; y < totalHeight+1; y++ {
			printBorderCharacter(1, y, '│')
			printBorderCharacter(totalWidth+2, y, '│')
		}
		// Bold borders around the board
		for x := 2; x < (width*2)+2; x++ {
			printBorderCharacter(x, headerHeight+1, '━')
			printBorderCharacter(x, headerHeight+height+2, '━')
		}
		for y := headerHeight+2; y < headerHeight+height+2; y++ {
			printBorderCharacter(1, y, '┃')
			printBorderCharacter((width*2)+2, y, '┃')
		}
		// Print the various corners
		printBorderCharacter(1, 0, '┌')
		printBorderCharacter(totalWidth+2, 0, '┐')
		printBorderCharacter(totalWidth+2, totalHeight+1, '┘')
		printBorderCharacter(1, totalHeight+1, '└')
		printBorderCharacter(1, headerHeight+1, '┢')
		printBorderCharacter((width*2)+2, headerHeight+1, '┱')
		printBorderCharacter(totalWidth+2, headerHeight+1, '┤')
		printBorderCharacter((width*2)+2, headerHeight+previewHeight+2, '┠')
		printBorderCharacter(totalWidth+2, headerHeight+previewHeight+2, '┤')
		printBorderCharacter(1, headerHeight+height+2, '┡')
		printBorderCharacter((width*2)+2, headerHeight+height+2, '┹')
		printBorderCharacter(totalWidth+2, headerHeight+height+2, '┤')

		// Print the header logo
		header := []string{"",
			"   ____         _____    _        _     ",
			"  / ___| ___   |_   _|__| |_ _ __(_)___ ",
			" | |  _ / _ \\    | |/ _ \\ __| '__| / __|",
			" | |_| | (_) |   | |  __/ |_| |  | \\__ \\",
			"  \\____|\\___/    |_|\\___|\\__|_|  |_|___/",
		}
		for i, line := range header {
			printString(2, i, line)
		}

		// Print the "NEXT" text vertically
		printStringVertical((width*2)+5, headerHeight+3, "NEXT")

		// Print the "SCORE" header
		printString((width*2)+10, headerHeight+previewHeight+4, "SCORE")

		// Print instructions below the game board.
		instructions := []string{"Controls:",
			"",
			"Move left       left arrow or 'h'",
			"Move right      right arrow or 'l'",
			"Move down       down arrow or 'j'",
			"Rotate piece    up arrow or 'k'",
			"Quick drop      space",
			"Quit            ctrl-c or 'q'",
		}
		for i, message := range instructions {
			printString(4, headerHeight+height+4+i, message)
		}
	}

	// Print the board contents. Each block will correspond to a side-by-side pair of cells in the termbox, so
	// that the visible blocks will be roughly square.
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			color := game.board.CellColor(Vector{x, y})
			termbox.SetCell(x*2+2, headerHeight+y+2, ' ', termbox.ColorDefault, color)
			termbox.SetCell(x*2+3, headerHeight+y+2, ' ', termbox.ColorDefault, color)
		}
	}

	// Print the preview piece. Need to clear the box first.
	previewPieceOffset := Vector{(width*2)+8, headerHeight+3}
	for x := 0; x < 6; x++ {
		for y := 0; y < 4; y++ {
			cursor := previewPieceOffset.plus(Vector{x, y})
			termbox.SetCell(cursor.x, cursor.y, ' ', termbox.ColorDefault, termbox.ColorDefault)
		}
	}
	for _, point := range game.nextPiece.rotations[0] {
		cursor := previewPieceOffset.plus(Vector{point.x*2, point.y})
		termbox.SetCell(cursor.x, cursor.y, ' ', termbox.ColorDefault, game.nextPiece.color)
		termbox.SetCell(cursor.x+1, cursor.y, ' ', termbox.ColorDefault, game.nextPiece.color)
	}

	// Print the current score

	termbox.Flush()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	err := termbox.Init()
	if err != nil {
		panic(err)
	}

	game := NewGame()
	game.Start()

	termbox.Close()
	fmt.Println("Bye!")
}
