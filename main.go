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

// nolint: gochecknoglobals
var (
	configFile  = flag.String("cfg", "./jagozzi.yml", "path to config file")
	logLevel    = flag.String("level", "info", "verbosity level for application logs")
	guiConsumer = flag.Bool("display", false, "set display to true if GUI need to display plugins results")
	oneShot     = flag.Bool("oneShot", false, "run jagozzi one time, no periodic checker (useful when used on cron)")
	version     string
)

func main() {
	flag.Parse()
	applyLogLevel(logLevel)
	log.Infof("jagozzi - %s", version)

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

	tearDown := yag.runMainLoop(ctx, &wg)
	log.Info("Received exit signal; stopping jagozzi")

	log.Debug("jagozzi: waiting for all goroutines")
	wg.Wait()
	log.Debug("jagozzi: subroutines exited")
	tearDown()

	yag.Unload()
	log.Debug("jagozzi: unloading complete; exit successful")
}

func (yag Jagozzi) runMainLoop(ctx context.Context, wg *sync.WaitGroup) func() {
	ticker := time.NewTicker(yag.cfg.Periodicity)
	defer ticker.Stop()
	var cancelFuncs []context.CancelFunc
	var first time.Time

	for {
		select {
		case t := <-ticker.C:
			var since time.Duration
			if first.IsZero() {
				first = t
			} else {
				since = time.Since(first).Truncate(time.Millisecond)
			}
			log.Println("Running checkers: ", t)
			loopCtx, cancelFunc := context.WithTimeout(ctx, yag.cfg.Periodicity*time.Duration(2))
			cancelFuncs = append(cancelFuncs, cancelFunc)
			for _, checker := range yag.Checkers() {
				if checker.Periodicity() != nil && since.Nanoseconds()%checker.Periodicity().Nanoseconds() != 0 {
					log.WithField("checker", checker.Name()).Debugf("skipped as periodicity not aligned: %s", since)
					continue
				}
				wg.Add(1)
				go yag.runChecker(loopCtx, checker, wg)
			}
		case <-ctx.Done():
			log.Println("main context closed")
			return tearDownLoop(cancelFuncs)
		}
		if *oneShot {
			return tearDownLoop(cancelFuncs)
		}

	}
}

func (yag Jagozzi) runChecker(ctx context.Context, checker plugins.Checker, wg *sync.WaitGroup) {
	defer wg.Done()
	result := checker.Run(ctx)

	if ctx.Err() != nil && ctx.Err() == context.Canceled {
		log.Infof("jagozzi: context cancelled while running checker: %s", checker.Name())
		return
	} else if ctx.Err() != nil && ctx.Err() == context.DeadlineExceeded {
		log.Errorf("jagozzi: context timed out while running checker: %s", checker.Name())
	}

	log.Debugf("checker: result was %q", result.Message)
	yag.SendConsumers(result)
}
