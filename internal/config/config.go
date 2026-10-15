// Package config loads connection settings for the AppHub/DeviceLink server
// that the eim-skill CLI talks to.
package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the settings needed to reach and authenticate against the
// DeviceOn-EIM server and its companion repo and ithings (sub-device/twin)
// servers.
type Config struct {
	BaseURL        string `mapstructure:"base_url"`
	Username       string `mapstructure:"username"`
	Password       string `mapstructure:"password"`
	RepoBaseURL    string `mapstructure:"repo_base_url"`
	IthingsBaseURL string `mapstructure:"ithings_base_url"`
	// CACertFile, if set, is a PEM file with one or more CA certificates to
	// trust in addition to the system's default trust store. Use this when
	// base_url/repo_base_url/ithings_base_url are served with a certificate
	// signed by an internal/private CA that isn't in the OS trust store.
	CACertFile string `mapstructure:"ca_cert_file"`
	// InsecureSkipVerify disables TLS certificate verification entirely.
	// Takes precedence over CACertFile. This defeats protection against
	// man-in-the-middle attacks - only use it against a network path you
	// already trust (e.g. a direct connection to a known internal server).
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

// Load reads configuration from (in increasing priority order): defaults,
// the config file at configPath (if it exists), and EIM_* environment
// variables (EIM_BASE_URL, EIM_USERNAME, EIM_PASSWORD, EIM_REPO_BASE_URL,
// EIM_ITHINGS_BASE_URL). If repo_base_url/ithings_base_url end up unset
// after that, each is derived from base_url instead of a fixed default -
// see deriveCompanionURL.
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetDefault("base_url", "http://127.0.0.1:30010")
	v.SetDefault("username", "admin")
	v.SetDefault("password", "admin")

	v.SetEnvPrefix("EIM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("config: reading %s: %w", configPath, err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	// m2m-management-edge, the repo server behind OTA package storage, and
	// apphub-ithings, the sub-device/digital-twin server - both run
	// alongside the main DeviceLink server (base_url) on their own port and
	// accept the same accesstoken issued by base_url's /login. When not set
	// explicitly, derive each from base_url (same scheme+host, its own
	// port) rather than assuming a fixed host - a bare port-only default
	// silently mismatched base_url's scheme (e.g. base_url on https with
	// ithings_base_url left at a hardcoded http://127.0.0.1:8082),
	// producing a confusing "plain HTTP request sent to HTTPS port" error.
	if cfg.RepoBaseURL == "" {
		derived, err := deriveCompanionURL(cfg.BaseURL, "30001")
		if err != nil {
			return nil, fmt.Errorf("config: deriving repo_base_url from base_url %q: %w", cfg.BaseURL, err)
		}
		cfg.RepoBaseURL = derived
	}
	if cfg.IthingsBaseURL == "" {
		derived, err := deriveCompanionURL(cfg.BaseURL, "8082")
		if err != nil {
			return nil, fmt.Errorf("config: deriving ithings_base_url from base_url %q: %w", cfg.BaseURL, err)
		}
		cfg.IthingsBaseURL = derived
	}

	return &cfg, nil
}

// deriveCompanionURL builds a companion server URL from baseURL - same
// scheme and hostname, with its port replaced by port.
func deriveCompanionURL(baseURL, port string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	u.Host = u.Hostname() + ":" + port
	return u.String(), nil
}

// NewHTTPClient builds the *http.Client every server client (AppHub, repo,
// ithings) should use. It applies to all three base URLs since they're
// typically the same host on different ports.
//
// If cfg.InsecureSkipVerify is set, TLS certificate verification is
// disabled entirely (takes precedence over CACertFile). Otherwise, if
// cfg.CACertFile is set, the named PEM file's certificates are trusted in
// addition to the system's default trust store.
func NewHTTPClient(cfg *Config) (*http.Client, error) {
	if cfg.InsecureSkipVerify {
		return &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}, nil
	}

	if cfg.CACertFile == "" {
		return &http.Client{Timeout: 30 * time.Second}, nil
	}

	pemBytes, err := os.ReadFile(cfg.CACertFile)
	if err != nil {
		return nil, fmt.Errorf("config: reading ca_cert_file %s: %w", cfg.CACertFile, err)
	}

	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("config: ca_cert_file %s contains no valid PEM certificates", cfg.CACertFile)
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}, nil
}
