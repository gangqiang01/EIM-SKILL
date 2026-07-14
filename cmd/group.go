package cmd

import "github.com/spf13/cobra"

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Query and create device groups managed by DeviceOn-EIM",
	Long: `Query and create device groups managed by DeviceOn-EIM.

查询和创建由 DeviceOn-EIM 管理的设备组。设备组是"device list
--group"、"device reboot-group" 等按组批量操作命令的作用范围。`,
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all device groups",
	Long: `List every device group defined on the server, with member counts.

列出服务器上所有已定义的设备组，包含成员数量统计。

Example 示例: eim-skill group list

Example output 输出示例 (real 真实返回):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "gid": 1, "name": "default", "desc": "default group in server",
        "type": "mix", "total": 6, "online": 1, "ts": 1779329588002
      }
    ]
  }`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		groups, err := client.ListGroups()
		if err != nil {
			return err
		}
		printJSON(groups)
		return nil
	},
}

var groupStatusCmd = &cobra.Command{
	Use:   "status [group-name]",
	Short: "Get device groups' online/warning/error status breakdown",
	Long: `With no group-name, returns every group's online/warning/error
breakdown; with one, returns just that group's.

不指定 group-name 时返回所有组各自的 在线/警告/错误 状态统计；
指定了就只返回那一个组的统计。

Example 示例: eim-skill group status
       eim-skill group status default

Example output 输出示例 (real 真实返回):
  {
    "status": 200, "msg": "",
    "data": [
      {"name": "default", "type": "mix", "total": 6, "online": 1, "warning": 0, "error": 0}
    ]
  }`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			status, err := client.GetGroupStatus(args[0])
			if err != nil {
				return err
			}
			printJSON(status)
			return nil
		}

		statuses, err := client.ListGroupStatuses()
		if err != nil {
			return err
		}
		printJSON(statuses)
		return nil
	},
}

var groupCreateCmd = &cobra.Command{
	Use:   "create <name> <type> [description]",
	Short: "Add a new device group",
	Long: `Create a new device group. type is typically one of "linux", "windows",
"android", "mix" (the server's built-in defaults, though not a strictly
enforced enum) - "mix" allows devices of any OS. description is optional.
--creator defaults to the CLI's configured login username if omitted.

创建一个新的设备组。type 通常是 "linux"、"windows"、"android"、"mix"
（服务器内置的默认几种，但并非强制枚举）之一——"mix" 表示不限设备
操作系统。description 可选。--creator 不填时默认使用当前CLI登录配置
的用户名。

Example 示例: eim-skill group create edge-line-1 mix "产线1边缘设备"

Example output 输出示例: {"status": 200, "msg": "", "data": {"status": "CHANGED"}}`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, groupType := args[0], args[1]
		desc := ""
		if len(args) > 2 {
			desc = args[2]
		}
		creator, _ := cmd.Flags().GetString("creator")

		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if creator == "" {
			creator = cfg.Username
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.CreateGroup(name, desc, groupType, creator); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(groupCmd)
	groupCmd.AddCommand(groupListCmd)
	groupCmd.AddCommand(groupStatusCmd)
	groupCmd.AddCommand(groupCreateCmd)

	groupCreateCmd.Flags().String("creator", "", "username recorded as the group's creator (default: the configured login username) - 记录为创建者的用户名（默认使用当前登录配置的用户名）")
}
