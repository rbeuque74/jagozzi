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

// result is the model used by HTTP checker to apply template on
type result struct {
	Cfg         httpConfig
	Result      plugins.StatusEnum
	Response    http.Response
	Request     http.Request
	ElapsedTime time.Duration
	Err         error
}

func (res result) Error() error {
	return res.Err
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

	elapsedTime := time.Since(duration).Round(time.Millisecond)

	// TODO: remove sensitive information from request such as credentials
	model := result{
		Cfg:         c.cfg,
		Result:      plugins.STATE_CRITICAL,
		Response:    *resp,
		Request:     *req,
		ElapsedTime: elapsedTime,
		Err:         nil,
	}

	if resp.StatusCode != int(c.cfg.Code) {
		model.Err = fmt.Errorf("invalid status code: %d instead of %d", resp.StatusCode, c.cfg.Code)
		return "", plugins.RenderError(c.cfg.template, model)
	}

	if elapsedTime > c.cfg.Critical {
		model.Err = fmt.Errorf("critical timeout: request took %s instead of %s", elapsedTime, c.cfg.Critical.Round(time.Millisecond))
		return "", plugins.RenderError(c.cfg.template, model)
	} else if elapsedTime > c.cfg.Warning {
		model.Err = fmt.Errorf("timeout: request took %s instead of %s", elapsedTime, c.cfg.Warning.Round(time.Millisecond))
		return "", plugins.RenderError(c.cfg.template, model)
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
