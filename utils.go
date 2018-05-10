package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// exitSignalHandling wait for interrump signal to gracefully shutdown the server with a timeout configured inside context
func exitSignalHandling(cancel context.CancelFunc) {
	quit := make(chan os.Signal, 10)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-quit
	cancel()
}

func exitTimeout(ctx context.Context) {
	<-ctx.Done()
	after := time.After(5 * time.Second)
	<-after
	log.Error("jagozzi: timeout while exiting")
	os.Exit(1)
}

func applyLogLevel(level *string) {
	if level == nil {
		return
	}

	switch strings.ToLower(*level) {
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	}
}
