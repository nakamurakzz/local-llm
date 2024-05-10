package main

import (
	"fmt"
	"os"
	"os/signal"
)

func main() {
	os.Exit(run())
}

func run() int {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		os.Exit(0)
	}()

	var buf [256]byte
	for {
		fmt.Fprint(os.Stdout, "prompt here: ")
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			fmt.Println("Error reading from stdin:", err)
			return 1
		}
		if n == 0 {
			return 0
		}
		if n == 1 {
			continue
		}

		_, err = os.Stdout.Write(buf[:n])
		if err != nil {
			fmt.Println("Error writing to stdout:", err)
			return 1
		}
	}
}
