package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	Cfg          httpConfig
	Result       plugins.StatusEnum
	Response     http.Response
	Request      http.Request
	ElapsedTime  time.Duration
	Err          error
	ResponseBody map[string]string
}

func (res result) Error() error {
	return res.Err
}

func (c *HTTPChecker) Run(ctx context.Context) (string, error) {
	httpCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	model := result{
		Cfg:    c.cfg,
		Result: plugins.STATE_CRITICAL,
		Err:    nil,
	}

	req, err := http.NewRequest(c.cfg.Method, c.cfg.URL, nil)
	if err != nil {
		model.Err = err
		return "", fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrNewHTTPRequest, model))
	}
	req = req.WithContext(httpCtx)

	model.Request = *req

	duration := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		model.Err = err
		return "", fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrRequest, model))
	}
	defer resp.Body.Close()

	elapsedTime := time.Since(duration).Round(time.Millisecond)

	model.Response = *resp
	model.ElapsedTime = elapsedTime
	// remove Authorisation header to prevent credentials leak
	model.Request.Header.Del("Authorization")

	// try to unmarshal body to map[string] to provide more context for error templating
	if bodyStr, err := ioutil.ReadAll(resp.Body); err == nil {
		responseBody := make(map[string]string)
		if err := json.Unmarshal(bodyStr, &responseBody); err == nil {
			model.ResponseBody = responseBody
		}
	}

	if resp.StatusCode != int(c.cfg.Code) {
		model.Err = fmt.Errorf("invalid status code: %d instead of %d", resp.StatusCode, c.cfg.Code)
		return "", fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrStatusCode, model))
	}

	if elapsedTime > c.cfg.Critical {
		model.Err = fmt.Errorf("critical timeout: request took %s instead of %s", elapsedTime, c.cfg.Critical.Round(time.Millisecond))
		return "", fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrTimeoutCritical, model))
	} else if elapsedTime > c.cfg.Warning {
		model.Err = fmt.Errorf("timeout: request took %s instead of %s", elapsedTime, c.cfg.Warning.Round(time.Millisecond))
		return "", fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrTimeoutWarning, model))
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
