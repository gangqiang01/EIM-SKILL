package cmd

import "github.com/spf13/cobra"

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Query device groups managed by DeviceLink/AppHub",
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all device groups",
	Args:  cobra.NoArgs,
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
	Long:  `With no group-name, returns every group's breakdown; with one, returns just that group's.`,
	Args:  cobra.MaximumNArgs(1),
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
	Long:  `type is typically one of "linux", "windows", "android", "mix" (the server's built-in defaults).`,
	Args:  cobra.RangeArgs(2, 3),
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

	groupCreateCmd.Flags().String("creator", "", "username recorded as the group's creator (default: the configured login username)")
}
