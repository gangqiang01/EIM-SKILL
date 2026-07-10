// Package config loads connection settings for the AppHub/DeviceLink server
// that the eim-skill CLI talks to.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the settings needed to reach and authenticate against the
// DeviceLink/AppHub server and its companion repo server.
type Config struct {
	BaseURL        string `mapstructure:"base_url"`
	Username       string `mapstructure:"username"`
	Password       string `mapstructure:"password"`
	RepoBaseURL    string `mapstructure:"repo_base_url"`
	IthingsBaseURL string `mapstructure:"ithings_base_url"`
}

// Load reads configuration from (in increasing priority order): defaults,
// the config file at configPath (if it exists), and EIM_* environment
// variables (EIM_BASE_URL, EIM_USERNAME, EIM_PASSWORD, EIM_REPO_BASE_URL).
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetDefault("base_url", "http://127.0.0.1:30010")
	v.SetDefault("username", "admin")
	v.SetDefault("password", "admin")
	// m2m-management-edge, the repo server behind the "software repository"
	// used to look up OTA-able packages. It runs alongside DeviceLink and
	// accepts the same accesstoken issued by base_url's /login.
	v.SetDefault("repo_base_url", "http://127.0.0.1:30001")
	// ithings, the microservice behind sub-device (Modbus/OPC UA/IEC104/S7/
	// SNMP gateway) management and AMQP/MQTT/Kafka data forwarding. It runs
	// alongside DeviceLink and accepts the same accesstoken as base_url.
	v.SetDefault("ithings_base_url", "http://127.0.0.1:8082")

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
	return &cfg, nil
}
