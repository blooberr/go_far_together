package main

import (
	"fmt"
	"math/rand"
	"time"
)

func hello(from string, i int) {
	delay := time.Duration(rand.Intn(3)+1) * time.Second
	time.Sleep(delay)
	fmt.Printf("Hello %s: %d\n", from, i)
}

func main() {
	for i := 0; i < 5; i++ {
		go hello("goroutine", i)
	}

	hello("Instacart", 100)
	time.Sleep(10 * time.Second)
}
