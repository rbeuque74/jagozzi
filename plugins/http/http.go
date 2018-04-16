package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ghodss/yaml"
	"github.com/rbeuque74/jagozzi/plugins"
	log "github.com/sirupsen/logrus"
)

const pluginName = "HTTP"

func init() {
	plugins.Register(pluginName, NewHTTPChecker)
}

// HTTPChecker is a plugin to check HTTP service
type HTTPChecker struct {
	cfg httpConfig
}

func (c *HTTPChecker) Name() string {
	return pluginName
}

func (c HTTPChecker) ServiceName() string {
	return c.cfg.Name
}

func (c *HTTPChecker) Run(ctx context.Context) (string, error) {
	httpCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	req, err := http.NewRequest(c.cfg.Method, c.cfg.URL, nil)
	if err != nil {
		return "", err
	}
	req = req.WithContext(httpCtx)

	duration := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != int(c.cfg.Code) {
		return "", fmt.Errorf("invalid status code: %d instead of %d", resp.StatusCode, c.cfg.Code)
	}

	elapsedTime := time.Since(duration)
	if elapsedTime > c.cfg.Critical {
		return "", fmt.Errorf("critical")
	} else if elapsedTime > c.cfg.Warning {
		return "", fmt.Errorf("warning")
	}
	return "OK", nil
}

func NewHTTPChecker(conf interface{}, pluginConf interface{}) (plugins.Checker, error) {
	out, err := yaml.Marshal(conf)
	if err != nil {
		return nil, err
	}

	checks := httpConfig{}
	err = yaml.Unmarshal(out, &checks)
	if err != nil {
		return nil, err
	}

	log.Infof("http: Checker activated for %s %q", checks.Method, checks.URL)
	return &HTTPChecker{
		cfg: checks,
	}, nil
}
