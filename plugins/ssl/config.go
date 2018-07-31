package ssl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rbeuque74/jagozzi/config"
	validator "gopkg.in/go-playground/validator.v9"
)

type rawSSLConfig struct {
	config.GenericPluginConfiguration
	Host     string   `json:"host" validate:"required"`
	Warning  Duration `json:"warn"`
	Critical Duration `json:"crit"`
}

type sslConfig struct {
	rawSSLConfig
}

func (cfg *sslConfig) UnmarshalJSON(b []byte) error {
	raw := &rawSSLConfig{}

	if err := config.UnmarshalConfig(b, raw); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(raw); err != nil {
		return err
	}

	cfg.rawSSLConfig = *raw

	if !strings.Contains(cfg.Host, ":") {
		// adding default https port
		cfg.Host = cfg.Host + ":443"
	}

	return nil
}

var validBigDuration = regexp.MustCompile(`^(\d+)(d|mo)$`)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	strvalue := strings.Trim(string(b), `"`)
	if validBigDuration.MatchString(strvalue) {
		results := validBigDuration.FindAllStringSubmatch(strvalue, 1)
		v, unit := results[0][1], results[0][2]

		var value int64
		value, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return err
		}
		if unit == "d" {
			// day
			d.Duration = time.Duration(value) * time.Duration(24) * time.Hour
		} else {
			// month
			d.Duration = time.Duration(value) * time.Duration(24*30) * time.Hour
		}
		return
	}
	d.Duration, err = time.ParseDuration(strvalue)
	return
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func (d Duration) String() string {
	var str string
	if d.Duration > (time.Duration(24*30) * time.Hour) {
		months := int64(d.Duration) / (int64(time.Hour) * 24 * 30)
		d.Duration = d.Duration % (time.Duration(24*30) * time.Hour)
		str = str + fmt.Sprintf("%dmonths", months)
	}
	if d.Duration > (time.Duration(24) * time.Hour) {
		days := int64(d.Duration) / (int64(time.Hour) * 24)
		d.Duration = d.Duration % (time.Duration(24) * time.Hour)
		str = str + fmt.Sprintf("%dd", days)
	}

	return str + d.Duration.String()
}
