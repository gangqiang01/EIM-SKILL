package cmd

import (
	"strconv"

	"github.com/gangqiang01/EIM-SKILL/internal/ithings"
	"github.com/spf13/cobra"
)

var dataforwardCmd = &cobra.Command{
	Use:   "dataforward",
	Short: "Forward device data to AMQP/MQTT/Kafka brokers",
	Long: `Configures the ithings server's data-forward rules, which push sub-device
telemetry (selected by device model) to an external message broker.`,
}

func init() {
	rootCmd.AddCommand(dataforwardCmd)
	dataforwardCmd.AddCommand(dataforwardListCmd)
	dataforwardCmd.AddCommand(dataforwardGetCmd)
	dataforwardCmd.AddCommand(dataforwardCreateCmd)
	dataforwardCmd.AddCommand(dataforwardSetCmd)
	dataforwardCmd.AddCommand(dataforwardEnableCmd)
	dataforwardCmd.AddCommand(dataforwardDisableCmd)
	dataforwardCmd.AddCommand(dataforwardDeleteCmd)
	dataforwardCmd.AddCommand(dataforwardLogsCmd)

	dataforwardSetCmd.Flags().String("model-id", "", "forward data from every device of this device-model id (source)")
	dataforwardSetCmd.Flags().String("host", "", "broker host:port")
	dataforwardSetCmd.Flags().String("username", "", "broker username")
	dataforwardSetCmd.Flags().String("password", "", "broker password")
	dataforwardSetCmd.Flags().String("topic", "", "MQTT/Kafka topic")
	dataforwardSetCmd.Flags().String("queue", "", "AMQP queue name")
	dataforwardSetCmd.Flags().String("client-id", "", "MQTT client id")
	dataforwardSetCmd.Flags().Int("qos", 0, "MQTT QoS (0-2)")
	dataforwardSetCmd.Flags().Bool("retain", false, "MQTT retain flag")
}

var dataforwardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List every configured data-forward rule",
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
		raw, err := ithingsClient.ListDataForwards()
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var dataforwardGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get one data-forward rule by name",
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
		raw, err := ithingsClient.GetDataForward(args[0])
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}

var dataforwardCreateCmd = &cobra.Command{
	Use:   "create <name> [description]",
	Short: "Create an empty (disabled) data-forward rule",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := ""
		if len(args) > 1 {
			description = args[1]
		}
		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		if err := ithingsClient.CreateDataForward(args[0], description); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var dataforwardSetCmd = &cobra.Command{
	Use:   "set <name> <amqp|kafka|mqtt> [description]",
	Short: "Configure a data-forward rule's source device model and broker destination",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, destType := args[0], args[1]
		description := ""
		if len(args) > 2 {
			description = args[2]
		}

		var sources []ithings.DataForwardSource
		if modelIDStr, _ := cmd.Flags().GetString("model-id"); modelIDStr != "" {
			modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
			if err != nil {
				return err
			}
			sources = append(sources, ithings.DataForwardSource{Type: "model", ModelId: &modelID})
		}

		host, _ := cmd.Flags().GetString("host")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		topic, _ := cmd.Flags().GetString("topic")
		queue, _ := cmd.Flags().GetString("queue")
		clientID, _ := cmd.Flags().GetString("client-id")
		qos, _ := cmd.Flags().GetInt("qos")
		retain, _ := cmd.Flags().GetBool("retain")

		destination := ithings.DataForwardDestination{
			Type:      destType,
			Host:      host,
			UserName:  username,
			Password:  password,
			Topic:     topic,
			QueueName: queue,
			ClientId:  clientID,
			QOS:       qos,
			Retain:    retain,
		}

		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		if err := ithingsClient.SetDataForward(name, description, sources, destination); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var dataforwardEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a data-forward rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setDataForwardEnabled(args[0], true)
	},
}

var dataforwardDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a data-forward rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setDataForwardEnabled(args[0], false)
	},
}

func setDataForwardEnabled(name string, enabled bool) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	ithingsClient, err := newIthingsClient(client)
	if err != nil {
		return err
	}
	if err := ithingsClient.SetDataForwardEnabled(name, enabled); err != nil {
		return err
	}
	printJSON(map[string]string{"status": "CHANGED"})
	return nil
}

var dataforwardDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a data-forward rule by id (see \"dataforward list\" for ids)",
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
		if err := ithingsClient.DeleteDataForward(args[0]); err != nil {
			return err
		}
		printJSON(map[string]string{"status": "CHANGED"})
		return nil
	},
}

var dataforwardLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "List the data-forward delivery log",
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
		raw, err := ithingsClient.ListDataForwardLogs()
		if err != nil {
			return err
		}
		printRawJSON(raw)
		return nil
	},
}
