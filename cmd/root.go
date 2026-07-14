package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

// version/commit/buildDate are overridden at build time via
//
//	-ldflags "-X github.com/gangqiang01/EIM-SKILL/cmd.version=... \
//	          -X github.com/gangqiang01/EIM-SKILL/cmd.commit=... \
//	          -X github.com/gangqiang01/EIM-SKILL/cmd.buildDate=..."
//
// (see the Makefile) - these defaults only show up for a plain
// "go build"/"go run" done outside that build path.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "eim-skill",
	Short: "Command-line skill tools for DeviceOn-EIM device management",
	Long: `eim-skill wraps DeviceOn-EIM's device management REST API as a set
of command-line tools: querying and controlling devices (reboot, shutdown,
volume, backlight, monitoring/alerting), device groups, operation-record
audit logs, agent log retrieval, and OTA package deployment.

eim-skill 把 DeviceOn-EIM 的设备管理 REST API 封装成一组命令行
工具：设备查询与控制（重启、关机、调音量、调背光、监控与告警）、
设备组管理、操作审计日志、agent 日志获取、以及 OTA 软件包部署。

Every command prints exactly one JSON object to stdout:
  {"status": 200, "msg": "", "data": ...}   on success 成功
  {"status": 500, "msg": "..."}             on failure 失败
with a matching process exit code (0/1) - safe to script against.

每个命令都只向 stdout 输出一个 JSON 对象（如上），并附带对应的进程
退出码（0/1），方便脚本化调用和判断成败。

Run "eim-skill <command> --help" (or "eim-skill <command> <subcommand>
--help") for details, parameters, and example output of any specific
command.

运行 "eim-skill <command> --help"（或更深一层的子命令 --help）可以看到
该命令的详细说明、参数含义和示例输出。

Top-level command groups 顶层命令分组:
  device   设备查询与控制（含监控、阈值告警、USB、Docker等）
  group    设备组查询与创建
  log      操作审计日志、agent 客户端日志
  ota      OTA 软件包部署（Linux/Android/Windows/Docker/通用文件）
  mcp      以 MCP (Model Context Protocol) 服务器模式运行`,
	Version: version,
	// Every command reports its own success/failure via the unified JSON
	// envelope (see output.go) - suppress cobra's default "Error: ..." +
	// usage-text printing so stdout always carries exactly one shape.
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("eim-skill version {{.Version}}\ncommit:  %s\nbuilt:   %s\n", commit, buildDate))
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "conf/config.yaml", "config file path (配置文件路径)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "AppHub/DeviceLink server base URL (overrides config file, e.g. http://127.0.0.1:30010) - 服务器地址（覆盖配置文件）")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "login username (overrides config file) - 登录用户名（覆盖配置文件）")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "login password (overrides config file) - 登录密码（覆盖配置文件）")
	rootCmd.PersistentFlags().StringVar(&repoBaseURL, "repo-base-url", "", "repo server (m2m-management-edge) base URL (overrides config file, e.g. http://127.0.0.1:30001) - 软件包仓库服务地址（覆盖配置文件）")
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

// lastDurationWindow parses a "--last" flag value (e.g. "30m", "12h", "3d")
// into a [beginTs, endTs] Unix-millisecond window ending now, for commands
// that filter by a relative recent time range instead of raw timestamps. An
// empty value means "no time filter" (both return values are 0).
func lastDurationWindow(last string) (beginTs, endTs int64, err error) {
	if last == "" {
		return 0, 0, nil
	}
	dur, err := parseLastDuration(last)
	if err != nil {
		return 0, 0, err
	}
	now := time.Now()
	return now.Add(-dur).UnixMilli(), now.UnixMilli(), nil
}

// parseLastDuration parses a single number+unit duration like "30m", "12h",
// "3d". Units s/m/h delegate to time.ParseDuration; "d" (days) is an
// eim-skill extension since ParseDuration doesn't support it.
func parseLastDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "d"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid --last value %q (expected e.g. \"30m\", \"12h\", \"3d\"): %w", s, err)
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --last value %q (expected e.g. \"30m\", \"12h\", \"3d\"): %w", s, err)
	}
	return dur, nil
}
