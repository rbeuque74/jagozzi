package ssl

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/ghodss/yaml"
	"github.com/rbeuque74/jagozzi/plugins"
	log "github.com/sirupsen/logrus"
)

const pluginName = "SSL"

func init() {
	plugins.Register(pluginName, NewSSLChecker)
}

// SSLChecker is a plugin to check SSL certificate expiration date
type SSLChecker struct {
	cfg            sslConfig
	executableName string
}

// Name returns the name of the checker
func (c SSLChecker) Name() string {
	return pluginName
}

// ServiceName returns the name of the NSCA service associated to the checker
func (c SSLChecker) ServiceName() string {
	return c.cfg.Name
}

// Periodicity returns the delay between two checks
func (c SSLChecker) Periodicity() *time.Duration {
	return c.cfg.Periodicity()
}

// Run is performing the checker protocol
func (c *SSLChecker) Run(ctx context.Context) plugins.Result {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", c.cfg.Host, nil)
	if conn != nil {
		defer conn.Close()
	}
	if err != nil {
		return plugins.ResultFromError(c, err, "can't dial host")
	}

	timeNow := time.Now()

	checkedCerts := make(map[string]struct{})
	var leastResultCN string
	var leastResultExpiration time.Duration

	for _, chain := range conn.ConnectionState().VerifiedChains {
		for _, cert := range chain {
			if _, checked := checkedCerts[string(cert.Signature)]; checked {
				continue
			}
			checkedCerts[string(cert.Signature)] = struct{}{}

			if timeNow.After(cert.NotAfter) {
				return plugins.Result{
					Status:  plugins.STATE_CRITICAL,
					Message: fmt.Sprintf("certificate expired: %q", cert.Subject.CommonName),
					Checker: c,
				}
			}

			lastResultCN := cert.Subject.CommonName
			lastResultExpiration := cert.NotAfter.Sub(timeNow).Truncate(time.Minute)

			log.Debugf("certificate %s expires in %s", lastResultCN, Duration{Duration: lastResultExpiration})
			log.Debugf("alternative names %+v", cert.Subject.ExtraNames)

			// Check the expiration.
			if lastResultExpiration < c.cfg.Critical.Duration {
				return plugins.Result{
					Status:  plugins.STATE_CRITICAL,
					Message: fmt.Sprintf("expiration due in %s for %q", Duration{Duration: lastResultExpiration}, cert.Subject.CommonName),
					Checker: c,
				}
			} else if lastResultExpiration < c.cfg.Warning.Duration {
				return plugins.Result{
					Status:  plugins.STATE_WARNING,
					Message: fmt.Sprintf("expiration due in %s for %q", Duration{Duration: lastResultExpiration}, cert.Subject.CommonName),
					Checker: c,
				}
			}

			if leastResultExpiration == 0 || lastResultExpiration < leastResultExpiration {
				leastResultExpiration = lastResultExpiration
				leastResultCN = lastResultCN
			}
		}
	}

	if leastResultExpiration == 0 {
		return plugins.Result{
			Status:  plugins.STATE_CRITICAL,
			Message: fmt.Sprintf("no certificate found for %q", c.cfg.Host),
			Checker: c,
		}
	}

	return plugins.Result{
		Status:  plugins.STATE_OK,
		Message: fmt.Sprintf("%q expires in %s", leastResultCN, Duration{Duration: leastResultExpiration}),
		Checker: c,
	}
}

// NewSSLChecker create a SSL checker
func NewSSLChecker(conf interface{}, pluginConf interface{}) (plugins.Checker, error) {
	out, err := yaml.Marshal(conf)
	if err != nil {
		return nil, err
	}

	cfg := sslConfig{}
	err = yaml.Unmarshal(out, &cfg)
	if err != nil {
		return nil, err
	}

	checker := &SSLChecker{
		cfg: cfg,
	}

	log.Infof("SSL: Checker activated for %q", checker.cfg.Host)
	return checker, nil
}
