package main

import (
	"bufio"
	"fmt"
	"github.com/gdamore/tcell"
	"io"
	"sync"
	"time"

	"log"
	"os"
	//	"strconv"
)

type board [][]rune

var width, height int
var cursorX int
var cursorY int

var boardMutex sync.RWMutex
var noValues = map[rune]bool{' ': true, '.': true, 0: true}
var zeroValues = map[rune]bool{' ': true, '0': true, '.': true, 0: true}

func nonValue(r rune) bool {
	if _, ok := noValues[r]; ok {
		return true
	}
	return false
}
func isZero(r rune) bool {
	if _, ok := zeroValues[r]; ok {
		return true
	}
	return false
}
func isDigit(r rune) bool {
	if r >= '0' && r <= '9' {
		return true
	}
	return false
}
func cond(control, yes, no rune) rune {
	if isZero(control) {
		return no
	}
	return yes
}

func checkConstant(b board, x int, y int) {
	if b[x-1][y+0] != b[x+1][y+0] {
		b[x][y] = '?'
		setMiddleMsg(b, fmt.Sprintf("'*' short circuit at %d %d: '%c' != '%c'", x, y, b[x-1][y+0], b[x+1][y+0]))
		return
	}
}

func checkLeftRightWire(b board, x int, y int) {
	if b[x-1][y] != b[x+1][y] {
		b[x][y] = '?'
		setMiddleMsg(b, fmt.Sprintf("'>' short circuit at %d %d: '%c' != '%c'", x, y, b[x-1][y+0], b[x+1][y+0]))
		return
	}
}

func checkUpDownWire(b board, x int, y int) {
	if len(b) != width || len(b[x]) != height {
		_, _ = fmt.Fprintf(os.Stderr, "checkUpDownWire %d %d\n", len(b), y)
	}
	setMiddleMsg(b, fmt.Sprintf("checkUpDownWire %d %d", x, y))
	if b[x][y-1] != b[x][y+1] {
		b[x][y] = '?'
		setMiddleMsg(b, fmt.Sprintf("'|' short circuit at %d %d: '%c' != '%c'", x, y, b[x][y-1], b[x][y+1]))
		return
	}
}

type coord struct{ x, y int }

func propogate(visited board, b board, p coord, value rune) {

	if p.x >= len(b) || p.y >= len(b[0]) {
		return
	}
	if len(visited) > 1 && nonValue(b[p.x][p.y]) {
		return
	}

	switch b[p.x][p.y] {
	case '*':
		//               .
		//              3*.
		//               .
		if visited[p.x][p.y] == 'Y' || visited[p.x-1][p.y] == 'Y' {
			if value != b[p.x-1][p.y] {
				setMiddleMsg(b, fmt.Sprintf("'*' short circuit at %d %d: '%c' != '%c'", p.x, p.y, b[p.x-1][p.y], value))
			}
			return
		}
		visited[p.x][p.y] = 'Y'
		visited[p.x-1][p.y] = 'Y'
		constant := b[p.x-1][p.y]
		propogate(visited, b, coord{p.x, p.y + 1}, constant)
		propogate(visited, b, coord{p.x + 1, p.y}, constant)
		propogate(visited, b, coord{p.x, p.y - 1}, constant)

	case 'C':
		modulo := 2
		fraction := 10
		if isDigit(b[p.x-1][p.y]) {
			modulo = int(b[p.x-1][p.y] - '0')
			if modulo == 0 {
				modulo = 10
			}
			if isDigit(b[p.x-2][p.y]) {
				fraction = int(b[p.x-2][p.y] - '0')
				if fraction == 0 {
					fraction = 10
				}
			}
		}
		divisor := 100 * fraction
		milliSeconds := time.Now().Second()*1000 + time.Now().Nanosecond()/1000000
		clock := '0' + rune((milliSeconds/divisor)%modulo)
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		visited[p.x-1][p.y] = 'Y'
		visited[p.x-2][p.y] = 'Y'
		propogate(visited, b, coord{p.x, p.y - 1}, clock)
		propogate(visited, b, coord{p.x, p.y + 1}, clock)
		propogate(visited, b, coord{p.x - 1, p.y}, clock)
		propogate(visited, b, coord{p.x + 1, p.y}, clock)
	case '-':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		propogate(visited, b, coord{p.x + 1, p.y}, value)
		propogate(visited, b, coord{p.x - 1, p.y}, value)
	case '>':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		propogate(visited, b, coord{p.x + 1, p.y}, value)
	case 'v':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		propogate(visited, b, coord{p.x, p.y + 1}, value)
	case '|':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		propogate(visited, b, coord{p.x, p.y - 1}, value)
		propogate(visited, b, coord{p.x, p.y + 1}, value)
	case 'N':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		inverted := cond(value, '0', '1')
		propogate(visited, b, coord{p.x, p.y - 1}, inverted)
		propogate(visited, b, coord{p.x, p.y + 1}, inverted)
		propogate(visited, b, coord{p.x + 1, p.y}, inverted)
		propogate(visited, b, coord{p.x - 1, p.y}, inverted)
	case 'S':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		if isZero(b[p.x][p.y-1]) {
			return
		}
		propogate(visited, b, coord{p.x - 1, p.y}, value)
		propogate(visited, b, coord{p.x + 1, p.y}, value)
	case '/':
		var end int
		for end = p.x + 1; end < width; end++ {
			if b[end][p.y] == '\\' {
				break
			}
		}
		if end == 0 {
			return
		}
		if visited[p.x][p.y] == 'Y' || visited[end][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		visited[end][p.y] = 'Y'
		propogate(visited, b, coord{end + 1, p.y}, value)
		propogate(visited, b, coord{p.x - 1, p.y}, value)
	case '\\':
		var begin int
		for begin = p.x - 1; begin > 0; begin-- {
			if b[begin][p.y] == '/' {
				break
			}
		}
		if begin == 0 {
			return
		}
		if visited[p.x][p.y] == 'Y' || visited[begin][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		visited[begin][p.y] = 'Y'
		propogate(visited, b, coord{begin - 1, p.y}, value)
		propogate(visited, b, coord{p.x - 1, p.y}, value)
	case '$':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		propogate(visited, b, coord{p.x, p.y - 1}, value)
		propogate(visited, b, coord{p.x, p.y + 1}, value)
		propogate(visited, b, coord{p.x + 1, p.y}, value)
		propogate(visited, b, coord{p.x - 1, p.y}, value)
	case 'L':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		visited[p.x][p.y-1] = 'Y'
		b[p.x][p.y-1] = value
		propogate(visited, b, coord{p.x + 1, p.y}, value)
		propogate(visited, b, coord{p.x - 1, p.y}, value)
	case 'J':
		if visited[p.x][p.y] == 'Y' {
			return
		}
		visited[p.x][p.y] = 'Y'
		visited[p.x][p.y+1] = 'Y'
		b[p.x][p.y+1] = value
		propogate(visited, b, coord{p.x + 1, p.y}, value)
		propogate(visited, b, coord{p.x - 1, p.y}, value)
	default:
	}
}

func interpreter(b board) {
	var visited board
	for {
		boardMutex.Lock()
		roots := make([]coord, 0)
		visited = makeBoard(len(b), len(b[0]))
		// Find roots
		for y := 0; y < height-1; y++ {
			for x := 0; x < width; x++ {
				switch b[x][y] {
				case 'L':
					b[x][y-1] = ' '
				case '*':
					roots = append(roots, coord{x, y})
				case 'C':
					roots = append(roots, coord{x, y})
				default:
				}
			}
		}

		for _, p := range roots {
			propogate(visited, b, p, ' ')
		}

		// Clear numeric values not reachable from roots
		for y := 0; y < height-1; y++ {
			for x := 0; x < width; x++ {
				if visited[x][y] != 'Y' {
					if b[x][y] >= '0' && b[x][y] <= '9' {
						b[x][y] = ' '
					}

				}
			}
		}
		boardMutex.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
}

func render(s tcell.Screen, b board) {
	for {
		boardMutex.Lock()
		setLeftMsg(b, fmt.Sprintf("%3d %3d", cursorX, cursorY))
		boardMutex.Unlock()
		view(s, b)
		s.Show()
		time.Sleep(100 * time.Millisecond)
	}
}
func maxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func sizeOfFile(filename string) (int, int, error) {
	fd, err := os.Open(filename)
	defer func(fd *os.File) { _ = fd.Close() }(fd)
	if err != nil {
		return 0, 0, err
	}
	rdr := bufio.NewReader(fd)
	height := 1
	width := 0
	x := 0

	for {
		r, _, err := rdr.ReadRune()
		if err == io.EOF {
			return width, height, nil
		}
		if err != nil {
			return 0, 0, err
		}
		if r == '\n' {
			width = maxInt(width, x)
			x = 0
			height += 1
			continue
		}
		x += 1
	}
}
func loadFile(filename string, width int, height int) (board, error) {
	fd, err := os.Open(filename)
	defer func(fd *os.File) { _ = fd.Close() }(fd)
	if err != nil {
		return nil, err
	}
	rdr := bufio.NewReader(fd)
	y := 0
	x := 0
	var b board = make([][]rune, width)
	for x := range b {
		b[x] = make([]rune, height)
	}

	for {
		r, _, err := rdr.ReadRune()
		if err == io.EOF {
			setMiddleMsg(b, fmt.Sprintf("Loaded %s, into width %d, height %d", filename, width, height))
			return b, nil
		}
		if err != nil {
			return nil, err
		}
		if r == '\n' {
			x = 0
			y += 1
			continue
		}
		b[x][y] = r
		x += 1
	}
}

func saveFile(b board, filename string) {

	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		setMiddleMsg(b, err.Error())
		return
	}
	defer func(fd *os.File) { _ = fd.Close() }(fd)

	var actualWidth = 0
	var actualHeight = 0
	for y := 0; y < len(b[0])-1; y++ {
		var maxX = 0
		for x := 0; x < len(b); x++ {
			if !isZero(b[x][y]) {
				maxX = x
			}
		}
		if maxX != 0 {
			actualHeight = y
		}
		actualWidth = maxInt(actualWidth, maxX)
	}

	for y := 0; y <= actualHeight; y++ {
		for x := 0; x <= actualWidth; x++ {
			r := b[x][y]
			if r == 0 {
				r = ' '
			}
			_, err := fmt.Fprintf(fd, "%c", r)
			if err != nil {
				setMiddleMsg(b, err.Error())
				return
			}
		}
		_, err = fmt.Fprintf(fd, "\n")
		if err != nil {
			setMiddleMsg(b, err.Error())
			return
		}
	}
	setMiddleMsg(b, fmt.Sprintf("Saved %s, width %d, height %d", filename, actualWidth, actualHeight))

}

func makeBoard(width int, height int) board {

	b := make([][]rune, width)
	for x := range b {
		b[x] = make([]rune, height)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			b[x][y] = ' '
		}
	}
	return b
}

func (b board) setIfEmpty(x int, y int, r rune) {
	if nonValue(b[x][y]) {
		b[x][y] = r
	}
}
func main() {
	var theBoard board
	filename := "untitled.betula"

	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}

	// Set default text style
	defStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	s.SetStyle(defStyle)

	// Clear screen
	s.Clear()

	screenWidth, screenHeight := s.Size()
	_, _ = fmt.Fprintf(os.Stderr, "screenWidth %d, screenHeight %d\n", screenWidth, screenHeight)
	cursorX = screenWidth / 2
	cursorY = screenHeight / 2

	if len(os.Args) > 1 {
		filename = os.Args[1]
		var err error
		fileWidth, fileHeight, err := sizeOfFile(filename)
		if err != nil {
			log.Fatalf("ERROR: file %s - %s\n", os.Args[1], err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "fileWidth %d, fileHeight %d\n", fileWidth, fileHeight)
		width = maxInt(fileWidth, screenWidth)
		height = maxInt(fileHeight, screenHeight)
		theBoard, err = loadFile(filename, width, height)
		if err != nil {
			log.Fatalf("ERROR: file %s - %s\n", os.Args[1], err)
		}
	} else {
		width = screenWidth
		height = screenHeight
		theBoard = makeBoard(width, height)
	}
	quit := func() {
		s.Fini()
		s.EnableMouse()
		os.Exit(0)
	}
	go interpreter(theBoard)
	go render(s, theBoard)

	for {
		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				quit()
			}
			switch ev.Key() {

			case tcell.KeyF5:
				boardMutex.Lock()
				r := theBoard[cursorX][cursorY]
				if nonValue(r) || r == '0' {
					theBoard[cursorX][cursorY] = '1'
				} else {
					theBoard[cursorX][cursorY] = '0'
				}
				boardMutex.Unlock()
			case tcell.KeyDelete:
				boardMutex.Lock()
				theBoard[cursorX][cursorY] = ' '
				boardMutex.Unlock()
			case tcell.KeyUp:
				if cursorY != 0 {
					cursorY -= 1
				}
			case tcell.KeyDown:
				if cursorY < height-2 {
					cursorY += 1
				}
			case tcell.KeyLeft:
				if cursorX != 0 {
					cursorX -= 1
				}
			case tcell.KeyRight:
				if cursorX < width-1 {
					cursorX += 1
				}
			case tcell.KeyCtrlS:
				boardMutex.Lock()
				saveFile(theBoard, filename)
				boardMutex.Unlock()
			case tcell.KeyRune:
				boardMutex.Lock()
				theBoard[cursorX][cursorY] = ev.Rune()
				boardMutex.Unlock()
			default:
			}
		case *tcell.EventMouse:

		}
	}
}

var colors = map[rune]tcell.Color{
	'-':  tcell.ColorLightBlue,
	'|':  tcell.ColorLightBlue,
	'/':  tcell.ColorLightBlue,
	'\\': tcell.ColorLightBlue,
	'$':  tcell.ColorBlue,
	'?':  tcell.ColorRed,

	'L': tcell.ColorBlack,
	'J': tcell.ColorBlack,
	'N': tcell.ColorBlue,
	'*': tcell.ColorBlack,
	'C': tcell.ColorDarkBlue,
	'S': tcell.ColorBlack,

	'0': tcell.ColorRed,
	'1': tcell.ColorOrange,
	'2': tcell.ColorOrange,
	'3': tcell.ColorOrange,
	'4': tcell.ColorOrange,
	'5': tcell.ColorOrange,
	'6': tcell.ColorOrange,
	'7': tcell.ColorOrange,
	'8': tcell.ColorOrange,
	'9': tcell.ColorOrange,
}
var backgrounds = map[rune]tcell.Color{
	'$': tcell.ColorLightBlue,
	'J': tcell.ColorLightBlue,
	'L': tcell.ColorLightBlue,
	'N': tcell.ColorLightPink,
	'S': tcell.ColorLightPink,
	'C': tcell.ColorLightGreen,
	'*': tcell.ColorLightGreen,
}

func styleOf(r rune) tcell.Style {
	var s = tcell.StyleDefault
	if c, ok := colors[r]; ok {
		s = s.Foreground(c)
	}
	if c, ok := backgrounds[r]; ok {
		s = s.Background(c)
	}
	if r == '*' {
		s = s.Bold(true)
	}
	return s

}

func setMiddleMsg(b board, msg string) {
	runes := []rune(msg)
	for i, r := range runes {
		b[i+20][height-1] = r
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s\n", msg)
}

func setLeftMsg(b board, msg string) {
	runes := []rune(msg)
	for i, r := range runes {
		b[i][height-1] = r
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s\n", msg)
}

func view(s tcell.Screen, b board) {
	boardMutex.Lock()

	for y := 0; y < height-1; y++ {
		for x := 0; x < width; x++ {
			s.SetContent(x, y, b[x][y], nil, styleOf(b[x][y]))
		}
	}
	for x := 0; x < width; x++ {
		s.SetContent(x, height-1, b[x][height-1], nil, tcell.StyleDefault)
	}
	s.SetContent(cursorX, cursorY, b[cursorX][cursorY], nil, tcell.StyleDefault.Reverse(true))
	boardMutex.Unlock()
}
