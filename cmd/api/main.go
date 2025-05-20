package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"evedem_api/internal/commons"
	"evedem_api/internal/server"
)

var VERSION string = `1.0.0`

func gracefulShutdown(apiServer *http.Server, done chan bool) {

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	log.Println("Starting Everdeem Server\n\t-- Version : ", VERSION)

	for _, val := range os.Args {
		switch val {
		case "-d", "--debug":
			log.Println("Warning!! Debug mode activated, no authntification is required.")
			commons.DebugMode = true
		case "-h", "--help":
			log.Println("Undefined")
		case "-v", "--version":
			return
		}
	}

	log.Println("Everdeem API Server Started")
	server := server.NewServer()

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(server, done)

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")
}
