//go:build headless
// +build headless

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	app := NewApp()
	app.startup(context.Background())

	log.Println("Headless Muhlya is running. Press Ctrl+C to stop.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutdown signal received, stopping Muhlya...")
	app.StopAutoTrading()
	if app.broker != nil {
		app.broker.Close()
	}
	if app.db != nil {
		_ = app.db.Close()
	}
}
