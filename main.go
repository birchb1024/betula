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

type wire struct {
	outputs []coord
}

func (w *wire) propagate(visited board, b board, p coord, value rune, multi map[coord]int) {
	if visited.yes(p) {
		return
	}
	visited.done(p)
	for _, out := range w.outputs {
		propagate(visited, b, p, out, value, multi)
	}
}

type diode struct {
	output coord
}

func (d *diode) propagate(visited board, b board, p coord, value rune, multi map[coord]int) {
	if visited.yes(p) {
		return
	}
	visited.done(p)
	if !isZero(value) {
		propagate(visited, b, p, d.output, value, multi)
	}
}

func propagate(visited board, b board, f coord, p coord, value rune, multi map[coord]int) {

	if b.off(p.x, p.y) {
		return
	}
	if len(visited) > 1 && nonValue(b[p.x][p.y]) {
		return
	}

	switch b.getC(p) {

	case '*':
		//               .
		//              3*.
		//               .
		outputs := []coord{{p.x, p.y + 1}, {p.x + 1, p.y}, {p.x, p.y - 1}}
		if visited.yes(p) {
			return
		}
		visited.done(p)
		constant := b.get(p.x-1, p.y)
		for _, out := range outputs {
			propagate(visited, b, p, out, constant, multi)
		}

	case 'R':
		//               .
		//              3R.
		//               .
		maxrand := 1
		maxrxy := coord{p.x-1, p.y}
		maxrune := b.getC(maxrxy)
		outputs := []coord{{p.x, p.y + 1}, {p.x + 1, p.y}, {p.x, p.y - 1}}
		if visited.yes(p) {
			return
		}
		visited.done(p)
		if isDigit(maxrune) {
			maxrand = rune2Int(maxrune)
			if maxrand == 0 {
				maxrand = 1
			}
		}
		randi := int2Rune(rand.Intn(maxrand))
		for _, out := range outputs {
			propagate(visited, b, p, out, randi, multi)
		}

	case 'C':
		//               .
		//             fmC.
		//               .
		if visited.yes(p) {
			return
		}
		outputs := []coord{{p.x, p.y + 1}, {p.x + 1, p.y}, {p.x, p.y - 1}}
		moduloCo := coord{p.x-1, p.y}
		moduloRune := b.getC(moduloCo)
		fractionCo := coord{p.x-2, p.y}
		fractionRune := b.getC(fractionCo)
		modulo := 2
		fraction := 4
		div := 1 << fraction
		if isDigit(moduloRune) {
			modulo = rune2Int(moduloRune)
			if modulo == 0 {
				modulo = 36
			}
			if isDigit(fractionRune) {
				fraction = rune2Int(fractionRune)
				div = 1 << fraction
			}
		}
		clock := (clockTicks / div) % modulo
		clockRune := int2Rune(clock)
		visited.done(p)
		for _, out := range outputs {
			propagate(visited, b, p, out, clockRune, multi)
		}

	case '-':
		leftRight := wire{[]coord{{p.x+1, p.y}, {p.x - 1, p.y}}}
		leftRight.propagate(visited, b, p, value, multi)

	case '|':
		upDown := wire{[]coord{{p.x, p.y - 1}, {p.x, p.y + 1}}}
		upDown.propagate(visited, b, p, value, multi)

	case '/':
		//
		//      /     \
		//
		var end int
		// Find the end
		for end = p.x + 1; end < width-2; end++ {
			if b.get(end, p.y) == '\\' {
				break
			}
		}
		if end == 0 { // no end so do nothing
			return
		}
		if visited.yes(p) || visited.yes(coord{end, p.y}) {
			return
		}
		visited.done(p)
		visited.done(coord{end, p.y})
		propagate(visited, b, p, coord{end + 1, p.y}, value, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)

	case '\\':
		//
		//      /     \
		//
		var begin int
		// find the start
		for begin = p.x - 1; begin > 0; begin-- {
			if b.get(begin, p.y) == '/' {
				break
			}
		}
		if begin == 0 { // no start so nothing to do
			return
		}
		if visited.yes(p) || visited.yes(coord{begin, p.y}) {
			return
		}
		visited.done(p)
		visited.done(coord{begin, p.y})
		propagate(visited, b, p, coord{begin - 1, p.y}, value, multi)
		propagate(visited, b, p, coord{p.x - 1, p.y}, value, multi)

	case '@':
		blob := wire{[]coord{
			{p.x, p.y - 1},
			{p.x, p.y + 1},
			{p.x + 1, p.y},
			{p.x - 1, p.y},
		}}
		blob.propagate(visited, b, p, value, multi)

	case '~':
		// Buffer left->right
		output := coord{p.x + 1, p.y}
		if visited.yes(p) || f.x != p.x-1 || f.y != p.y {
			return
		}
		visited.done(p)
		propagate(visited, b, p, output, toBinary(value), multi)

	case '>':
		// Diode
		lrdiode := diode{coord{p.x + 1, p.y}}
		lrdiode.propagate(visited, b, p, value, multi)

	case '<':
		// Diode
		rldiode := diode{coord{p.x - 1, p.y}}
		rldiode.propagate(visited, b, p, value, multi)

	case 'E':
		// Exit
		if visited.yes(p) {
			return
		}
		visited.done(p)
		if nonValue(value) || !isZero(value) {
			comment := b.getComment(coord{p.x + 1, p.y})
			_, _ = fmt.Fprintf(os.Stderr, "E cell exit at location %d %d. Expected '0', got '%c' (%d) - message: '%s'\n", p.x, p.y, value, rune2Int(value), comment)
			os.Exit(rune2Int(value))
		}

	case 'B':
		// Beep
		if visited.yes(p) {
			return
		}
		visited.done(p)
		if !isZero(value) {
			beep()
		}

	case 'N':
		// Invert
		inverter := wire{[]coord{
			{p.x, p.y - 1},
			{p.x, p.y + 1},
			{p.x + 1, p.y},
			{p.x - 1, p.y},
		}}
		inverted := cond(value, '0', '1')
		inverter.propagate(visited, b, p, inverted, multi)

	case 'S':
		// Normally Open Relay Switch
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

	case 'Z':
		// Normally Closed Relay Switch
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
		// Lamp on top of wire
		//       .
		//		.L.
		//
		topLamp := wire{[]coord{{p.x+1, p.y}, {p.x - 1, p.y}}}
		b.set(p.x, p.y-1, value)
		topLamp.propagate(visited, b, p, value, multi)

	case 'J':
		// Lamp underneath wire
		//
		//		.J.
		//       '
		//
		bottomLamp := wire{[]coord{{p.x+1, p.y}, {p.x - 1, p.y}}}
		b.set(p.x, p.y+1, value)
		bottomLamp.propagate(visited, b, p, value, multi)

	case '=':
		//       ..
		//       .=.
		//       ..
		//
		equals := func(A, B rune) bool {
			return A == B
		}
		runeGate(visited, b, f, p, value, equals, multi)
	case '.':
		and := func(A, B bool) bool { return A && B }
		logicGate(visited, b, f, p, value, and, multi)
	case '+':
		or := func(A, B bool) bool { return A || B }
		logicGate(visited, b, f, p, value, or, multi)
	case '#':
		exclusiveOr := func(A, B bool) bool { return A != B }
		logicGate(visited, b, f, p, value, exclusiveOr, multi)
	case '^':
		nand := func(A, B bool) bool { return !(A && B) }
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

type gate struct {
	inTop coord
	inBottom coord
	vTopXY coord
	vBottomXY coord
	vOut coord
	output coord
	runeCondition func(rune, rune) bool
	condition func(bool, bool) bool
}

func runeGate(visited board, b board, f coord, p coord, value rune, conditionFn func(rune, rune) bool, multi map[coord]int) {
	//
	//    ..
	//    .X
	//    ..
	//
	var g gate

	g.inTop = coord{p.x, p.y - 1}
	g.inBottom = coord{p.x, p.y + 1}
	g.vTopXY = coord{p.x - 1, p.y - 1}
	g.vBottomXY = coord{p.x - 1, p.y + 1}
	g.vOut = coord{p.x - 1, p.y}
	g.output = coord{p.x + 1, p.y}
	g.runeCondition = conditionFn
	g.propagate(visited, b, f, p, value, multi)
}

func logicGate(visited board, b board, f coord, p coord, value rune, conditionFn func(bool, bool) bool, multi map[coord]int) {
	//
	//    ..
	//    .X
	//    ..
	//
	var g gate

	g.inTop = coord{p.x, p.y - 1}
	g.inBottom = coord{p.x, p.y + 1}
	g.vTopXY = coord{p.x - 1, p.y - 1}
	g.vBottomXY = coord{p.x - 1, p.y + 1}
	g.vOut = coord{p.x - 1, p.y}
	g.output = coord{p.x + 1, p.y}
	g.condition = conditionFn
	g.propagate(visited, b, f, p, value, multi)
}
func (g *gate) propagate(visited board, b board, f coord, p coord, value rune, multi map[coord]int) {

	// ignore if not the two inputs
	if !(f == g.inTop || f == g.inBottom ) {
		return
	}
	if visited.yes(p) {
		return
	}
	// if not seen before
	if _, ok:= multi[p] ; !ok {
		multi[p] = 1
		// reset variables
		for _, v := range []coord{g.vTopXY, g.vBottomXY, g.vOut} {
			b.setC(v, ' ')
		}
	}
	if f == g.inTop {
		b.setC(g.vTopXY, value)
	} else if f == g.inBottom {
		b.setC(g.vBottomXY, value)
	}
	top := b.getC(g.vTopXY)
	bottom := b.getC(g.vBottomXY)
	// if not enough inputs wait for next pass
	if top == ' ' || bottom == ' ' {
		multi[p] += 1
		return
	}
	// Have both inputs
	delete(multi, p) // no further passes
	visited.done(p)

	A := !isZero(top)
	B := !isZero(bottom)
	b.setC(g.vTopXY, ' ')
	b.setC(g.vBottomXY, ' ')
	outputValue := '0'
	if g.condition != nil {
		if g.condition(A, B) {
			outputValue = '1'
		}
	} else if g.runeCondition != nil {
		if g.runeCondition(top, bottom) {
			outputValue = '1'
		}
	}
	b.setC(g.vOut, outputValue)
	propagate(visited, b, p, g.output, outputValue, multi)
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
		multiPass := make(map[coord]int)

		for pass := 1;  ; pass++ {
			for _, p := range roots {
				visited := makeBoard(width, height)
				propagate(visited, b, nowhere, p, ' ', multiPass)
			}
			if len(multiPass) == 0 || pass > 4 {
				break
			}
			for p := range multiPass {
				visited := makeBoard(width, height)
				propagate(visited, b, nowhere, p, ' ', multiPass)
			}
		}
		boardMutex.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
}
func render(s tcell.Screen, b board) {
	for {
		boardMutex.Lock()
		val := b.get(cursorX, cursorY)
		setLeftMsg(fmt.Sprintf("%d %3d %3d %c %2d", clockTicks, cursorX, cursorY, val, rune2Int(val)))
		boardMutex.Unlock()
		view(s, b)
		s.Show()
		time.Sleep(200 * time.Millisecond)
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
