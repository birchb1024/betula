package main

import (
	"github.com/gdamore/tcell"
	//	"fmt"
	"log"
	"math/rand"
	"os"

	//	"strconv"
)

var width, height int

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

	matrix := make([][]uint8, width)
	for x := range matrix {
		matrix[x] = make([]uint8, height)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			matrix[x][y] = uint8(rand.Uint32())
		}
	}
	for i := 0; i < 2; i++ {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if matrix[x][y] < 128 && x > 1 && x < width-1 {
					matrix[(x+1)%width][y] += 4
					matrix[(x-1)%width][y] -= 4
				}
			}
		}
		//var ignore int
		//fmt.Println(i)
		//fmt.Scanln(&ignore)
	}
	quit := func() {
		s.Fini()
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
		}
	}
}

func view(s tcell.Screen, m [][]uint8) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			s.SetContent(x, y, rune('A' + m[x][y]), nil, tcell.StyleDefault)
			//fmt.Printf("%3d ", m[x][y])
		}
		//fmt.Printf("\n")
	}
	//fmt.Printf("\n")
}
