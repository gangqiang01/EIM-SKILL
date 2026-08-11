package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gangqiang01/EIM-SKILL/internal/apphub"
	"github.com/gangqiang01/EIM-SKILL/internal/config"
	"github.com/gangqiang01/EIM-SKILL/internal/ithings"
	"github.com/gangqiang01/EIM-SKILL/internal/repo"
	"github.com/spf13/cobra"
)

var (
	cfgFile        string
	baseURL        string
	username       string
	password       string
	repoBaseURL    string
	ithingsBaseURL string
	caCertFile     string
	tlsInsecure    bool
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
volume, backlight, monitoring/alerting), device groups, system/operation
logs, OTA package deployment, and server-level configuration.

eim-skill 把 DeviceOn-EIM 的设备管理 REST API 封装成一组命令行
工具：设备查询与控制（重启、关机、调音量、调背光、监控与告警）、
设备组管理、系统日志/操作审计日志、OTA 软件包部署、以及服务器级别
配置。

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
  log      设备系统日志、操作审计日志
  ota      OTA 软件包部署（Linux/Android/Windows/Docker/通用文件）
  server   服务器级别配置（不针对具体设备）
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

// requireArgs wraps a cobra positional-arg validator (ExactArgs,
// MaximumNArgs, RangeArgs, NoArgs, etc) so a mismatch reports the
// command's usage line (e.g. "Usage: eim-skill device reboot
// <endpoint-or-name>") instead of cobra's bare default message ("accepts
// 1 arg(s), received 0"), which doesn't say what's actually missing or
// expected - confirmed confusing via a real bug report.
func requireArgs(validate cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := validate(cmd, args); err != nil {
			return fmt.Errorf("%s\nUsage: %s", err, cmd.UseLine())
		}
		return nil
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "conf/config.yaml", "config file path (配置文件路径)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "DeviceOn-EIM server base URL (overrides config file, e.g. http://127.0.0.1:30010) - 服务器地址（覆盖配置文件）")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "login username (overrides config file) - 登录用户名（覆盖配置文件）")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "login password (overrides config file) - 登录密码（覆盖配置文件）")
	rootCmd.PersistentFlags().StringVar(&repoBaseURL, "repo-base-url", "", "repo server (m2m-management-edge) base URL (overrides config file, e.g. http://127.0.0.1:30001) - 软件包仓库服务地址（覆盖配置文件）")
	rootCmd.PersistentFlags().StringVar(&ithingsBaseURL, "ithings-base-url", "", "sub-device/twin server (apphub-ithings) base URL (overrides config file, e.g. http://127.0.0.1:8082) - 子设备/数字孪生服务地址（覆盖配置文件）")
	rootCmd.PersistentFlags().StringVar(&caCertFile, "ca-cert-file", "", "PEM file of CA certificate(s) to trust, for servers using an internal/private CA (overrides config file) - 信任的 CA 证书文件（覆盖配置文件），用于服务器证书由内部私有 CA 签发的场景")
	rootCmd.PersistentFlags().BoolVar(&tlsInsecure, "insecure", false, "skip TLS certificate verification entirely (overrides config file; takes precedence over --ca-cert-file) - 完全跳过 TLS 证书校验（覆盖配置文件，优先级高于 --ca-cert-file），仅在信任该网络路径时使用")
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
	if ithingsBaseURL != "" {
		cfg.IthingsBaseURL = ithingsBaseURL
	}
	if caCertFile != "" {
		cfg.CACertFile = caCertFile
	}
	if tlsInsecure {
		cfg.InsecureSkipVerify = true
	}

	return cfg, nil
}

// newClient loads configuration and returns a ready-to-use AppHub client.
func newClient() (*apphub.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	httpClient, err := config.NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return apphub.NewClient(cfg.BaseURL, cfg.Username, cfg.Password, httpClient), nil
}

// newRepoClient loads configuration and returns a repo server client that
// authenticates by reusing appClient's AppHub session.
func newRepoClient(appClient *apphub.Client) (*repo.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	httpClient, err := config.NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return repo.NewClient(cfg.RepoBaseURL, appClient, httpClient), nil
}

// newIthingsClient loads configuration and returns a sub-device/twin
// server client that authenticates by reusing appClient's AppHub session.
func newIthingsClient(appClient *apphub.Client) (*ithings.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	httpClient, err := config.NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return ithings.NewClient(cfg.IthingsBaseURL, appClient, httpClient), nil
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

// addTimeWindowFlags registers the --last/--begin/--end flags shared by
// every command that filters results to a time range (device monitor,
// log system list, log operations list) - see resolveTimeWindow for how
// they're read back.
func addTimeWindowFlags(cmd *cobra.Command) {
	cmd.Flags().String("last", "", `only return entries from the last period, e.g. "30m", "12h", "3d" (default: no time filter; mutually exclusive with --begin/--end) - 只返回最近一段时间内的记录，如 "30m"(30分钟)/"12h"(12小时)/"3d"(3天)，不填则不限制时间范围（不能和 --begin/--end 同时使用）`)
	cmd.Flags().String("begin", "", `absolute start of the time range, RFC3339 or "YYYY-MM-DD HH:MM" (e.g. "2026-07-01 00:00") - only entries at or after this time (时间范围的起始点，RFC3339 或 "YYYY-MM-DD HH:MM"（如 "2026-07-01 00:00"）——只返回这个时间点及之后的记录)`)
	cmd.Flags().String("end", "", `absolute end of the time range, RFC3339 or "YYYY-MM-DD HH:MM" - only entries at or before this time; combine with --begin for an explicit window, or use alone/with just --begin for an open-ended one (时间范围的结束点，格式同 --begin——只返回这个时间点及之前的记录；可以和 --begin 搭配组成一个明确的时间区间，也可以单独使用，表示不限制起点)`)
}

// resolveTimeWindow reads the --last/--begin/--end flags (registered via
// addTimeWindowFlags) and returns the resulting [beginTs, endTs]
// Unix-millisecond window (0 on either side means unbounded). --last is
// a relative window ending now; --begin/--end give an absolute range
// instead (either or both may be given - an omitted bound is left
// unbounded) and are mutually exclusive with --last.
func resolveTimeWindow(cmd *cobra.Command) (beginTs, endTs int64, err error) {
	last, _ := cmd.Flags().GetString("last")
	beginStr, _ := cmd.Flags().GetString("begin")
	endStr, _ := cmd.Flags().GetString("end")

	if last != "" && (beginStr != "" || endStr != "") {
		return 0, 0, fmt.Errorf("--last cannot be combined with --begin/--end - use one or the other")
	}
	if last != "" {
		return lastDurationWindow(last)
	}

	if beginStr != "" {
		t, err := parseAbsoluteTime(beginStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid --begin %w", err)
		}
		beginTs = t.UnixMilli()
	}
	if endStr != "" {
		t, err := parseAbsoluteTime(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid --end %w", err)
		}
		endTs = t.UnixMilli()
	}
	return beginTs, endTs, nil
}
