package plugins

import (
	"context"
	"errors"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

var checkerFactories = make(map[string]CheckerFactory)
var launchLog sync.Once

// UnknownCheckerTypeErr is the error returned when the factory can't create a checker because type is not registered
var UnknownCheckerTypeErr = errors.New("Unknown checker name")

// WithServiceName concerns configuration type that is capable to perform a lookup on a ServiceName
type WithServiceName interface {
	ServiceName() string
}

type Checker interface {
	WithServiceName
	Name() string
	Run(context.Context) (string, error)
}

type CheckerFactory func(checkerCfg interface{}, pluginCfg interface{}) (Checker, error)

func Register(name string, factory CheckerFactory) {
	if factory == nil {
		log.Panicf("Checker factory %s does not exist.", name)
	}
	_, registered := checkerFactories[name]
	if registered {
		log.Errorf("Checker factory %s already registered. Ignoring.", name)
	}
	checkerFactories[name] = factory
}

func getCheckersName() []string {
	var keys []string
	for key := range checkerFactories {
		keys = append(keys, key)
	}
	return keys
}

func CreateChecker(name string, checkerCfg interface{}, pluginCfg interface{}) (Checker, error) {
	launchLog.Do(func() {
		log.Debugf("Availables checkers: %s", strings.Join(getCheckersName(), ", "))
	})

	engineFactory, ok := checkerFactories[name]
	if !ok {
		// Factory has not been registered.
		// Make a list of all available datastore factories for logging.
		availables := make([]string, len(checkerFactories))
		for k, _ := range checkerFactories {
			availables = append(availables, k)
		}
		return nil, UnknownCheckerTypeErr
	}

	// Run the factory with the configuration.
	return engineFactory(checkerCfg, pluginCfg)
}
