package cmd

import (
	"fmt"
	"os"

	"github.com/gangqiang01/EIM-SKILL/internal/apphub"
	"github.com/gangqiang01/EIM-SKILL/internal/config"
	"github.com/gangqiang01/EIM-SKILL/internal/repo"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	baseURL     string
	username    string
	password    string
	repoBaseURL string
)

var rootCmd = &cobra.Command{
	Use:     "eim-skill",
	Short:   "Command-line skill tools for DeviceLink/AppHub device management",
	Long:    `eim-skill wraps DeviceLink's device management REST API as a set of command-line tools: listing devices, managing device groups, and controlling individual devices (reboot, shutdown, volume, backlight).`,
	Version: "0.1.0",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "conf/config.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "AppHub/DeviceLink server base URL (overrides config file, e.g. http://127.0.0.1:30010)")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "login username (overrides config file)")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "login password (overrides config file)")
	rootCmd.PersistentFlags().StringVar(&repoBaseURL, "repo-base-url", "", "repo server (m2m-management-edge) base URL (overrides config file, e.g. http://127.0.0.1:30001)")
}

// loadConfig loads configuration (file + env + flags, in increasing priority).
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, err
	}

	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if username != "" {
		cfg.Username = username
	}
	if password != "" {
		cfg.Password = password
	}
	if repoBaseURL != "" {
		cfg.RepoBaseURL = repoBaseURL
	}

	return cfg, nil
}

// newClient loads configuration and returns a ready-to-use AppHub client.
func newClient() (*apphub.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return apphub.NewClient(cfg.BaseURL, cfg.Username, cfg.Password), nil
}

// newRepoClient loads configuration and returns a repo server client that
// authenticates by reusing appClient's AppHub session.
func newRepoClient(appClient *apphub.Client) (*repo.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return repo.NewClient(cfg.RepoBaseURL, appClient), nil
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
