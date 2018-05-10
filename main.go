package main

import (
	"context"
	"flag"
	"os"
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
	applyLogLevel(logLevel)

	ctx, cancel := context.WithCancel(context.Background())
	go exitSignalHandling(cancel)
	go exitTimeout(ctx)

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

	go yag.ListenForConsumersError(ctx, &wg)

	yag.runMainLoop(ctx, &wg)
	log.Info("Received exit signal; stopping jagozzi")

	log.Debug("jagozzi: waiting for all goroutines")
	wg.Wait()
	log.Debug("jagozzi: subroutines exited")

	yag.Unload()
	log.Debug("jagozzi: unloading complete; exit successful")
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
