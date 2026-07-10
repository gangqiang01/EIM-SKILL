package cmd

import (
	"github.com/spf13/cobra"
)

var subdeviceCmd = &cobra.Command{
	Use:   "subdevice",
	Short: "Manage sub-devices behind a gateway (Modbus/OPC UA/IEC104/S7/SNMP)",
	Long: `Sub-devices are non-IP devices (PLCs, sensors, meters) that a gateway
device polls over an industrial protocol on their behalf. Models and
sub-devices are both created by uploading Excel workbooks (the
"model import-excel"/"import-excel" commands) - there is no one-at-a-time
create command.`,
}

var subdeviceModelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage sub-device models (schemas: modules/properties/events)",
}

func init() {
	rootCmd.AddCommand(subdeviceCmd)
	subdeviceCmd.AddCommand(subdeviceModelCmd)
	subdeviceModelCmd.AddCommand(subdeviceModelListCmd)
	subdeviceModelCmd.AddCommand(subdeviceModelGetCmd)
	subdeviceModelCmd.AddCommand(subdeviceModelImportExcelCmd)

	subdeviceCmd.AddCommand(subdeviceListCmd)
	subdeviceCmd.AddCommand(subdeviceGetCmd)
	subdeviceCmd.AddCommand(subdeviceImportExcelCmd)
	subdeviceCmd.AddCommand(subdeviceSetCollectionCmd)
	subdeviceCmd.AddCommand(subdeviceTwinCmd)
	subdeviceCmd.AddCommand(subdeviceTwinHistoryCmd)
	subdeviceCmd.AddCommand(subdeviceProtocolsCmd)

	subdeviceImportExcelCmd.Flags().StringSlice("edge-ids", nil, "comma-separated gateway/edge device endpoints to create the sub-devices on")
	subdeviceImportExcelCmd.Flags().String("group", "", "alternative to --edge-ids: create the sub-devices on every device in this group")

	subdeviceTwinHistoryCmd.Flags().Int64("begin", 0, "Unix millisecond timestamp lower bound")
	subdeviceTwinHistoryCmd.Flags().Int64("end", 0, "Unix millisecond timestamp upper bound")
}

var subdeviceModelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List every sub-device model",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		raw, err := ithingsClient.ListDeviceModels()
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var subdeviceModelGetCmd = &cobra.Command{
	Use:   "get <model-name>",
	Short: "Get a sub-device model's full detail (modules, properties, events)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		raw, err := ithingsClient.GetDeviceModel(args[0])
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var subdeviceModelImportExcelCmd = &cobra.Command{
	Use:   "import-excel <file.xlsx>",
	Short: "Bulk-create a sub-device model (modules/properties/events) from an Excel workbook",
	Long: `The workbook must have three sheets:
  deviceModel: name, description, manufacturer, industry, dataFormat (one data row)
  property:    module, name, description, dataType, maxValue, minValue, writeAble, unit
  event:       module, name, description`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		if err := ithingsClient.ImportDeviceModelExcel(args[0]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var subdeviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List every sub-device",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		raw, err := ithingsClient.ListSubDevices()
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var subdeviceGetCmd = &cobra.Command{
	Use:   "get <device-id>",
	Short: "Get a sub-device's full detail (services, properties, events)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		raw, err := ithingsClient.GetSubDevice(args[0])
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var subdeviceImportExcelCmd = &cobra.Command{
	Use:   "import-excel <file.xlsx>",
	Short: "Bulk-create sub-devices from an Excel workbook",
	Long: `The workbook must have sheets:
  subDevice:      deviceId, name, edgeId, description, deviceType, liftTimeOfDesiredValue,
                  deviceModel, protocolType, protocol, tags, reliable, reportOnChange (one data row - a template)
  propertyConfig: module, name, plus protocol-specific access-config columns
  eventConfig:    module, name, accessConfig
  commandConfig:  module, name, accessConfig
One sub-device is created per resolved gateway - pass either --edge-ids or --group, not both.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		edgeIDs, _ := cmd.Flags().GetStringSlice("edge-ids")
		group, _ := cmd.Flags().GetString("group")

		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		if err := ithingsClient.ImportSubDeviceExcel(args[0], edgeIDs, group); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var subdeviceSetCollectionCmd = &cobra.Command{
	Use:   "set-collection <device-id> <on|off>",
	Short: "Start or stop a sub-device's data collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		if err := ithingsClient.SetCollectionEnabled(args[0], enabled); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var subdeviceTwinCmd = &cobra.Command{
	Use:   "twin <device-id>",
	Short: "Get a sub-device's real-time collected data (device shadow)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		raw, err := ithingsClient.GetTwin(args[0])
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var subdeviceTwinHistoryCmd = &cobra.Command{
	Use:   "twin-history <device-id> <service-name> <property-name>",
	Short: "Get a sub-device property's collected data over a time range",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		begin, _ := cmd.Flags().GetInt64("begin")
		end, _ := cmd.Flags().GetInt64("end")

		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		raw, err := ithingsClient.QueryTwinHistory(args[0], args[1], args[2], begin, end)
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var subdeviceProtocolsCmd = &cobra.Command{
	Use:   "protocols",
	Short: "List the protocol types the server currently supports for sub-devices",
	Long: `Observed values include "modbus" and "snmp". This CLI has not seen "opcua",
"iec104", or "s7" registered in any deployment researched so far - if your
server supports them, this command will show it; otherwise those protocols
may need to be added on the server side first.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		raw, err := ithingsClient.ListProtocolTypes()
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}
