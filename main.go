package main

import (
	"fmt"
	"math/rand"
)

var width, height int

func main() {
	width = 10
	height = 10
	matrix := make([][]uint8, width)
	for x := range matrix {
		matrix[x] = make([]uint8, height)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			matrix[x][y] = uint8(rand.Uint32())
		}
	}
	for i := 0; i < 20; i++ {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if matrix[x][y] < 128 && x > 1 && x < width-1 {
					matrix[(x+1)%width][y] += 4
					matrix[(x-1)%width][y] -= 4
				}
			}
		}
		view(matrix)
		var ignore int
		fmt.Println(i)
		fmt.Scanln(&ignore)
	}
}

func view(m [][]uint8) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fmt.Printf("%3d ", m[x][y])
		}
		fmt.Printf("\n")
	}
	fmt.Printf("\n")
}
