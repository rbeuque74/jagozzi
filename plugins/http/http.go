package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
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
	cfg    httpConfig
	client *http.Client
}

// Name returns the name of the checker
func (c HTTPChecker) Name() string {
	return pluginName
}

// ServiceName returns the name of the NSCA service associated to the checker
func (c HTTPChecker) ServiceName() string {
	return c.cfg.Name
}

// Periodicity returns the delay between two checks
func (c HTTPChecker) Periodicity() *time.Duration {
	return c.cfg.Periodicity()
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

// Run is performing the checker protocol
func (c *HTTPChecker) Run(ctx context.Context) plugins.Result {
	model := result{
		Cfg:    c.cfg,
		Result: plugins.STATE_CRITICAL,
		Err:    nil,
	}

	req, err := http.NewRequest(c.cfg.Method, c.cfg.URL, nil)
	if err != nil {
		model.Err = err
		err = fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrNewHTTPRequest, model))
		return plugins.ResultFromError(c, err, "")
	}
	req = req.WithContext(ctx)

	model.Request = *req

	duration := time.Now()
	client := &http.Client{}
	if c.client != nil {
		client = c.client
	}
	client.Timeout = c.cfg.Timeout
	resp, err := client.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			model.Err = err
			model.ElapsedTime = c.cfg.Timeout
			err = fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrTimeoutCritical, model))
			return plugins.ResultFromError(c, err, "")
		}
		model.Err = err
		err = fmt.Errorf(plugins.RenderError(c.cfg.templates.ErrRequest, model))
		return plugins.ResultFromError(c, err, "")
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

		return plugins.Result{
			Checker: c,
			Message: plugins.RenderError(c.cfg.templates.ErrStatusCode, model),
			Status:  plugins.STATE_CRITICAL,
		}
	}

	if elapsedTime > c.cfg.Critical {
		model.Err = fmt.Errorf("critical timeout: request took %s instead of %s (%s)", elapsedTime, c.cfg.Critical.Round(time.Millisecond), resp.Status)

		return plugins.Result{
			Checker: c,
			Message: plugins.RenderError(c.cfg.templates.ErrTimeoutCritical, model),
			Status:  plugins.STATE_CRITICAL,
		}
	} else if elapsedTime > c.cfg.Warning {
		model.Err = fmt.Errorf("timeout: request took %s instead of %s (%s)", elapsedTime, c.cfg.Warning.Round(time.Millisecond), resp.Status)

		return plugins.Result{
			Checker: c,
			Message: plugins.RenderError(c.cfg.templates.ErrTimeoutWarning, model),
			Status:  plugins.STATE_WARNING,
		}
	}
	return plugins.Result{
		Checker: c,
		Message: fmt.Sprintf("%s - %s elapsed", resp.Status, elapsedTime),
		Status:  plugins.STATE_OK,
	}
}

// NewHTTPChecker create a HTTP checker
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
