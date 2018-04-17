package http

import (
	"bytes"
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
	Timeout  time.Duration `json:"-"`
	Warning  time.Duration `json:"-"`
	Critical time.Duration `json:"-"`
	template *template.Template
}

type rawHTTPConfig struct {
	Type        string `json:"type"`
	URL         string `json:"url" validate:"required"`
	VerifyCACRT bool   `json:"verify_ca_crt"`
	Method      string `json:"method" validate:"required,eq=GET|eq=POST|eq=PUT|eq=DELETE"`
	Code        int64  `json:"code" default:"200"`
	Content     string `json:"content"`
	RawTimeout  int64  `json:"timeout"`
	RawWarning  int64  `json:"warn"`
	RawCritical int64  `json:"crit"`
	Name        string `json:"name" validate:"required"`
	Template    string `json:"template"`
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

	if cfg.Template != "" {
		// testing that we can parse template
		tmpl, err := template.New("httpTemplate").Parse(cfg.Template)
		if err != nil {
			return err
		}

		model := result{
			Cfg:         *cfg,
			Result:      plugins.STATE_CRITICAL,
			Response:    http.Response{},
			Request:     http.Request{},
			ElapsedTime: time.Duration(2) * time.Second,
		}

		// testing that we can apply template to model
		buf := new(bytes.Buffer)
		err = tmpl.Execute(buf, model)
		if err != nil {
			return err
		}

		cfg.template = tmpl
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return err
	}

	return nil
}
