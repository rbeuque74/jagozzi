package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rbeuque74/jagozzi/config"
	"github.com/rbeuque74/jagozzi/plugins"
	_ "github.com/rbeuque74/jagozzi/plugins/command"
	_ "github.com/rbeuque74/jagozzi/plugins/http"
	_ "github.com/rbeuque74/jagozzi/plugins/marathon"
	_ "github.com/rbeuque74/jagozzi/plugins/processes"
	_ "github.com/rbeuque74/jagozzi/plugins/supervisor"
	log "github.com/sirupsen/logrus"
)

var (
	configFile = flag.String("cfg", "./jagozzi.yml", "path to config file")
	logLevel   = flag.String("level", "info", "verbosity level for application logs")
)

func main() {
	flag.Parse()
	if logLevel != nil {
		switch strings.ToLower(*logLevel) {
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

	ctx, cancel := context.WithCancel(context.Background())
	go exitSignalHandling(cancel)

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatal(err)
	} else if ctx.Err() != nil {
		log.Info("main context already closed")
		os.Exit(0)
	}

	yag, err := Load(*cfg)
	if err != nil {
		log.Fatal(err)
	} else if ctx.Err() != nil {
		log.Info("main context already closed")
		os.Exit(0)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for {
			select {
			case err := <-yag.consumerErrorChannel:
				if err != nil {
					log.Errorf("consumer: problem while sending to NSCA: %s", err)
				} else {
					log.Info("consumer: message sent!")
				}
			case <-ctx.Done():
				log.Debug("consumer: stop listening for NSCA errors")
				wg.Done()
				return
			}
		}
	}()

	yag.runMainLoop(ctx, &wg)
	log.Info("Received exit signal; stopping jagozzi")
	log.Debug("main: waiting for all goroutines")
	exitCtx, cancelFunc := context.WithTimeout(context.Background(), time.Second*2)
	defer cancelFunc()
	wg.Add(1)
	exitChan := make(chan interface{}, 1)
	go func() {
		wg.Wait()
		exitChan <- true
	}()
	select {
	case <-exitChan:
		log.Debug("subroutines exited")
	case <-exitCtx.Done():
		log.Debug("exit context timeout")
	}
	yag.Unload()
	log.Debug("main: all goroutines exited, exiting main")
}

func (yag Jagozzi) runMainLoop(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(yag.cfg.Periodicity)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			log.Println("Running checkers: ", t)
			loopCtx, cancelFunc := context.WithTimeout(ctx, yag.cfg.Periodicity*time.Duration(2))
			defer cancelFunc()
			for _, checker := range yag.Checkers() {
				go yag.runChecker(loopCtx, checker, wg)
			}
		case <-ctx.Done():
			log.Println("main context closed")
			return
		}
	}
}

func (yag Jagozzi) runChecker(ctx context.Context, checker plugins.Checker, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	result := checker.Run(ctx)

	if ctx.Err() != nil && ctx.Err() == context.Canceled {
		log.Infof("jagozzi: context cancelled while running checker: %s", checker.Name())
		return
	} else if ctx.Err() != nil && ctx.Err() == context.DeadlineExceeded {
		log.Errorf("jagozzi: context timed out while running checker: %s", checker.Name())
		ctxConsumer, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		ctx = ctxConsumer
	}

	log.Debugf("checker: result was %q", result.Message)
	yag.SendConsumers(ctx, result)
}
