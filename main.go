package main

import (
	"bufio"
	"fmt"
	"github.com/gdamore/tcell"
	"io"
	"math/rand"
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
var noValues = map[rune]bool{' ': true, 0: true}
var zeroValues = map[rune]bool{' ': true, '0': true, 0: true}

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
func cond(control, yes, no rune) rune {
	if isZero(control) {
		return no
	}
	return yes
}

func toBinary(x rune) rune {
	if isZero(x) {
		return '0'
	}
	return '1'
}

type coord struct{ x, y int }

var nowhere = coord{-1, -1}
var clockTicks int

type relay struct {
	vSwitchState coord
	vControl coord
	vLeft coord
	vRight coord
	inControl coord
	inLeft coord
	inRight coord
	defaultState rune
	switchONfn func(rune) bool
}

func (r *relay) propagate(visited board, b board, f coord, p coord, value rune, multi map[coord]int) {
	// ignore if not the three inputs
	if !(f == r.inLeft || f == r.inRight || f == r.inControl || f == nowhere) {
		return
	}
	if visited.yes(p) {
		return
	}
	// if not seen before
	if _, ok:= multi[p] ; !ok {
		multi[p] = 1
		// reset variables
		for _, v := range []coord{r.vLeft, r.vRight, r.vControl} {
			b.setC(v, ' ')
		}
	}
	if f == r.inControl {
		// new control signal
		b.setC(r.vControl, value)
		var flag rune
		if r.switchONfn(value) {
			flag = '1'
		} else {
			flag = '0'
		}
		b.setC(r.vSwitchState, flag)
	} else if f == r.inLeft {
		// new left signal
		b.setC(r.vLeft, value)
	} else if f == r.inRight {
		// new right signal
		b.setC(r.vRight, value)
	}

	// wait for the end, if no control relax switch to default state
	if _, ok:= multi[p] ; ok {
		if b.getC(r.vControl) == ' ' && multi[p] > 3 {
			b.setC(r.vSwitchState, r.defaultState)
			b.setC(r.vControl, r.defaultState)
		}
	}
	// if not enough inputs wait for next pass
	if !(b.getC(r.vControl) != ' ' && (b.getC(r.vLeft) != ' ' || b.getC(r.vRight) != ' ') ) {
		multi[p] += 1
		return
	}
	delete(multi, p) // no further passes
	if isZero(b.getC(r.vSwitchState)) {
		visited.done(p)
		return
	}
	if b.getC(r.vLeft) != ' ' {
		// left
		visited.done(p)
		propagate(visited, b, p, coord{p.x + 1, p.y}, b.getC(r.vLeft), multi)
	} else if b.getC(r.vRight) != ' '  {
		// from right
		visited.done(p)
		propagate(visited, b, p, coord{p.x - 1, p.y}, b.getC(r.vRight), multi)
	}

}

func propagate(visited board, b board, f coord, p coord, value rune, multi map[coord]int) {

	if b.off(p.x, p.y) {
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
		if visited.yes(p) || visited.yes(coord{p.x - 1, p.y}) {
			if value != b[p.x-1][p.y] {
				setMiddleMsg(fmt.Sprintf("'*' short circuit at %d %d: '%c' != '%c'", p.x, p.y, b[p.x-1][p.y], value))
			}
			return
		}
		visited.done(p)
		visited.set(p.x-1, p.y, 'Y')
		constant := b.get(p.x-1, p.y)
		propagate(visited, b, p, coord{p.x, p.y + 1}, constant, multi)
		propagate(visited, b, p, coord{p.x + 1, p.y}, constant, multi)
		propagate(visited, b, p, coord{p.x, p.y - 1}, constant, multi)

	case 'R':
		//               .
		//              3R.
		//               .
		if visited.yes(p) || visited.yes(coord{p.x - 1, p.y}) {
			return
		}
		visited.done(p)
		visited.set(p.x-1, p.y, 'Y')
		maxr := 1
		if isDigit(b[p.x-1][p.y]) {
			maxr = rune2Int(b[p.x-1][p.y])
			if maxr == 0 {
				maxr = 1
			}
		}
		randi := int2Rune(rand.Intn(maxr))
		propagate(visited, b, p, coord{p.x, p.y + 1}, randi, multi)
		propagate(visited, b, p, coord{p.x + 1, p.y}, randi, multi)
		propagate(visited, b, p, coord{p.x, p.y - 1}, randi, multi)

	case 'C':
		modulo := 2
		fraction := 4
		div := 1 << fraction
		if p.x > 0 && isDigit(b[p.x-1][p.y]) {
			visited[p.x-1][p.y] = 'Y'
			modulo = rune2Int(b[p.x-1][p.y])
			if modulo == 0 {
				modulo = 36
			}
			if p.x-1 > 0 && isDigit(b[p.x-2][p.y]) {
				visited[p.x-2][p.y] = 'Y'
				fraction = rune2Int(b[p.x-2][p.y])
				div = 1 << fraction
			}
		}
		clock := (clockTicks / div) % modulo
		clockRune := int2Rune(clock)
		if visited.yes(p) {
			return
		}
		visited.done(p)
		propagate(visited, b, p, coord{p.x, p.y - 1}, clockRune, multi)
		propagate(visited, b, p, coord{p.x, p.y + 1}, clockRune, multi)
		// propagate(visited, b, p, coord{p.x - 1, p.y}, clockRune)
		propagate(visited, b, p, coord{p.x + 1, p.y}, clockRune, multi)
	case '-':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		propagate(visited, b, p, coord{p.x + 1, p.y}, value, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)

	case '|':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		propagate(visited, b, p, coord{p.x, p.y - 1}, value, multi)
		propagate(visited, b, p, coord{p.x, p.y + 1}, value, multi)
	case '/':
		var end int
		for end = p.x + 1; end < width-2; end++ {
			if b[end][p.y] == '\\' {
				break
			}
		}
		if end == 0 {
			return
		}
		if visited.yes(p) || visited.yes(coord{end, p.y}) {
			return
		}
		visited.done(p)
		visited[end][p.y] = 'Y'
		propagate(visited, b, p, coord{end + 1, p.y}, value, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)
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
		if visited.yes(p) || visited[begin][p.y] == 'Y' {
			return
		}
		visited.done(p)
		visited[begin][p.y] = 'Y'
		propagate(visited, b, p, coord{begin - 1, p.y}, value, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)
	case '@':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		propagate(visited, b, p, coord{p.x, p.y - 1}, value, multi)
		propagate(visited, b, p, coord{p.x, p.y + 1}, value, multi)
		propagate(visited, b, p, coord{p.x + 1, p.y}, value, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)
	// Buffer left->right
	case '~':
		if visited.yes(p) || f.x != p.x-1 || f.y != p.y {
			return
		}
		visited.done(p)
		propagate(visited, b, p, coord{p.x + 1, p.y}, toBinary(value), multi)
	// Diode
	case '>':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		if !isZero(value) {
			propagate(visited, b, p, coord{p.x + 1, p.y}, value, multi)
		}
	// Diode
	case '<':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		if !isZero(value) {
			propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)
		}
	// Exit
	case 'E':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		if nonValue(value) || !isZero(value) {
			comment := b.getComment(coord{p.x + 1, p.y})
			_, _ = fmt.Fprintf(os.Stderr, "E cell exit at location %d %d. Expected '0', got '%c' (%d) - message: '%s'\n", p.x, p.y, value, rune2Int(value), comment)
			os.Exit(rune2Int(value))
		}

	// Beep
	case 'B':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		if !isZero(value) {
			beep()
		}

	// Invert
	case 'N':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		inverted := cond(value, '0', '1')
		propagate(visited, b, p, coord{p.x, p.y - 1}, inverted, multi)
		propagate(visited, b, p, coord{p.x, p.y + 1}, inverted, multi)
		propagate(visited, b, p, coord{p.x + 1, p.y}, inverted, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, inverted, multi)
	// Switch
	case 'S':
		//    .
		//   .S.
		//   ...
		//
		r := relay{}
		r.vSwitchState = coord{p.x+1, p.y-1}
		r.vControl = coord{p.x, p.y+1}
		r.vLeft = coord{p.x-1, p.y+1}
		r.vRight = coord{p.x+1, p.y+1}
		r.inControl = coord{p.x, p.y-1}
		r.inLeft = coord{p.x-1, p.y}
		r.inRight = coord{p.x+1, p.y}
		r.defaultState = '0' // OFF Normally Open (NO)
		r.switchONfn = func (r rune) bool { return !isZero(r) }
		r.propagate(visited, b, f, p, value, multi)

	// Inverted Switch
	case 'Z':
		//   ...
		//   .S.
		//    .
		//
		r := relay{}
		r.vSwitchState = coord{p.x+1, p.y+1}
		r.vControl = coord{p.x, p.y-1}
		r.vLeft = coord{p.x-1, p.y-1}
		r.vRight = coord{p.x+1, p.y-1}
		r.inControl = coord{p.x, p.y+1}
		r.inLeft = coord{p.x-1, p.y}
		r.inRight = coord{p.x+1, p.y}
		r.defaultState = '1' // ON Normally Closed (NC)
		r.switchONfn = isZero
		r.propagate(visited, b, f, p, value, multi)

	case 'L':
		if visited.yes(p) {
			return
		}
		visited.done(p)
		visited.done(coord{p.x, p.y - 1})
		b.set(p.x, p.y-1, value)
		propagate(visited, b, p, coord{p.x + 1, p.y}, value, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)
	case 'J':
		if visited.yes(p) {
			return
		}
		b.set(p.x, p.y+1, value)
		visited.done(p)
		visited.done(coord{p.x, p.y + 1})
		propagate(visited, b, p, coord{p.x + 1, p.y}, value, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)

	//       ..
	//       .=.
	//       ..
	//

	case '=':
		equals := func() bool {
			return b[p.x-1][p.y-1] == b[p.x-1][p.y+1]
		}
		logicGate(visited, b, f, p, value, equals, multi)
	case '.':
		and := func() bool {
			A := !isZero(b[p.x-1][p.y-1])
			B := !isZero(b[p.x-1][p.y+1])
			return A && B
		}
		logicGate(visited, b, f, p, value, and, multi)
	case '+':
		or := func() bool {
			A := !isZero(b[p.x-1][p.y-1])
			B := !isZero(b[p.x-1][p.y+1])
			return A || B
		}
		logicGate(visited, b, f, p, value, or, multi)
	case '#':
		exclusiveOr := func() bool {
			A := !isZero(b[p.x-1][p.y-1])
			B := !isZero(b[p.x-1][p.y+1])
			return A != B
		}
		logicGate(visited, b, f, p, value, exclusiveOr, multi)
	case '^':
		nand := func() bool {
			A := !isZero(b[p.x-1][p.y-1])
			B := !isZero(b[p.x-1][p.y+1])
			return !(A && B)
		}
		logicGate(visited, b, f, p, value, nand, multi)
	default:
	}
}

func int2Rune(i int) rune {
	if i >= 0 && i <= 9 {
		return rune('0' + i)
	}
	if i > 9 && i <= 9+26 {
		return rune('a' + i - 10)
	}
	return ' '
}
func isDigit(r rune) bool {
	return rune2Int(r) != -1
}
func isDecimal(r rune) bool {
	x := rune2Int(r)
	return x >= 0 && x <= 9
}

func rune2Int(r rune) int {
	if r >= '0' && r <= '9' {
		return int(r - '0')
	}
	if r >= 'a' && r <= 'z' {
		return int(r-'a') + 10
	}
	return -1
}

func logicGate(visited board, b board, f coord, p coord, value rune, conditionFn func() bool, multi map[coord]int) {
	//
	//    ..
	//    .X
	//    ..
	//
	if f.x == p.x && f.y == p.y-1 {
		b.set(p.x-1, p.y-1, value)
		visited[p.x-1][p.y-1] = 'Y'
	}
	if f.x == p.x && f.y == p.y+1 {
		b.set(p.x-1, p.y+1, value)
		visited[p.x-1][p.y+1] = 'Y'
	}
	//if visited[p.x-1][p.y+1] != 'Y' || visited[p.x-1][p.y-1] != 'Y' {
	//	return
	//}
	if visited.yes(p) {
		return
	}
	visited.done(p)
	if conditionFn() {
		b.set(p.x-1, p.y, '1')
		visited[p.x-1][p.y] = 'Y'
		propagate(visited, b, p, coord{p.x + 1, p.y}, '1', multi)
		return
	}
	b.set(p.x-1, p.y, '0')
	visited[p.x-1][p.y] = 'Y'
	propagate(visited, b, p, coord{p.x + 1, p.y}, '0', multi)
	return
}

var macros = make(map[string]board)

func expandMacro(pb board, home coord, name string) {
	mb, ok := macros[name]
	if !ok {
		macroBoard, err := loadMacroFile(fmt.Sprintf("%s.betula", name))
		if err != nil {
			setMiddleMsg(err.Error())
			return
		}
		macros[name] = macroBoard
		mb = macroBoard
	}
	parentWidth := len(pb)
	parentHeight := len(pb[0])
	macroWidth := len(mb)
	macroHeight := len(mb[0])
	for x := 0; x < macroWidth; x++ {
		for y := 0; y < macroHeight; y++ {
			if home.x+x >= parentWidth || home.y+y >= parentHeight {
				continue
			}
			if nonValue(mb[x][y]) {
				continue
			}
			pb.set(home.x+x, home.y+y, mb[x][y])
		}
	}

}
func interpreter(b board) {
	//var visited board
	for {
		clockTicks += 1
		boardMutex.Lock()
		roots := make([]coord, 0)
		//visited = makeBoard(width, height)
		// Find and copy Macros # TODO recursive...
		for y := 0; y < height-1; y++ {
			for x := 0; x < width; x++ {
				switch b[x][y] {
				case 'M':
					// collect the name
					name := make([]rune, width)
					var i int
					for i = 0; ; i++ {
						if x+i >= width {
							break
						}
						if nonValue(b[x+1+i][y]) {
							break
						}
						name[i] = b[x+1+i][y]
					}
					name = name[:i]
					if len(name) == 0 {
						break
					}
					expandMacro(b, coord{x, y + 1}, string(name))
				}
			}
		}
		// Find comments, roots and reset indicators
		for y := 0; y < height-1; y++ {
			for x := 0; x < width; x++ {
				switch b.get(x, y) {
				case '_':
					x = b.findCommentEnd(x+1, y) + 1
				case 'L':
					b.set(x, y-1, ' ')
				case 'J':
					b.set(x, y+1, ' ')
				case '*':
					roots = append(roots, coord{x, y})
				case 'C':
					roots = append(roots, coord{x, y})
				case 'R':
					roots = append(roots, coord{x, y})
				default:
				}
			}
		}
		visits := make([]*board, 0)
		multiPass := make(map[coord]int)

		for pass := 1;  ; pass++ {
			for _, p := range roots {
				visited := makeBoard(width, height)
				visits = append(visits, &visited)
				propagate(visited, b, nowhere, p, ' ', multiPass)
			}
			if len(multiPass) == 0 || pass > 5 {
				break
			}
			for p := range multiPass {
				visited := makeBoard(width, height)
				visits = append(visits, &visited)
				propagate(visited, b, nowhere, p, ' ', multiPass)
			}
		}
		//// Clear numeric values not reachable from any of the roots unless it's a comment
		//for y := 0; y < height-1; y++ {
		//	for x := 0; x < width; x++ {
		//		val := b.get(x, y)
		//		if val == '_' {
		//			x = b.findCommentEnd(x+1, y)
		//			continue
		//		}
		//		forget := true
		//		for _, v := range visits {
		//			if v.yes(coord{x, y}) {
		//				forget = false
		//				break
		//			}
		//		}
		//		if forget && isDecimal(val) {
		//			// b.set(x, y, ' ')
		//		}
		//	}
		//}
		boardMutex.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
}
func render(s tcell.Screen, b board) {
	for {
		boardMutex.Lock()
		val := b.get(cursorX, cursorY)
		setLeftMsg(fmt.Sprintf("%3d %3d %c %2d", cursorX, cursorY, val, rune2Int(val)))
		boardMutex.Unlock()
		view(s, b)
		s.Show()
		time.Sleep(100 * time.Millisecond)
	}
}
func minInt(x int, x2 int) int {
	if x2 < x {
		return x2
	}
	return x
}

func maxInt(x, x2 int) int {
	if x2 > x {
		return x2
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
func loadMacroFile(filename string) (board, error) {
	macroWidth, macroHeight, err := sizeOfFile(filename)
	if err != nil {
		return nil, err
	}
	return loadFile(filename, macroWidth, macroHeight)
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
			setMiddleMsg(fmt.Sprintf("Loaded %s, into width %d, height %d", filename, width, height))
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
		b.set(x, y, r)
		x += 1
	}
}

func (b board) saveFile(filename string) {

	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		setMiddleMsg(err.Error())
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
				setMiddleMsg(err.Error())
				return
			}
		}
		_, err = fmt.Fprintf(fd, "\n")
		if err != nil {
			setMiddleMsg(err.Error())
			return
		}
	}
	setMiddleMsg(fmt.Sprintf("Saved %s, width %d, height %d", filename, actualWidth, actualHeight))

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

// off() - Are we off the board?
func (b board) off(x int, y int) bool {
	if x < 0 || y < 0 || x >= len(b) {
		return true
	}
	if y >= len(b[x]) {
		return true
	}
	return false
}

// set() - Set a value but don't throw an error if outside the board
func (b board) set(x int, y int, r rune) {
	b.setC(coord{x, y}, r)
}

// setC() - Set a value but don't throw an error if outside the board
func (b board) setC(p coord, r rune) {
	if b.off(p.x, p.y) {
		return
	}
	b[p.x][p.y] = r
}

func (b board) yes(p coord) bool {
	if b.off(p.x, p.y) {
		return false
	}
	return b[p.x][p.y] == 'Y'
}

func (b board) done(p coord) {
	if b.off(p.x, p.y) {
		return
	}
	b[p.x][p.y] = 'Y'
}

func (b board) getC(p coord) rune {
	if b.off(p.x, p.y) {
		return ' '
	}
	return b[p.x][p.y]
}

func (b board) get(x, y int) rune {
	return b.getC(coord{x, y})
}

func (b board) findCommentEnd(x int, y int) int {
	for ; x < len(b); x++ {
		if b.get(x, y) == '_' {
			break
		}
	}
	return x
}

// getComment - look for the next comment on this row.
// p.x may be to the left of the comment
// if no comment found return empty string
func (b board) getComment(p coord) interface{} {
	msg := make([]rune, 0)
	x := p.x
	for ; x < len(b); x++ {
		if b.get(x, p.y) == '_' {
			break
		}
	}
	if x == len(b) {
		return "" // did not find a comment
	}
	x += 1
	for ; x < len(b); x++ {
		if b.get(x, p.y) == '_' {
			break
		}
		msg = append(msg, b.get(x, p.y))
	}
	return string(msg)
}
func setMiddleMsgRaw(s tcell.Screen, msg string) {
	w, _ := s.Size()
	runes := []rune(msg)
	for i, r := range runes {
		if i >= w {
			break
		}
		s.SetContent(i, 0, r, nil, tcell.StyleDefault)
	}
	_, _ = fmt.Fprintf(logfd, "%s\n", msg)
}

var setMiddleMsg func(string)
var setLeftMsg func(string)
var beep func()

var logfd *os.File

type rectangle struct {
	topLeft     coord
	bottomRight coord
}

const (
	// KeysNormal State of the user interface handling of keys
	KeysNormal = iota
	KeysSelecting
)

type editor struct {
	ks                 int
	pivot              coord
	selectionRectangle rectangle
	cutPasteBuffer     board
}

var theEditor = newEditor()

func newEditor() (e *editor) {
	return &editor{
		ks: KeysNormal,
		selectionRectangle: rectangle{
			coord{0, 0},
			coord{0, 0},
		},
		cutPasteBuffer: makeBoard(0, 0),
	}
}

func newRectangle(x, y, x2, y2 int) rectangle {
	return rectangle{
		coord{minInt(x, x2), minInt(y, y2)},
		coord{maxInt(x, x2), maxInt(y, y2)},
	}
}

func (r *rectangle) inside(p coord) bool {
	return p.x >= r.topLeft.x && p.x <= r.bottomRight.x && p.y >= r.topLeft.y && p.y <= r.bottomRight.y
}

func (e *editor) update(p coord) {
	tl := coord{minInt(p.x, e.pivot.x), minInt(p.y, e.pivot.y)}
	br := coord{maxInt(p.x, e.pivot.x), maxInt(p.y, e.pivot.y)}
	e.selectionRectangle.topLeft = tl
	e.selectionRectangle.bottomRight = br
}

func (e *editor) noShift() {
	theEditor.ks = KeysNormal
}

func (e *editor) move(cursor coord, cursorAfter coord, modifiers tcell.ModMask) {
	if modifiers&tcell.ModShift != 0 { // Shift key
		if e.ks == KeysNormal {
			// starting selection
			e.pivot = cursor
			e.selectionRectangle = newRectangle(cursor.x, cursor.y, cursorAfter.x, cursorAfter.y)
		} else {
			// already in selection mode
			e.update(cursorAfter)
		}
		e.ks = KeysSelecting
	}
}

func (e *editor) copy(b board) {
	if e.ks == KeysNormal {
		return
	} else {
		// in selection mode
		e.cutPasteBuffer = makeBoard(e.selectionRectangle.bottomRight.x-e.selectionRectangle.topLeft.x+1, e.selectionRectangle.bottomRight.y-e.selectionRectangle.topLeft.y+1)
		for x := e.selectionRectangle.topLeft.x; x <= e.selectionRectangle.bottomRight.x; x++ {
			for y := e.selectionRectangle.topLeft.y; y <= e.selectionRectangle.bottomRight.y; y++ {
				e.cutPasteBuffer.set(x-e.selectionRectangle.topLeft.x, y-e.selectionRectangle.topLeft.y, b.get(x, y))
			}
		}
		e.ks = KeysNormal
		cursorX = e.selectionRectangle.topLeft.x
		cursorY = e.selectionRectangle.topLeft.y
	}
}

func (e *editor) paste(b board, cursor coord) {
	for x := 0; x < len(e.cutPasteBuffer); x++ {
		for y := 0; y < len(e.cutPasteBuffer[x]); y++ {
			b.set(cursor.x+x, cursor.y+y, e.cutPasteBuffer.get(x, y))
		}
	}
}

func (e *editor) cut(b board, cursor coord) {
	if e.ks == KeysNormal {
		return
	} else {
		e.copy(b)
		e.ks = KeysSelecting // TODO
		e.delete(b, cursor)
		e.ks = KeysNormal
	}
}

func (e *editor) delete(b board, cursor coord) {
	if e.ks == KeysNormal {
		b.set(cursor.x, cursor.y, ' ')
	} else {
		// in selection mode
		for x := e.selectionRectangle.topLeft.x; x <= e.selectionRectangle.bottomRight.x; x++ {
			for y := e.selectionRectangle.topLeft.y; y <= e.selectionRectangle.bottomRight.y; y++ {
				b.set(x, y, ' ')
			}
		}
		e.ks = KeysNormal
		cursorX = e.selectionRectangle.topLeft.x
		cursorY = e.selectionRectangle.topLeft.y
	}
}

func (e *editor) style(p coord, cellStyle tcell.Style) tcell.Style {
	if e.ks == KeysSelecting && e.selectionRectangle.inside(p) {
		return cellStyle.Background(tcell.ColorLightSlateGray)
	}
	return cellStyle
}

func main() {
	logfd, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func(fd *os.File) { _ = fd.Close() }(logfd)

	var theBoard board

	setLeftMsg = func(msg string) {
		runes := []rune(msg)
		for i, r := range runes {
			theBoard.set(i, height-1, r)
		}
		//		_, _ = fmt.Fprintf(logfd, "%s\n", msg)
	}

	filename := "untitled.betula"

	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	setMiddleMsg = func(msg string) {
		setMiddleMsgRaw(s, msg)
	}

	beep = func() {
		_ = s.Beep()
	}

	// Set default text style
	defStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	s.SetStyle(defStyle)

	// Clear screen
	s.Clear()

	screenWidth, screenHeight := s.Size()
	_, _ = fmt.Fprintf(logfd, "screenWidth %d, screenHeight %d\n", screenWidth, screenHeight)
	cursorX = screenWidth / 2
	cursorY = screenHeight / 2

	if len(os.Args) > 1 {
		filename = os.Args[1]
		var err error
		fileWidth, fileHeight, err := sizeOfFile(filename)
		if err != nil {
			log.Fatalf("ERROR: file %s - %s\n", os.Args[1], err)
		}
		_, _ = fmt.Fprintf(logfd, "fileWidth %d, fileHeight %d\n", fileWidth, fileHeight)
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
			if ev.Modifiers()&tcell.ModShift == 0 && ev.Key() != tcell.KeyDelete && ev.Key() != tcell.KeyCtrlC && ev.Key() != tcell.KeyCtrlX { // TODO
				theEditor.noShift()
			}
			switch ev.Key() {
			case tcell.KeyCtrlQ:
				quit()
			case tcell.KeyF5:
				// Toggle the value under the cursor
				boardMutex.Lock()
				r := theBoard[cursorX][cursorY]
				if nonValue(r) || r == '0' {
					theBoard.set(cursorX, cursorY, '1')
				} else {
					theBoard.set(cursorX, cursorY, '0')
				}
				boardMutex.Unlock()
			case tcell.KeyDelete:
				boardMutex.Lock()
				theEditor.delete(theBoard, coord{cursorX, cursorY})
				boardMutex.Unlock()
				// follow wires
				if !nonValue(theBoard.get(cursorX+1, cursorY)) {
					cursorX += 1
				} else if !nonValue(theBoard.get(cursorX-1, cursorY)) {
					cursorX -= 1
				} else if !nonValue(theBoard.get(cursorX, cursorY+1)) {
					cursorY += 1
				} else if !nonValue(theBoard.get(cursorX, cursorY-1)) {
					cursorY -= 1
				}
			case tcell.KeyCtrlC:
				boardMutex.Lock()
				theEditor.copy(theBoard)
				boardMutex.Unlock()
			case tcell.KeyCtrlV:
				boardMutex.Lock()
				theEditor.paste(theBoard, coord{cursorX, cursorY})
				boardMutex.Unlock()
			case tcell.KeyCtrlX:
				boardMutex.Lock()
				theEditor.cut(theBoard, coord{cursorX, cursorY})
				boardMutex.Unlock()
			case tcell.KeyBackspace2:
				if cursorX > 0 {
					cursorX -= 1
				}
				boardMutex.Lock()
				theBoard.set(cursorX, cursorY, ' ')
				boardMutex.Unlock()
			case tcell.KeyUp:
				if cursorY != 0 {
					theEditor.move(coord{cursorX, cursorY}, coord{cursorX, cursorY - 1}, ev.Modifiers())
					cursorY -= 1
				}
			case tcell.KeyDown:
				if cursorY < height-2 {
					theEditor.move(coord{cursorX, cursorY}, coord{cursorX, cursorY + 1}, ev.Modifiers())
					cursorY += 1
				}
			case tcell.KeyLeft:
				if cursorX != 0 {
					theEditor.move(coord{cursorX, cursorY}, coord{cursorX - 1, cursorY}, ev.Modifiers())
					cursorX -= 1
				}
			case tcell.KeyRight:
				if cursorX < width-1 {
					theEditor.move(coord{cursorX, cursorY}, coord{cursorX + 1, cursorY}, ev.Modifiers())
					cursorX += 1
				}
			case tcell.KeyF4: // for inside the debugger
				boardMutex.Lock()
				theBoard.saveFile(filename)
				boardMutex.Unlock()
			case tcell.KeyCtrlS:
				boardMutex.Lock()
				theBoard.saveFile(filename)
				boardMutex.Unlock()
			case tcell.KeyRune:
				k := ev.Rune()
				boardMutex.Lock()
				theBoard.set(cursorX, cursorY, k)
				boardMutex.Unlock()
				// follow wires, user-friendly cursor positions
				switch k {
				case '*':
					cursorX -= 1
				case '|':
					if nonValue(theBoard[cursorX][cursorY+1]) {
						cursorY += 1
					} else if nonValue(theBoard[cursorX][cursorY-1]) {
						cursorY -= 1
					}
				case '-':
					if nonValue(theBoard[cursorX+1][cursorY]) {
						cursorX += 1
					} else if nonValue(theBoard[cursorX-1][cursorY]) {
						cursorX -= 1
					}
				default:
					cursorX += 1
				}
			default:
			}
		case *tcell.EventMouse:
			cursorX, cursorY = ev.Position()
		}
	}
}

var colors = map[rune]tcell.Color{
	'=': tcell.ColorBlack,
	'.': tcell.ColorBlack,
	'#': tcell.ColorBlack,
	'+': tcell.ColorBlack,
	'^': tcell.ColorBlack,

	'-':  tcell.ColorLightBlue,
	'|':  tcell.ColorLightBlue,
	'/':  tcell.ColorLightBlue,
	'\\': tcell.ColorLightBlue,
	'@':  tcell.ColorLightBlue,

	'?': tcell.ColorRed,

	'E': tcell.ColorBlack,
	'B': tcell.ColorBlack,

	'_': tcell.ColorLightSeaGreen,

	'L': tcell.ColorBlack,
	'J': tcell.ColorBlack,
	'N': tcell.ColorBlue,
	'*': tcell.ColorBlack,
	'R': tcell.ColorBlack,
	'C': tcell.ColorDarkBlue,
	'S': tcell.ColorBlack,
	'Z': tcell.ColorBlack,

	'M': tcell.ColorBlack,

	'0': tcell.ColorRed,
	'9': tcell.ColorOrange,
	'a': tcell.ColorBeige,
}
var backgrounds = map[rune]tcell.Color{
	'=': tcell.ColorOrange,
	'.': tcell.ColorOrange,
	'#': tcell.ColorOrange,
	'+': tcell.ColorOrange,
	'^': tcell.ColorOrange,

	'E': tcell.ColorRed,
	'B': tcell.ColorRed,

	'J': tcell.ColorLightBlue,
	'L': tcell.ColorLightBlue,
	'N': tcell.ColorLightPink,
	'S': tcell.ColorLightPink,
	'Z': tcell.ColorLightPink,

	'M': tcell.ColorLightGoldenrodYellow,

	'C': tcell.ColorLightGreen,
	'*': tcell.ColorLightGreen,
	'R': tcell.ColorLightGreen,
}

func styleOf(r rune) tcell.Style {
	var s = tcell.StyleDefault
	if isDigit(r) {
		if isDecimal(r) {
			if r == '0' {
				s = s.Foreground(colors['0'])
			} else {
				s = s.Foreground(colors['9'])
			}
		} else {
			s = s.Foreground(colors['a'])
		}
	} else {
		if c, ok := colors[r]; ok {
			s = s.Foreground(c)
		}
	}
	if c, ok := backgrounds[r]; ok {
		s = s.Background(c)
	}
	if r == '*' {
		s = s.Bold(true)
	}
	return s

}

func view(s tcell.Screen, b board) {
	boardMutex.Lock()
	commentStyle := tcell.StyleDefault.Foreground(colors['_'])
	for y := 0; y < height-1; y++ {
		inComment := false // parsing state
		for x := 0; x < width; x++ {
			val := b.get(x, y)
			sty := styleOf(val)
			if val == '_' { // Scan and display the comment
				sty = commentStyle
				inComment = !inComment
			}
			if inComment {
				sty = commentStyle
			}
			stile := theEditor.style(coord{x, y}, sty)
			s.SetContent(x, y, b[x][y], nil, stile)
		}
	}
	for x := 0; x < width; x++ {
		s.SetContent(x, height-1, b[x][height-1], nil, tcell.StyleDefault)
	}
	s.SetContent(cursorX, cursorY, b.get(cursorX, cursorY), nil, tcell.StyleDefault.Reverse(true))
	boardMutex.Unlock()
}
