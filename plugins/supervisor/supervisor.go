package supervisor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ochinchina/supervisord/process"
	"github.com/ochinchina/supervisord/xmlrpcclient"
	"github.com/rbeuque74/jagozzi/plugins"
	log "github.com/sirupsen/logrus"
)

const pluginName = "Supervisor"

func init() {
	plugins.Register(pluginName, NewSupervisorChecker)
}

// SupervisorChecker is a plugin to check status code of command
type SupervisorChecker struct {
	cfg            checkerConfig
	pluginCfg      pluginConfig
	executableName string
}

// Name returns the name of the checker
func (c SupervisorChecker) Name() string {
	return pluginName
}

// ServiceName returns the name of the NSCA service associated to the checker
func (c SupervisorChecker) ServiceName() string {
	return c.cfg.Name
}

// Periodicity returns the delay between two checks
func (c SupervisorChecker) Periodicity() *time.Duration {
	return c.cfg.Periodicity()
}

// Run is performing the checker protocol
func (c *SupervisorChecker) Run(ctx context.Context) plugins.Result {
	rpcc := xmlrpcclient.NewXmlRPCClient(c.pluginCfg.ServerURL.String())
	username := c.pluginCfg.ServerURL.User.Username()
	password, passwordSet := c.pluginCfg.ServerURL.User.Password()
	if passwordSet && username != "" {
		rpcc.SetUser(username)
		rpcc.SetPassword(password)
	}
	if c.pluginCfg.Timeout > 0 {
		rpcc.SetTimeout(c.pluginCfg.Timeout)
	}

	processesStates, err := rpcc.GetAllProcessInfo()
	if err != nil {
		return plugins.ResultFromError(c, err, "unable to contact supervisor daemon")
	}

	for _, pinfo := range processesStates.Value {
		name := strings.ToLower(pinfo.Name)

		if c.cfg.Service != nil && *c.cfg.Service != name {
			continue
		}

		description := pinfo.Description
		processState := process.ProcessState(pinfo.State)
		if strings.ToLower(description) == "<string></string>" {
			description = ""
		}
		if pinfo.State != process.RUNNING {
			return plugins.Result{
				Status:  plugins.STATE_CRITICAL,
				Message: fmt.Sprintf("Service %q is currently %s: %s", name, processState.String(), description),
				Checker: c,
			}
		} else if c.cfg.Service != nil {
			return plugins.Result{
				Status:  plugins.STATE_OK,
				Message: fmt.Sprintf("Service %q is running: %s", name, description),
				Checker: c,
			}
		}
	}

	if c.cfg.Service != nil {
		return plugins.Result{
			Status:  plugins.STATE_CRITICAL,
			Message: fmt.Sprintf("Service %q not found", *c.cfg.Service),
			Checker: c,
		}
	}

	return plugins.Result{
		Status:  plugins.STATE_OK,
		Message: "All services are running",
		Checker: c,
	}
}

// NewSupervisorChecker create a Supervisor checker
func NewSupervisorChecker(checkerCfg interface{}, pluginCfg interface{}) (plugins.Checker, error) {
	cfg, err := loadConfiguration(checkerCfg)
	if err != nil {
		return nil, fmt.Errorf("supervisor/cfg: %s", err)
	}

	pCfg, err := loadPluginConfiguration(pluginCfg)
	if err != nil {
		return nil, fmt.Errorf("supervisor/pluginCfg: %s", err)
	}

	checker := &SupervisorChecker{
		cfg:       cfg,
		pluginCfg: pCfg,
	}

	log.Infof("supervisor: Checker %q activated", checker.cfg.Type)
	return checker, nil
}
