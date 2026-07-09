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

func init() {
	rootCmd.AddCommand(groupCmd)
	groupCmd.AddCommand(groupListCmd)
}
