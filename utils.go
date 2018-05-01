package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// exitSignalHandling wait for interrump signal to gracefully shutdown the server with a timeout configured inside context
func exitSignalHandling(cancel context.CancelFunc) {
	quit := make(chan os.Signal, 10)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-quit
	cancel()
}
