package cmd

import "github.com/spf13/cobra"

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Query server-level configuration (not scoped to a specific device)",
	Long: `Query configuration that applies to the whole DeviceOn-EIM server,
rather than to a specific device.

查询整个 DeviceOn-EIM 服务器级别的配置信息，而不是针对某台设备。`,
}

var serverGetConfigCmd = &cobra.Command{
	Use:   "get-config",
	Short: "Get the server's client-onboarding config blob",
	Long: `Get the config blob used to connect a new device agent directly to this
server - paste the returned "data" string into the device agent's own
configuration UI to have it connect here, instead of typing the server
address/credentials into the agent by hand. This mirrors the web
console's QR-code device-pairing feature - the same data is what gets
encoded into the QR code shown there.

获取用于让新设备agent直接连接到本服务器的配置数据——把返回的 "data"
字符串复制粘贴到设备agent自己的配置界面里，就能让它连接到这台服务器，
不需要在agent里手动逐项填写服务器地址/凭据。这对应网页控制台的二维码
设备配对功能——网页上生成的二维码内容就是这段数据。

Not documented in AppHub-v2.3.0.md; confirmed against a live server.

未在 AppHub-v2.3.0.md 中记录；已用真实服务器验证过。

Example 示例: eim-skill server get-config

Example output 输出示例 (real, truncated 真实返回，已截断):
  {"status": 200, "msg": "", "data": {"id": "WP.SRP", "data": "ilNczWH8CSiy32AC9kz+pTTxR..."}}`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		cfg, err := client.GetQRCodeData()
		if err != nil {
			return err
		}
		printJSON(cfg)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverGetConfigCmd)
}
