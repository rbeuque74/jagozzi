package http

import (
	"bytes"
	"errors"
	"net/http"
	"text/template"
	"time"

	"github.com/rbeuque74/jagozzi/config"
	"github.com/rbeuque74/jagozzi/plugins"
	validator "gopkg.in/go-playground/validator.v9"
	defaults "gopkg.in/mcuadros/go-defaults.v1"
)

type httpConfig struct {
	rawHTTPConfig
	Timeout   time.Duration `json:"-"`
	Warning   time.Duration `json:"-"`
	Critical  time.Duration `json:"-"`
	templates templates
}

type rawHTTPConfig struct {
	config.GenericPluginConfiguration
	Type         string       `json:"type"`
	URL          string       `json:"url" validate:"required"`
	VerifyCACRT  bool         `json:"verify_ca_crt"`
	Method       string       `json:"method" validate:"required,eq=GET|eq=POST|eq=PUT|eq=DELETE"`
	Code         int64        `json:"code" default:"200"`
	Content      string       `json:"content"`
	RawTimeout   int64        `json:"timeout"`
	RawWarning   int64        `json:"warn"`
	RawCritical  int64        `json:"crit"`
	RawTemplates rawTemplates `json:"templates"`
}

type rawTemplates struct {
	ErrNewHTTPRequest  string `default:"{{.Err}}"`
	ErrRequest         string `default:"{{.Err}}"`
	ErrStatusCode      string `default:"invalid status code: {{.Response.StatusCode}} instead of {{.Cfg.Code}}"`
	ErrTimeoutCritical string `default:"critical timeout: request took {{.ElapsedTime}} instead of {{.Cfg.Critical}}"`
	ErrTimeoutWarning  string `default:"timeout: request took {{.ElapsedTime}} instead of {{.Cfg.Warning}}"`
}

type templates struct {
	ErrNewHTTPRequest  *template.Template `json:"-"`
	ErrRequest         *template.Template `json:"-"`
	ErrStatusCode      *template.Template `json:"-"`
	ErrTimeoutCritical *template.Template `json:"-"`
	ErrTimeoutWarning  *template.Template `json:"-"`
}

func (cfg *httpConfig) UnmarshalJSON(b []byte) error {
	raw := &rawHTTPConfig{}

	if err := config.UnmarshalConfig(b, raw); err != nil {
		return err
	}

	defaults.SetDefaults(raw)

	cfg.rawHTTPConfig = *raw
	cfg.Timeout = time.Duration(raw.RawTimeout) * time.Millisecond
	cfg.Warning = time.Duration(raw.RawWarning) * time.Millisecond
	cfg.Critical = time.Duration(raw.RawCritical) * time.Millisecond

	var err error
	var tmpl *template.Template
	if tmpl, err = testTemplate("HttpErrNewHTTPRequest", raw.RawTemplates.ErrNewHTTPRequest, true, false, false); err != nil {
		return err
	}
	cfg.templates.ErrNewHTTPRequest = tmpl

	if tmpl, err = testTemplate("HttpErrRequest", raw.RawTemplates.ErrRequest, true, true, false); err != nil {
		return err
	}
	cfg.templates.ErrRequest = tmpl

	if tmpl, err = testTemplate("HttpErrStatusCode", raw.RawTemplates.ErrStatusCode, true, true, true); err != nil {
		return err
	}
	cfg.templates.ErrStatusCode = tmpl

	if tmpl, err = testTemplate("HttpErrTimeoutCritical", raw.RawTemplates.ErrTimeoutCritical, true, true, true); err != nil {
		return err
	}
	cfg.templates.ErrTimeoutCritical = tmpl

	if tmpl, err = testTemplate("HttpErrTimeoutWarning", raw.RawTemplates.ErrTimeoutWarning, true, true, true); err != nil {
		return err
	}
	cfg.templates.ErrTimeoutWarning = tmpl

	validate := validator.New()
	return validate.Struct(cfg)
}

func testTemplate(templateName, stringTemplate string, includeErr, includeRequest, includeResponse bool) (*template.Template, error) {
	// testing that we can parse template
	tmpl, err := template.New(templateName).Parse(stringTemplate)
	if err != nil {
		return nil, err
	}

	model := result{
		Cfg:    httpConfig{},
		Result: plugins.STATE_CRITICAL,
	}
	if includeErr {
		model.Err = errors.New("standard error")
	}
	if includeRequest {
		model.Request = http.Request{}
	}
	if includeResponse {
		model.Response = http.Response{
			Request: &model.Request,
		}
		model.ElapsedTime = time.Duration(2) * time.Second
	}

	// testing that we can apply template to model
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, model)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
