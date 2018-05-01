package http

import (
	"time"

	"github.com/rbeuque74/jagozzi/config"
	validator "gopkg.in/go-playground/validator.v9"
	defaults "gopkg.in/mcuadros/go-defaults.v1"
)

type httpConfig struct {
	rawHTTPConfig
	Timeout  time.Duration `json:"-"`
	Warning  time.Duration `json:"-"`
	Critical time.Duration `json:"-"`
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

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return err
	}

	return nil
}
