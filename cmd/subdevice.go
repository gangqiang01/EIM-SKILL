package cmd

import (
	"fmt"
	"strings"

	"github.com/gangqiang01/EIM-SKILL/internal/ithings"
	"github.com/spf13/cobra"
)

// resolveSubDeviceID finds the sub-device matching identifier - its
// device id, or failing that, its name (which isn't guaranteed unique
// across different parent devices, unlike endpoint/devName) - and
// returns its device id.
func resolveSubDeviceID(client *ithings.Client, identifier string) (string, error) {
	devices, err := client.ListSubDevices()
	if err != nil {
		return "", err
	}

	for _, d := range devices {
		if d.DeviceID == identifier {
			return d.DeviceID, nil
		}
	}

	var matches []ithings.SubDevice
	for _, d := range devices {
		if d.Name == identifier {
			matches = append(matches, d)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no sub-device found with id or name %q", identifier)
	case 1:
		return matches[0].DeviceID, nil
	default:
		descriptions := make([]string, len(matches))
		for i, m := range matches {
			descriptions[i] = fmt.Sprintf("%s (parent %s)", m.DeviceID, m.EdgeID)
		}
		return "", fmt.Errorf("multiple sub-devices are named %q - specify the device id instead: %s", identifier, strings.Join(descriptions, ", "))
	}
}

var subdeviceCmd = &cobra.Command{
	Use:   "subdevice",
	Short: "Query sub-devices (\"子设备\") registered under a parent device - services, properties, history, events",
	Long: `Sub-devices are things registered under a parent DeviceOn-EIM-managed
device (the "edge") via a protocol like Modbus/SNMP/executor/restful -
each reports a set of properties grouped into named services, and can
raise events. This is a separate, read-only query surface backed by the
apphub-ithings server (not documented in AppHub-v2.3.0.md for most of
what it wraps), talking to a different server/port than the main
DeviceOn-EIM API.

子设备是注册在某个父设备（"edge"）下的物模型设备，通过 Modbus/SNMP/
executor/restful 等协议接入——每个子设备上报若干个属性，按"服务"分组，
也可以上报事件。这是一套独立的只读查询能力，由 apphub-ithings 服务
提供（其中大部分接口未在 AppHub-v2.3.0.md 中记录），走的是和主
DeviceOn-EIM API 不同的服务器/端口。`,
}

var subdeviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sub-devices, optionally filtered to one parent device",
	Long: `List every sub-device registered on the server. --parent optionally
narrows this to sub-devices belonging to one specific parent device
(there is no server-side filter for this - it's applied client-side
after fetching the full list).

列出服务器上注册的所有子设备。可选 --parent，只看某一个父设备下的
子设备（服务器本身不支持按父设备过滤，这里是拉取全量列表后在客户端
过滤）。

Example 示例: eim-skill subdevice list --parent "146打包机"

Example output 输出示例 (real 真实返回):
  {
    "status": 200, "msg": "",
    "data": [
      {
        "deviceId": "65bcef7a5d0d4593a1e13d000ebe96d9",
        "name": "test",
        "edgeID": "C4:00:AD:11:DA:97",
        "description": "",
        "deviceType": "subDevice",
        "deviceModelRef": "ResourceMonitor",
        "protocolType": "monitor",
        "protocol": "{\"it_u\":\"s\",\"it\":10}",
        "createTimeStamp": 1784715561236,
        "updateTimeStamp": 1784715589928,
        "deviceStatus": "online",
        "state": "started",
        "deviceModelId": 1
      }
    ]
  }`,
	Args: requireArgs(cobra.NoArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		parent, _ := cmd.Flags().GetString("parent")

		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}

		devices, err := ithingsClient.ListSubDevices()
		if err != nil {
			return err
		}

		if parent == "" {
			printJSON(devices)
			return nil
		}

		parentEndpoint, err := resolveDeviceEndpoint(client, parent)
		if err != nil {
			return err
		}
		filtered := make([]ithings.SubDevice, 0, len(devices))
		for _, d := range devices {
			if d.EdgeID == parentEndpoint {
				filtered = append(filtered, d)
			}
		}
		printJSON(filtered)
		return nil
	},
}

var subdeviceGetCmd = &cobra.Command{
	Use:   "get <device-id-or-name>",
	Short: "Get a sub-device's services/properties and their current real-time values",
	Long: `Get a sub-device's "digital twin" - every service+property it has ever
reported, along with its current real-time value. Since a sub-device has
no separate schema-only endpoint wrapped here, this is also how you
discover which service-name/property-name pairs exist, to pass to
"subdevice history".

获取子设备的"数字孪生"——它上报过的每一个 服务+属性，以及各自当前的
实时值。由于这里没有单独封装"仅看结构不看数据"的接口，这个命令同时
也是查看有哪些 服务名/属性名 组合可用（供 "subdevice history" 使用）
的方式。

An offline or never-reported sub-device returns an empty list rather than
an error - confirmed against a live server.

一台离线或从未上报过数据的子设备会返回空列表而不是报错——已用真实
服务器验证过。

--service/--property narrow this down - there is no server-side filter
for either (the underlying endpoint always returns every reported
property), so this is applied client-side after fetching the full twin.
With both given, the single matching property's object is returned
directly (not wrapped in a list) if found - or an error if no property
currently has a value under that exact service+property name. With just
one of the two, every match is returned as a list.

--service/--property 用来缩小范围——服务器本身不支持按这两个条件过滤
（底层接口总是返回全部已上报的属性），这里是拉取完整"数字孪生"后在
客户端过滤。两者都指定时，如果找到唯一匹配，直接返回这一个属性对象
（不包在列表里）；如果指定的 服务+属性 组合当前没有值，则报错。只指定
其中一个时，返回所有匹配项组成的列表。

Example 示例: eim-skill subdevice get "test"
       eim-skill subdevice get "test" --service "Default Module" --property "Memory Used"

Example output 输出示例 (real 真实返回，无 --service/--property 时):
  {
    "status": 200, "msg": "",
    "data": [
      {"svc": "Default Module", "pn": "Memory Used", "val": 2666, "ts": 1784715998240}
    ]
  }
Example output 输出示例 (real 真实返回，--service/--property 都指定时):
  {"status": 200, "msg": "", "data": {"svc": "Default Module", "pn": "Memory Used", "val": 2666, "ts": 1784715998240}}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _ := cmd.Flags().GetString("service")
		property, _ := cmd.Flags().GetString("property")

		client, err := newClient()
		if err != nil {
			return err
		}
		ithingsClient, err := newIthingsClient(client)
		if err != nil {
			return err
		}
		deviceID, err := resolveSubDeviceID(ithingsClient, args[0])
		if err != nil {
			return err
		}
		props, err := ithingsClient.SubDeviceTwin(deviceID)
		if err != nil {
			return err
		}

		if service == "" && property == "" {
			printJSON(props)
			return nil
		}

		matches := make([]ithings.TwinProperty, 0, len(props))
		for _, p := range props {
			if service != "" && p.Svc != service {
				continue
			}
			if property != "" && p.Pn != property {
				continue
			}
			matches = append(matches, p)
		}

		if service != "" && property != "" {
			switch len(matches) {
			case 0:
				return fmt.Errorf("no current value found for service %q, property %q - run \"subdevice get\" without --service/--property to see the exact svc/pn names this device reports", service, property)
			case 1:
				printJSON(matches[0])
				return nil
			}
		}

		printJSON(matches)
		return nil
	},
}

var subdeviceHistoryCmd = &cobra.Command{
	Use:   "history <device-id-or-name> <service-name> <property-name>",
	Short: "Get historical values of one specific service+property on a sub-device",
	Long: `Get historical samples of one service+property on a sub-device.
service-name/property-name must exactly match an entry from "subdevice
get" (its "svc"/"pn" fields).

获取子设备某一个 服务+属性 的历史数据。service-name/property-name
必须和 "subdevice get" 返回结果里的 "svc"/"pn" 字段完全一致。

--last/--begin/--end restrict the time range, same as elsewhere in this
CLI (see "device monitor --help") - HOWEVER, confirmed against a live
server that this endpoint's beginTs/endTs/count currently have NO
observable filtering effect: every range tested (including deliberately
absurd ones) returned the identical most-recent ~20 samples. They're
still sent (matching a real packet capture) in case a different server
version honors them, but don't rely on them actually narrowing results
today - fetch and filter client-side if you need a real window.

--last/--begin/--end 用来限定时间范围，用法和本CLI其他命令一致（见
"device monitor --help"）——但是，已用真实服务器验证：这个接口的
beginTs/endTs/count 目前没有任何可观察到的过滤效果，无论传入什么范围
（包括故意构造的极端范围）返回的都是同样的最近约20条数据。这里仍然
会发送这些参数（和真实抓包的请求形状保持一致），以防某个其他版本的
服务器会认这些参数，但今天不要指望它们能真正缩小结果范围——如果需要
真正限定时间窗口，请自行在客户端过滤。

Example 示例: eim-skill subdevice history "test" "Default Module" "Memory Used"

Example output 输出示例 (real 真实返回，已截断):
  {
    "status": 200, "msg": "",
    "data": [
      {"svc": "Default Module", "pn": "Memory Used", "val": 2666, "ts": 1784715828241}
    ]
  }`,
	Args: requireArgs(cobra.ExactArgs(3)),
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")
		begin, end, err := resolveTimeWindow(cmd)
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
		deviceID, err := resolveSubDeviceID(ithingsClient, args[0])
		if err != nil {
			return err
		}

		props, err := ithingsClient.QueryTwinHistory(deviceID, args[1], args[2], begin, end, count)
		if err != nil {
			return err
		}
		printJSON(props)
		return nil
	},
}

var subdeviceEventsCmd = &cobra.Command{
	Use:   "events <device-id-or-name>",
	Short: "List a sub-device's recorded events",
	Long: `List one page of a sub-device's recorded events. Not documented in
AppHub-v2.3.0.md at all; confirmed against a live server to respond
successfully, but no sub-device available to test against had any
recorded events, so a populated response's per-event shape is unconfirmed
- entries are passed through as-is rather than reshaped.

列出子设备的事件记录（分页）。这个接口完全未在 AppHub-v2.3.0.md 中
记录；已确认对真实服务器请求能成功返回，但测试环境里没有任何子设备
有真实的事件记录，所以"有数据时"每条事件的具体字段结构尚未验证——这里
直接透传原始返回内容，不做重新整形。

--last/--begin/--end restrict the time range - untested whether the
server actually honors them for this endpoint (no recorded events were
available to verify against), unlike "subdevice history" where a lack of
effect was directly confirmed.

--last/--begin/--end 用来限定时间范围——由于没有真实事件数据可供验证，
这个接口是否真的支持按时间过滤尚未确认（不同于 "subdevice history"，
那边已经直接验证过无效）。

Example 示例: eim-skill subdevice events "test" --limit 10

Example output 输出示例 (real 真实返回，当前测试环境无事件记录):
  {"status": 200, "msg": "", "data": {"total": 0, "events": []}}`,
	Args: requireArgs(cobra.ExactArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		begin, end, err := resolveTimeWindow(cmd)
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
		deviceID, err := resolveSubDeviceID(ithingsClient, args[0])
		if err != nil {
			return err
		}

		events, total, err := ithingsClient.ListDeviceEvents("", deviceID, page, limit, begin, end)
		if err != nil {
			return err
		}
		printJSON(map[string]interface{}{
			"total":  total,
			"events": events,
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(subdeviceCmd)
	subdeviceCmd.AddCommand(subdeviceListCmd)
	subdeviceCmd.AddCommand(subdeviceGetCmd)
	subdeviceCmd.AddCommand(subdeviceHistoryCmd)
	subdeviceCmd.AddCommand(subdeviceEventsCmd)

	subdeviceListCmd.Flags().String("parent", "", "only list sub-devices belonging to this parent device, endpoint or device name (只列出这个父设备下的子设备，可用endpoint或设备名)")

	subdeviceGetCmd.Flags().String("service", "", "only return properties under this service name (svc) (只返回这个服务名下的属性)")
	subdeviceGetCmd.Flags().String("property", "", "only return this property name (pn) - combine with --service to get a single exact value (只返回这个属性名——和 --service 一起用可以精确定位到唯一一个值)")

	addTimeWindowFlags(subdeviceHistoryCmd)
	subdeviceHistoryCmd.Flags().Int("count", 0, "cap on how many samples to return (0 = no cap sent) - observed to have no filtering effect on this server, see caveat above (返回条数上限（0表示不传该参数）——已确认在当前服务器上无过滤效果，见上方说明)")

	addTimeWindowFlags(subdeviceEventsCmd)
	subdeviceEventsCmd.Flags().Int("page", 1, "page number, 1-based (页码，从1开始)")
	subdeviceEventsCmd.Flags().Int("limit", 20, "how many events per page (每页返回多少条事件)")
}
