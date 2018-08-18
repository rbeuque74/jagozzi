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
	_ "github.com/rbeuque74/jagozzi/plugins/ssl"
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
	exiting := exitSignalHandling(cancel)
	go exitTimeout(ctx)

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatal(err)
	} else if ctx.Err() != nil {
		log.Info("main context already closed")
		os.Exit(0)
	}

	jag, err := Load(*cfg)
	if err != nil {
		log.Fatal(err)
	} else if ctx.Err() != nil {
		log.Info("main context already closed")
		os.Exit(0)
	}

	var wg sync.WaitGroup

	jag.runMainLoop(ctx, &wg)
	if !*oneShot {
		<-exiting
	}

	log.Debug("jagozzi: waiting for all goroutines")
	wg.Wait()
	log.Debug("jagozzi: subroutines exited")

	jag.Unload()
	log.Debug("jagozzi: unloading complete; exit successful")
}

func (jag Jagozzi) runMainLoop(ctx context.Context, wg *sync.WaitGroup) {
	checkersPerPeridicity := map[time.Duration][]plugins.Checker{}
	for _, checker := range jag.Checkers() {
		periodicity := jag.cfg.Periodicity
		if p := checker.Periodicity(); p != nil {
			periodicity = *p
		}

		var currentArray []plugins.Checker
		if array, ok := checkersPerPeridicity[periodicity]; ok {
			currentArray = array
		}
		currentArray = append(currentArray, checker)
		checkersPerPeridicity[periodicity] = currentArray
	}

	for loopPeriodicity, loopCheckers := range checkersPerPeridicity {
		periodicity := loopPeriodicity
		checkers := loopCheckers

		wg.Add(1)
		go func() {
			defer wg.Done()
			timeout := periodicity * time.Duration(2)
			cancellationTimeout := timeout + time.Second

			ticker := time.NewTicker(periodicity)
			defer ticker.Stop()
			log := log.WithField("periodicity", periodicity.String())
			log.Debug("loop: starting")

			for {
				select {
				case t := <-ticker.C:
					log.Debugf("triggered: %s", t)
					// context should never be cancelled as we are canceling the parent
					loopCtx, cancelCtx := context.WithTimeout(ctx, timeout)
					time.AfterFunc(cancellationTimeout, func() {
						cancelCtx()
					})

					for _, checker := range checkers {
						wg.Add(1)
						go jag.runChecker(loopCtx, checker, wg)
					}
				case <-ctx.Done():
					log.Debug("loop: main context closed, exiting")
					return
				}
				if *oneShot {
					log.Debug("loop: one shot activated, exiting")
					return
				}
			}
		}()
	}
}

func (jag Jagozzi) runChecker(ctx context.Context, checker plugins.Checker, wg *sync.WaitGroup) {
	defer wg.Done()
	log := log.WithFields(log.Fields{"name": checker.Name(), "serviceName": checker.ServiceName()})
	log.Debugf("perform check")
	result := checker.Run(ctx)

	if ctx.Err() != nil && ctx.Err() == context.Canceled {
		log.Debug("jagozzi: context cancelled while running checker")
		return
	} else if ctx.Err() != nil && ctx.Err() == context.DeadlineExceeded {
		log.Errorf("jagozzi: context timed out while running checker: %s", checker.Name())
	}

	log.Debugf("checker: result was %q", result.Message)
	jag.SendConsumers(result)
}
