package cmd

import (
	"fmt"

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
	// Every command reports its own success/failure via the unified JSON
	// envelope (see output.go) - suppress cobra's default "Error: ..." +
	// usage-text printing so stdout always carries exactly one shape.
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the root command, printing any failure in the unified
// {"status":500,"msg":"..."} envelope. It returns the process exit code:
// 0 on success, 1 on failure.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		printErrorResult(err)
		return 1
	}
	return 0
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

// resolveDeviceEndpoint finds the device matching identifier - its
// endpoint id, or failing that, its device name (devName) - and returns
// its endpoint id. This lets every device-targeting command accept either
// form, since the underlying API only operates on endpoint ids.
func resolveDeviceEndpoint(client *apphub.Client, identifier string) (string, error) {
	devices, err := client.ListDevices()
	if err != nil {
		return "", err
	}

	for _, d := range devices {
		if d.Endpoint == identifier {
			return d.Endpoint, nil
		}
	}
	for _, d := range devices {
		if d.DevName == identifier {
			return d.Endpoint, nil
		}
	}
	return "", fmt.Errorf("no device found with endpoint or name %q", identifier)
}
