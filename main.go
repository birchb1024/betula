package main

import (
	"github.com/gdamore/tcell"

	"log"
	"os"
	//	"strconv"
)

var width, height int
var cursorX int
var cursorY int

func main() {
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

	width, height = s.Size()
	cursorX = width/2
	cursorY = height/2

	matrix := make([][]uint8, width)
	for x := range matrix {
		matrix[x] = make([]uint8, height)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			matrix[x][y] = 0
		}
	}
	//for i := 0; i < 2; i++ {
	//	for y := 0; y < height; y++ {
	//		for x := 0; x < width; x++ {
	//			if matrix[x][y] < 128 && x > 1 && x < width-1 {
	//				matrix[(x+1)%width][y] += 4
	//				matrix[(x-1)%width][y] -= 4
	//			}
	//		}
	//	}
		//var ignore int
		//fmt.Println(i)
		//fmt.Scanln(&ignore)
	//}
	quit := func() {
		s.Fini()
		s.EnableMouse()
		os.Exit(0)
	}
	for {
		// Update screen
		view(s, matrix)

		s.Show()

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

			case tcell.KeyDelete:
				matrix[cursorX][cursorY] = 0
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
				matrix[cursorX][cursorY] = uint8(ev.Rune() - ' ')
			default:
			}
		case *tcell.EventMouse:
			s.SetContent(0, height-1, nil,  []rune(Sprint("%#v", ev.Buttons())), )
		}
	}
}

func view(s tcell.Screen, m [][]uint8) {

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			s.SetContent(x, y, rune(' '+m[x][y]), nil, tcell.StyleDefault)
			//fmt.Printf("%3d ", m[x][y])
		}
		//fmt.Printf("\n")
	}
	//fmt.Printf("\n")
	s.SetContent(cursorX, cursorY, rune(' '+m[cursorX][cursorY]), nil, tcell.StyleDefault.Reverse(true))
}
