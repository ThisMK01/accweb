package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	fmt.Println("Server starting with version 1")
	fmt.Println("Track monza was set and updated")
	fmt.Println("RegisterToLobby succeeded")
	fmt.Println("Detected sessionPhase <Starting> -> <Active> (Active)")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("Shutting down mock accServer...")
			return
		case t := <-ticker.C:
			fmt.Printf("== Server running at %s ==\n", t.Format(time.RFC3339))
		}
	}
}
