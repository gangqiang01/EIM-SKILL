package cmd

import (
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
	deviceCmd.AddCommand(deviceCountsCmd)
	deviceCmd.AddCommand(deviceHealthCmd)
	deviceCmd.AddCommand(deviceGetCmd)
	deviceCmd.AddCommand(deviceDetailsCmd)
	deviceCmd.AddCommand(deviceDeleteCmd)
	deviceCmd.AddCommand(deviceAddToGroupCmd)
	deviceCmd.AddCommand(deviceMonitorCmd)
	deviceCmd.AddCommand(deviceRebootCmd)
	deviceCmd.AddCommand(deviceShutdownCmd)
	deviceCmd.AddCommand(deviceSetVolumeCmd)
	deviceCmd.AddCommand(deviceSetBacklightCmd)

	deviceListCmd.Flags().String("group", "", "only list devices in this device group")
	deviceListCmd.Flags().Bool("online", false, "only list online devices")
	deviceListCmd.Flags().Bool("offline", false, "only list offline devices")
	deviceListCmd.Flags().Bool("alarm", false, "only list devices currently in an alarm state (software/hardware status not normal)")
	deviceListCmd.MarkFlagsMutuallyExclusive("online", "offline", "alarm")

	deviceMonitorCmd.Flags().String("type", "", "monitor data type: cpu or memory (required)")
	deviceMonitorCmd.Flags().Int64("begin", 0, "only return samples at or after this Unix millisecond timestamp")
	deviceMonitorCmd.Flags().Int64("end", 0, "only return samples at or before this Unix millisecond timestamp")
	_ = deviceMonitorCmd.MarkFlagRequired("type")
}

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices, or filter by group / online / offline / alarm state",
	RunE: func(cmd *cobra.Command, args []string) error {
		group, _ := cmd.Flags().GetString("group")
		online, _ := cmd.Flags().GetBool("online")
		offline, _ := cmd.Flags().GetBool("offline")
		alarm, _ := cmd.Flags().GetBool("alarm")

		client, err := newClient()
		if err != nil {
			return err
		}

		if alarm {
			devices, err := client.ListAlarmDevices(group)
			if err != nil {
				return err
			}
			printJSON(devices)
			return nil
		}

		if online || offline {
			isOnline := online
			devices, err := client.ListDevicesByCondition(&isOnline, group, "", "")
			if err != nil {
				return err
			}
			printJSON(devices)
			return nil
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

var deviceCountsCmd = &cobra.Command{
	Use:   "counts",
	Short: "Get the number of online and offline devices",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		counts, err := client.GetDeviceCounts()
		if err != nil {
			return err
		}
		printJSON(counts)
		return nil
	},
}

var deviceHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Get the status breakdown (normal/warning/error) of online devices",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		health, err := client.GetOnlineDeviceHealth()
		if err != nil {
			return err
		}
		printJSON(health)
		return nil
	},
}

var deviceGetCmd = &cobra.Command{
	Use:   "get <endpoint-or-name>",
	Short: "Get the details of a single device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		device, err := client.GetDevice(endpoint)
		if err != nil {
			return err
		}
		printJSON(device)
		return nil
	},
}

var deviceDetailsCmd = &cobra.Command{
	Use:   "details <endpoint-or-name>",
	Short: "Get a device's detailed system info (cpu, memory, storage, network interfaces, etc)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		details, err := client.GetDeviceDetails(endpoint)
		if err != nil {
			return err
		}
		printRawJSON(details)
		return nil
	},
}

var deviceDeleteCmd = &cobra.Command{
	Use:   "delete <endpoint-or-name>",
	Short: "Delete a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.DeleteDevice(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceMonitorCmd = &cobra.Command{
	Use:   "monitor <endpoint-or-name>",
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
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		points, err := client.MonitorData(endpoint, dataType, begin, end)
		if err != nil {
			return err
		}
		printJSON(points)
		return nil
	},
}

var deviceAddToGroupCmd = &cobra.Command{
	Use:   "add-to-group <endpoint-or-name> <group-name>",
	Short: "Add a device to a device group (keeps its existing group memberships)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}

		groups, err := client.AddDeviceToGroup(endpoint, args[1])
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
	Use:   "reboot <endpoint-or-name>",
	Short: "Reboot a device immediately",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.Reboot(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceShutdownCmd = &cobra.Command{
	Use:   "shutdown <endpoint-or-name>",
	Short: "Shut down a device immediately",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.Shutdown(endpoint); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSetVolumeCmd = &cobra.Command{
	Use:   "set-volume <endpoint-or-name> <0-100>",
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
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SetVolume(endpoint, volume); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var deviceSetBacklightCmd = &cobra.Command{
	Use:   "set-backlight <endpoint-or-name> <0-100>",
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
		endpoint, err := resolveDeviceEndpoint(client, args[0])
		if err != nil {
			return err
		}
		if err := client.SetBacklight(endpoint, brightness); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}
