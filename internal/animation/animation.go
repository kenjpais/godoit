package animation

import "fmt"

func StartBootAnimation() {
	fmt.Print("\033[H\033[2J")
	fmt.Println("Welcome to GoDoIt! 🚀")
	fmt.Println("Starting up...")
	spinner := []rune{'|', '/', '-', '\\'}
	for i := 0; i < 20; i++ {
		fmt.Printf("\r%c Loading tasks... ", spinner[i%len(spinner)])
	}
	fmt.Printf("\r✔ All systems ready!\n")	
}