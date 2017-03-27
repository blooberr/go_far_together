package main

import (
	"log"
	"time"
)

// START OMIT
func Boom(stop chan bool) {
	time.Sleep(10 * time.Second)
	log.Printf("Sending stop..")
	stop <- true
}

func Worker(stop chan bool) {
	heartbeat := time.Tick(2 * time.Second)
	for {
		select {
		case <-heartbeat:
			log.Printf("heartbeating!\n")
		case <-stop:
			log.Printf("Boom!\n")
			return
		}
	}
}

func main() {
	stopChan := make(chan bool)
	go Boom(stopChan)
	Worker(stopChan)
}
// END OMIT
