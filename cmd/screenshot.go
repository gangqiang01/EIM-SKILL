package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// screenshotResult is one device's outcome from "device screenshot".
type screenshotResult struct {
	Endpoint string `json:"endpoint"`
	DevName  string `json:"devName"`
	Status   string `json:"status"`
	File     string `json:"file,omitempty"`
	Error    string `json:"error,omitempty"`
}

func extensionForContentType(contentType string) string {
	switch {
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		return ".jpg"
	default:
		return ".bin"
	}
}

var deviceScreenshotCmd = &cobra.Command{
	Use:   "screenshot <endpoint-or-name> [endpoint-or-name...]",
	Short: "Capture and save a screenshot from one or more devices",
	Long: `Capture a screenshot from one or more devices and save each to a file.
Accepts any number of devices (endpoint or device name, mixed freely) in
one call - they're all told to start capturing at once, so this is
efficient for capturing several devices together rather than one at a
time.

This is a two-step, undocumented (in AppHub-v2.3.0.md) process
confirmed against a live server via a real packet capture: first, every
listed device is told (in one batch call) to start pushing periodic
screenshots to this server every --interval milliseconds; after waiting
that same interval (so at least one screenshot has actually arrived),
each device's most recent screenshot is fetched and saved. A device
without an active display session (some headless Linux devices,
confirmed via a live test) reports an error ("Get screen image error")
for that device rather than failing the whole batch - check each
result's "status".

对一台或多台设备截屏，并把每张图片保存成文件。一次调用可以传入任意
数量的设备（endpoint或设备名可以混用）——所有设备会被一次性通知开始
截屏，所以批量截取多台设备比逐台调用更高效。

这是一个未在 AppHub-v2.3.0.md 中记录的两步流程，已通过真实抓包在
真实服务器上验证过：第一步，一次性通知所有列出的设备，让它们开始
每隔 --interval 毫秒向本服务器推送一次截屏；等待同样长的时间（确保
至少有一张截屏已经推送过来）之后，第二步再逐台获取每台设备最近一次
的截屏并保存。如果某台设备没有活动的图形界面会话（部分无头Linux
设备，已通过真实测试确认），那台设备会报错（"Get screen image
error"），但不会影响批次里其他设备——请检查每条结果里的 "status"。

Example 示例: eim-skill device screenshot "146打包机" "联想笔记本" --out-dir ./shots

Example output 输出示例 (real, one succeeded and one failed 真实返回，
一台成功一台失败):
  {
    "status": 200, "msg": "",
    "data": [
      {"endpoint": "28:16:AD:F2:56:5C", "devName": "联想笔记本", "status": "OK", "file": "shots/28_16_AD_F2_56_5C.jpg"},
      {"endpoint": "74:FE:48:05:0A:EB", "devName": "74:FE:48:05:0A:EB", "status": "ERROR", "error": "DeviceOn-EIM: Get screen image error (http 400)"}
    ]
  }`,
	Args: requireArgs(cobra.MinimumNArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		intervalMs, _ := cmd.Flags().GetInt("interval")
		outDir, _ := cmd.Flags().GetString("out-dir")
		host, _ := cmd.Flags().GetString("host")
		if intervalMs <= 0 {
			return fmt.Errorf("--interval must be a positive number of milliseconds")
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		devices, err := client.ListDevices()
		if err != nil {
			return err
		}

		type target struct {
			endpoint string
			devName  string
		}
		targets := make([]target, 0, len(args))
		for _, arg := range args {
			found := false
			for _, d := range devices {
				if d.Endpoint == arg || d.DevName == arg {
					targets = append(targets, target{endpoint: d.Endpoint, devName: d.DevName})
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("no device found with endpoint or name %q", arg)
			}
		}

		endpoints := make([]string, len(targets))
		for i, t := range targets {
			endpoints[i] = t.endpoint
		}

		if host == "" {
			host = client.BaseURL()
		}
		if err := client.BatchStartScreenshotStream(endpoints, host, intervalMs); err != nil {
			return err
		}

		time.Sleep(time.Duration(intervalMs) * time.Millisecond)

		if outDir != "" {
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return fmt.Errorf("create --out-dir %s: %w", outDir, err)
			}
		}

		results := make([]screenshotResult, 0, len(targets))
		for _, t := range targets {
			data, contentType, err := client.GetScreenshot(t.endpoint)
			if err != nil {
				results = append(results, screenshotResult{
					Endpoint: t.endpoint, DevName: t.devName, Status: "ERROR", Error: err.Error(),
				})
				continue
			}

			fname := strings.ReplaceAll(t.endpoint, ":", "") + extensionForContentType(contentType)
			if outDir != "" {
				fname = filepath.Join(outDir, fname)
			}
			if err := os.WriteFile(fname, data, 0o644); err != nil {
				results = append(results, screenshotResult{
					Endpoint: t.endpoint, DevName: t.devName, Status: "ERROR",
					Error: fmt.Sprintf("write %s: %v", fname, err),
				})
				continue
			}
			results = append(results, screenshotResult{
				Endpoint: t.endpoint, DevName: t.devName, Status: "OK", File: fname,
			})
		}

		printJSON(results)
		return nil
	},
}

func init() {
	deviceCmd.AddCommand(deviceScreenshotCmd)
	deviceScreenshotCmd.Flags().Int("interval", 5000, "how often devices push a new screenshot, and how long to wait before fetching, in milliseconds (设备推送截屏的间隔，以及获取前的等待时间，单位毫秒)")
	deviceScreenshotCmd.Flags().String("out-dir", "", "directory to save screenshots in (default: current directory) (保存截屏图片的目录，默认为当前目录)")
	deviceScreenshotCmd.Flags().String("host", "", "server address devices should push screenshots to (default: this CLI's configured --base-url) - override only if devices can't reach that address directly (设备应该把截屏推送到的服务器地址，默认使用本CLI配置的 --base-url——只有当设备无法直接访问该地址时才需要覆盖)")
}
