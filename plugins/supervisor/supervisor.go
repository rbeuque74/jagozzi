package supervisor

import (
	"context"
	"fmt"
	"strings"

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

func (c *SupervisorChecker) Name() string {
	return pluginName
}

func (c SupervisorChecker) ServiceName() string {
	return c.cfg.Name
}

func (c *SupervisorChecker) Run(ctx context.Context) (string, error) {
	rpcc := xmlrpcclient.NewXmlRPCClient(c.pluginCfg.ServerURL.String())
	username := c.pluginCfg.ServerURL.User.Username()
	password, passwordSet := c.pluginCfg.ServerURL.User.Password()
	if passwordSet && username != "" {
		rpcc.SetUser(username)
		rpcc.SetPassword(password)
	}

	processesStates, err := rpcc.GetAllProcessInfo()
	if err != nil {
		return "KO", fmt.Errorf("unable to contact supervisor daemon: %s", err)
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
			return "KO", fmt.Errorf("Service %s is currently %s: %s", name, processState.String(), description)
		} else if c.cfg.Service != nil {
			return fmt.Sprintf("Service %s is running: %s", name, description), nil
		}
	}

	return "All services are RUNNING", nil
}

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
