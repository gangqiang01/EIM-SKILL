package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Query and control devices managed by DeviceLink/AppHub",
}

func init() {
	rootCmd.AddCommand(deviceCmd)
	deviceCmd.AddCommand(deviceListCmd)
	deviceCmd.AddCommand(deviceGetCmd)
	deviceCmd.AddCommand(deviceDeleteCmd)
	deviceCmd.AddCommand(deviceAddToGroupCmd)
	deviceCmd.AddCommand(deviceMonitorCmd)
	deviceCmd.AddCommand(deviceRebootCmd)
	deviceCmd.AddCommand(deviceShutdownCmd)
	deviceCmd.AddCommand(deviceSetVolumeCmd)
	deviceCmd.AddCommand(deviceSetBacklightCmd)

	deviceListCmd.Flags().String("group", "", "only list devices in this device group")

	deviceMonitorCmd.Flags().String("type", "", "monitor data type: cpu or memory (required)")
	deviceMonitorCmd.Flags().Int64("begin", 0, "only return samples at or after this Unix millisecond timestamp")
	deviceMonitorCmd.Flags().Int64("end", 0, "only return samples at or before this Unix millisecond timestamp")
	_ = deviceMonitorCmd.MarkFlagRequired("type")
}

func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatalf("failed to render output: %v", err)
	}
	fmt.Println(string(data))
}

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices, or the devices in a given group",
	RunE: func(cmd *cobra.Command, args []string) error {
		group, _ := cmd.Flags().GetString("group")

		client, err := newClient()
		if err != nil {
			return err
		}

		if group != "" {
			devices, err := client.ListDevicesByGroup(group)
			if err != nil {
				return err
			}
			printJSON(devices)
			return nil
		}

		devices, err := client.ListDevices()
		if err != nil {
			return err
		}
		printJSON(devices)
		return nil
	},
}

var deviceGetCmd = &cobra.Command{
	Use:   "get <endpoint>",
	Short: "Get the details of a single device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		device, err := client.GetDevice(args[0])
		if err != nil {
			return err
		}
		printJSON(device)
		return nil
	},
}

var deviceDeleteCmd = &cobra.Command{
	Use:   "delete <endpoint>",
	Short: "Delete a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.DeleteDevice(args[0]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceMonitorCmd = &cobra.Command{
	Use:   "monitor <endpoint>",
	Short: "Get CPU or memory monitoring data for a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dataType, _ := cmd.Flags().GetString("type")
		begin, _ := cmd.Flags().GetInt64("begin")
		end, _ := cmd.Flags().GetInt64("end")

		client, err := newClient()
		if err != nil {
			return err
		}
		points, err := client.MonitorData(args[0], dataType, begin, end)
		if err != nil {
			return err
		}
		printJSON(points)
		return nil
	},
}

var deviceAddToGroupCmd = &cobra.Command{
	Use:   "add-to-group <endpoint> <group-name>",
	Short: "Add a device to a device group (keeps its existing group memberships)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		groups, err := client.AddDeviceToGroup(args[0], args[1])
		if err != nil {
			return err
		}
		printJSON(map[string]interface{}{
			"status": "CHANGED",
			"groups": groups,
		})
		return nil
	},
}

var deviceRebootCmd = &cobra.Command{
	Use:   "reboot <endpoint>",
	Short: "Reboot a device immediately",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.Reboot(args[0]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceShutdownCmd = &cobra.Command{
	Use:   "shutdown <endpoint>",
	Short: "Shut down a device immediately",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.Shutdown(args[0]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSetVolumeCmd = &cobra.Command{
	Use:   "set-volume <endpoint> <0-100>",
	Short: "Set a device's system audio volume (best effort, see NOTE in internal/apphub)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		volume, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("volume must be an integer 0-100: %w", err)
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.SetVolume(args[0], volume); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSetBacklightCmd = &cobra.Command{
	Use:   "set-backlight <endpoint> <0-100>",
	Short: "Set a device's screen backlight/brightness (best effort, see NOTE in internal/apphub)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		brightness, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("brightness must be an integer 0-100: %w", err)
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		if err := client.SetBacklight(args[0], brightness); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}
