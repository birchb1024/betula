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
var noValues = map[rune]bool{' ': true, '.': true}
var zeroValues = map[rune]bool{' ': true, '0': true, '.': true}

func nonValue(r rune) bool {
	if _, ok := noValues[r]; ok {
		return true
	}
	return false
}
func evalConstant(b board, x int, y int) {
	boardMutex.Lock()
	defer boardMutex.Unlock()
	b[x+1][y] = b[x-1][y]
	b[x][y] = '*'
}
func checkConstant(b board, x int, y int) {
	boardMutex.Lock()
	defer boardMutex.Unlock()
	if b[x-1][y+0] != b[x+1][y+0] {
		b[x][y] = '?'
		errorMessage(b, fmt.Sprintf("'*' short circuit at %d %d: '%c' != '%c'", x, y, b[x-1][y+0], b[x+1][y+0]))
		return
	}
}

func evalLeftRightWire(b board, x int, y int) {
	boardMutex.Lock()
	defer boardMutex.Unlock()
	b[x][y] = '-'
	if nonValue(b[x-1][y+0]) && nonValue(b[x+1][y+0]) {
		return
	}
	if nonValue(b[x-1][y+0]) {
		b[x-1][y+0] = b[x+1][y+0]
		return
	}
	b[x+1][y+0] = b[x-1][y+0]
}
func checkLeftRightWire(b board, x int, y int) {
	boardMutex.Lock()
	defer boardMutex.Unlock()
	if b[x-1][y+0] != b[x+1][y+0] {
		b[x][y] = '?'
		errorMessage(b, fmt.Sprintf("'-' short circuit at %d %d: '%c' != '%c'", x, y, b[x-1][y+0], b[x+1][y+0]))
		return
	}
}

func evalUpDownWire(b board, x int, y int) {
	boardMutex.Lock()
	defer boardMutex.Unlock()
	b[x][y] = '|'
	if nonValue(b[x][y-1]) && nonValue(b[x][y+1]) {
		return
	}
	if nonValue(b[x][y-1]) {
		b[x][y-1] = b[x][y+1]
		return
	}
	b[x][y+1] = b[x][y-1]
}
func checkUpDownWire(b board, x int, y int) {
	boardMutex.Lock()
	defer boardMutex.Unlock()
	if len(b) != width || len(b[x]) != height {
		_, _ = fmt.Fprintf(os.Stderr, "checkUpDownWire %d %d\n", len(b), y)
	}
	errorMessage(b, fmt.Sprintf("checkUpDownWire %d %d", x, y))
	if b[x][y-1] != b[x][y+1] {
		b[x][y] = '?'
		errorMessage(b, fmt.Sprintf("'|' short circuit at %d %d: '%c' != '%c'", x, y, b[x][y-1], b[x][y+1]))
		return
	}
}

func evalRelay(b board, x int, y int) {
	boardMutex.Lock()
	defer boardMutex.Unlock()
	in := b[x-1][y]
	b[x][y] = 'R'
	if in == '0' || in == ' ' || in == '.' {
		// Relay is OFF
		b[x+1][y+1] = b[x-1][y+1]
		b[x+1][y+3] = b[x-1][y+2]
		b[x+1][y+0] = '.'
		b[x+1][y+2] = '.'
		return
	}
	// Relay is ON
	b[x+1][y+0] = b[x-1][y+1]
	b[x+1][y+2] = b[x-1][y+2]
	b[x+1][y+1] = '.'
	b[x+1][y+3] = '.'
}
func checkRelay(b board, x int, y int) bool {
	in := b[x-1][y]
	if _, ok := zeroValues[in]; ok {
		// Relay is OFF
		if b[x+1][y+1] == b[x-1][y+1] &&
			b[x+1][y+3] == b[x-1][y+2] {
			return true // OK
		}
		b[x][y] = '?'
		errorMessage(b, fmt.Sprintf("Relay constraint failure at %d %d", x, y))
		return false
	}
	// Relay is ON
	if b[x+1][y+0] == b[x-1][y+1] &&
		b[x+1][y+2] == b[x-1][y+2] {
		return true // OK
	}
	b[x][y] = '?'
	errorMessage(b, fmt.Sprintf("Relay constraint failure at %d %d", x, y))
	return false
}

func errorMessage(b board, msg string) {
	runes := []rune(msg)
	for i, r := range runes {
		b[i][height-1] = r
	}
}
func interpreter(b board) {
	for {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				switch b[x][y] {
				case '*':
					evalConstant(b, x, y)
				case 'R':
					evalRelay(b, x, y)
				case '-':
					evalLeftRightWire(b, x, y)
				case '|':
					evalUpDownWire(b, x, y)
				default:
				}
			}
		}

		for y := height - 1; y > 0; y-- {
			for x := width - 1; x > 0; x-- {
				switch b[x][y] {
				case '*':
					evalConstant(b, x, y)
				case 'R':
					evalRelay(b, x, y)
				case '-':
					evalLeftRightWire(b, x, y)
				case '|':
					evalUpDownWire(b, x, y)
				default:
				}
			}
		}

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				switch b[x][y] {
				case '*':
					checkConstant(b, x, y)
				case 'R':
					// checkRelay(b, x, y)
				case '-':
					checkLeftRightWire(b, x, y)
				case '|':
					checkUpDownWire(b, x, y)
				default:
				}
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func render(s tcell.Screen, b board) {
	for {
		boardMutex.Lock()
		setRightSideMsg(b, fmt.Sprintf("%3d %3d", cursorX, cursorY))
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
	defer fd.Close()
	if err != nil{
		return 0,0,err
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
			return 0,0,err
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
	defer fd.Close()
	if err != nil{
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
			setRightSideMsg(b, filename)
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

func main() {
	var theBoard board
	filename := "untitled.betula"

	if len(os.Args) > 1 {
		filename = os.Args[1]
		var err error
		width, height, err = sizeOfFile(filename)
		if err != nil {
			log.Fatalf("ERROR: file %s - %s\n", os.Args[1], err)
		}
		theBoard, err = loadFile(filename, width, height)
		if err != nil {
			log.Fatalf("ERROR: file %s - %s\n", os.Args[1], err)
		}

	}

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

	cursorX = screenWidth / 2
	cursorY = screenHeight / 2

	if filename == "untitled.betula" {
		width = screenWidth
		height = screenHeight
		// TODO constructor DRY
		theBoard = make([][]rune, width)
		for x := range theBoard {
			theBoard[x] = make([]rune, height)
		}

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				theBoard[x][y] = ' '
			}
		}
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
			//fmt.Printf("\r%#v %#v\n",ev.Key(), ev.Rune())
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
				theBoard[cursorX][cursorY] = '.'
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
			case tcell.KeyRune:
				boardMutex.Lock()
				switch ev.Rune() {
				case '*':
					//       3*.
					//
					theBoard[cursorX][cursorY] = '*'
					theBoard[cursorX+1][cursorY] = '.'
					theBoard[cursorX-1][cursorY] = '.'
				case '-':
					//       .-.
					//
					theBoard[cursorX][cursorY] = '-'
					theBoard[cursorX+1][cursorY] = '.'
					theBoard[cursorX-1][cursorY] = '.'
				case '|':
					//       .
					//       |
					//       .
					theBoard[cursorX][cursorY] = '|'
					theBoard[cursorX][cursorY-1] = '.'
					theBoard[cursorX][cursorY+1] = '.'
				case 'R':
					//       .R.
					//		 . .
					//		 . .
					//		   .
					//
					theBoard[cursorX][cursorY] = 'R'
					for i := 0; i < 3; i++ {
						theBoard[cursorX-1][cursorY+i] = '.'
					}
					for i := 0; i < 4; i++ {
						theBoard[cursorX+1][cursorY+i] = '.'
					}
				default:
					theBoard[cursorX][cursorY] = ev.Rune()
				}
				boardMutex.Unlock()
			default:
			}
		case *tcell.EventMouse:

		}
	}
}

func setRightSideMsg(b board, msg string) {
	runes := []rune(msg)
	for i, r := range runes {
		b[width-1-len(runes)+i][height-1] = r
	}
}

func view(s tcell.Screen, b board) {
	boardMutex.Lock()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			s.SetContent(x, y, b[x][y], nil, tcell.StyleDefault)
			//fmt.Printf("%3d ", m[x][y])
		}
		//fmt.Printf("\n")
	}

	//fmt.Printf("\n")
	s.SetContent(cursorX, cursorY, b[cursorX][cursorY], nil, tcell.StyleDefault.Reverse(true))
	boardMutex.Unlock()
}
