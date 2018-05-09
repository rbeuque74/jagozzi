package marathon

import (
	"context"
	"fmt"
	"net/http"
	"time"

	marathonlib "github.com/gambol99/go-marathon"
	"github.com/rbeuque74/jagozzi/plugins"
	log "github.com/sirupsen/logrus"
)

const pluginName = "Marathon"

// MarathonChecker is a plugin to check Marathon infrastructure
type MarathonChecker struct {
	cfg          checkerConfig
	pluginCfg    pluginConfig
	client       marathonlib.Marathon
	roundtripper *httproundtripper
	staggedtasks map[string]time.Time
	exitedtasks  []time.Time
}

func init() {
	plugins.Register(pluginName, NewMarathonChecker)
}

// NewMarathonChecker create a Marathon checker
func NewMarathonChecker(checkerCfg interface{}, pluginCfg interface{}) (plugins.Checker, error) {
	cfg, err := loadConfiguration(checkerCfg)
	if err != nil {
		return nil, fmt.Errorf("marathon/cfg: %s", err)
	}

	pCfg, err := loadPluginConfiguration(pluginCfg)
	if err != nil {
		return nil, fmt.Errorf("marathon/pluginCfg: %s", err)
	}

	marathonCfg := marathonlib.NewDefaultConfig()
	marathonCfg.URL = pCfg.Host
	if pCfg.User != "" && pCfg.Password != "" {
		marathonCfg.HTTPBasicAuthUser = pCfg.User
		marathonCfg.HTTPBasicPassword = pCfg.Password
	}
	roundtripper := &httproundtripper{}
	marathonCfg.HTTPClient = &http.Client{
		Transport: roundtripper,
	}
	client, err := marathonlib.NewClient(marathonCfg)
	if err != nil {
		return nil, fmt.Errorf("marathon: %s", err)
	}

	log.Infof("marathon: Checker %q activated for application %q (warn: %d, crit; %d)", cfg.Type, cfg.ID, cfg.Warning, cfg.Critical)
	return &MarathonChecker{
		cfg:          cfg,
		pluginCfg:    pCfg,
		client:       client,
		roundtripper: roundtripper,
	}, nil
}

// Name returns the name of the checker
func (c *MarathonChecker) Name() string {
	return pluginName
}

// ServiceName returns the name of the NSCA service associated to the checker
func (c MarathonChecker) ServiceName() string {
	return c.cfg.ServiceName()
}

type httproundtripper struct {
	ctx *context.Context
}

func (rt httproundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.WithContext(*rt.ctx)
	return http.DefaultTransport.RoundTrip(req)
}

// Run is performing the checker protocol
func (c *MarathonChecker) Run(ctx context.Context) (string, error) {
	appID := c.cfg.ID
	c.roundtripper.ctx = &ctx
	app, err := c.client.Application(appID)
	if err != nil {
		log.Warnf("marathon/err: %s", err)
		return "KO", err
	}

	log.WithFields(log.Fields{"healthy": app.TasksHealthy, "running": app.TasksRunning, "staged": app.TasksStaged, "unhealthy": app.TasksUnhealthy}).Info(app.ID)
	running := int64(app.TasksRunning)
	if running < c.cfg.Critical {
		return "KO", fmt.Errorf("%d/%d instances running, threshold: %d", running, *app.Instances, c.cfg.Critical)
	} else if running < c.cfg.Warning {
		return "KO", fmt.Errorf("%d/%d instances running, threshold: %d", running, *app.Instances, c.cfg.Warning)
	} else if running != 0 && running == int64(app.TasksUnhealthy) {
		return "KO", fmt.Errorf("%d unhealthy; %d/%d healthy instances running", app.TasksUnhealthy, (app.TasksRunning - app.TasksUnhealthy), *app.Instances)
	}

	log.Infof("%d instances found for %s", running, appID)

	if res, err := c.runStaggedTasks(ctx, *app); err != nil {
		return res, err
	}

	if res, err := c.runExitedTasks(ctx); err != nil {
		return res, err
	}

	return fmt.Sprintf("OK: %d running; %d unhealthy; %d staged", (app.TasksRunning - app.TasksUnhealthy), app.TasksUnhealthy, app.TasksStaged), nil
}

func (c *MarathonChecker) runStaggedTasks(ctx context.Context, app marathonlib.Application) (string, error) {
	tasks := app.Tasks

	for _, taskPtr := range tasks {
		if taskPtr == nil {
			continue
		}

		task := *taskPtr
		staggedDate, err := parseMarathonDateTime(task.StagedAt)
		if err != nil {
			log.Error(err)
			continue
		}
		startedDate, err := parseMarathonDateTime(task.StartedAt)
		if err != nil {
			log.Error(err)
			continue
		}

		if !startedDate.IsZero() {
			continue
		}

		staggedSince := time.Since(staggedDate)
		fifteenMinute := 15 * time.Minute
		if staggedSince > fifteenMinute {
			return "KO", fmt.Errorf("task stagged since 15 minutes")
		}
	}

	return "OK", nil
}

func (c *MarathonChecker) runExitedTasks(ctx context.Context) (string, error) {
	return "OK", nil
}

func parseMarathonDateTime(value string) (time.Time, error) {
	var date time.Time
	if value == "" {
		return date, nil
	}
	return time.Parse(time.RFC3339, value)
}
