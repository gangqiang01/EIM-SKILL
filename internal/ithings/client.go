// Package ithings is an HTTP client for the "apphub-ithings" server -
// DeviceOn-EIM's sub-device/digital-twin service. Sub-devices ("子设备")
// are things registered under a parent device (the "edge" - a regular
// DeviceOn-EIM-managed device) via LwM2M/Modbus/SNMP/etc protocols; each
// reports a set of properties, grouped into named services, and can raise
// events. Not documented in AppHub-v2.3.0.md's "子设备模型与子设备/子设备
// 管理" section for most of what this package wraps (only the sub-device
// list, twin, and history endpoints are - the deviceEvent endpoint isn't
// documented at all); confirmed against a live server via a real packet
// capture. It runs alongside the main AppHub/DeviceLink server (on its
// own port, e.g. 8082) and accepts the same accesstoken issued by the
// main server's /login - confirmed live, there is no separate ithings
// login.
package ithings

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// TokenSource supplies the accesstoken used to authenticate against the
// ithings server, refreshing it on demand. *apphub.Client satisfies this.
type TokenSource interface {
	Token() (string, error)
	Relogin() error
}

// Client talks to the apphub-ithings server over HTTP.
type Client struct {
	baseURL    string
	tokens     TokenSource
	httpClient *http.Client
}

// NewClient creates a Client for the ithings server at baseURL (e.g.
// "http://127.0.0.1:8082"), authenticating with tokens and issuing
// requests over httpClient.
func NewClient(baseURL string, tokens TokenSource, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		tokens:     tokens,
		httpClient: httpClient,
	}
}

// APIError is returned when the ithings server reports a failure, either
// via HTTP status or its {"code":<not 200>, "msg":...} response envelope.
type APIError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ithings: %s (http %d)", e.Message, e.StatusCode)
}

// envelope is the ithings server's response wrapper: {"code":200,"data":...,"msg":"success"} -
// confirmed against a live server; code 200 means success regardless of
// HTTP status (every response observed live was itself HTTP 200).
type envelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

// do issues a GET request against path and decodes the envelope's data
// field into out. It retries once after a fresh AppHub login on 401/403.
func (c *Client) do(path string, out interface{}) error {
	token, err := c.tokens.Token()
	if err != nil {
		return err
	}

	doOnce := func(tok string) (*http.Response, []byte, error) {
		req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("accesstoken", tok)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		return resp, data, nil
	}

	resp, data, err := doOnce(token)
	if err != nil {
		return fmt.Errorf("ithings: request GET %s failed: %w", path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if err := c.tokens.Relogin(); err != nil {
			return err
		}
		token, err = c.tokens.Token()
		if err != nil {
			return err
		}
		resp, data, err = doOnce(token)
		if err != nil {
			return fmt.Errorf("ithings: request GET %s failed: %w", path, err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Message: string(data)}
	}

	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("ithings: decode response for GET %s: %w", path, err)
	}
	if env.Code != 200 {
		return &APIError{StatusCode: resp.StatusCode, Code: env.Code, Message: env.Msg}
	}

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("ithings: decode data for GET %s: %w", path, err)
		}
	}
	return nil
}

// SubDevice is one sub-device ("子设备") registered under a parent
// (EdgeID, a regular DeviceOn-EIM device endpoint), as returned by
// ListSubDevices. Field shape confirmed against a live server - it's a
// subset of AppHub-v2.3.0.md's documented shape (the doc additionally
// shows "ronchge"/"reliable" string fields on some server versions, not
// present in this environment's response, so they're omitted here rather
// than assumed).
type SubDevice struct {
	DeviceID        string `json:"deviceId"`
	Name            string `json:"name"`
	EdgeID          string `json:"edgeID"`
	Description     string `json:"description"`
	DeviceType      string `json:"deviceType"`
	DeviceModelRef  string `json:"deviceModelRef"`
	ProtocolType    string `json:"protocolType"`
	Protocol        string `json:"protocol"`
	CreateTimeStamp int64  `json:"createTimeStamp"`
	UpdateTimeStamp int64  `json:"updateTimeStamp"`
	DeviceStatus    string `json:"deviceStatus"`
	State           string `json:"state"`
	DeviceModelID   int64  `json:"deviceModelId"`
}

// ListSubDevices returns every sub-device registered on the server,
// across every parent device. Not documented in AppHub-v2.3.0.md;
// confirmed against a live server.
func (c *Client) ListSubDevices() ([]SubDevice, error) {
	var devices []SubDevice
	if err := c.do("/v1/ithings", &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// TwinProperty is one reported (SubDeviceTwin) or historical
// (QueryTwinHistory) sample of a sub-device property: Svc/Pn identify
// which service/property it belongs to, Val is the raw reported value
// (kept as raw JSON rather than a fixed Go type - confirmed against a
// live server that its JSON type varies by property: plain numbers here,
// but AppHub-v2.3.0.md's own historical-query example shows a numeric
// *string* for the same shape on a different property, so nothing safe
// can be assumed about it beyond "valid JSON"), and Ts is the
// Unix-millisecond timestamp it was recorded at.
type TwinProperty struct {
	Svc string          `json:"svc"`
	Pn  string          `json:"pn"`
	Val json.RawMessage `json:"val"`
	Ts  int64           `json:"ts"`
}

// subDeviceTwin is GET /v1/ithings/twin/{deviceId}'s data shape.
type subDeviceTwin struct {
	Reported []TwinProperty `json:"reported"`
}

// SubDeviceTwin returns a sub-device's "digital twin" - the current
// real-time value of every property it has ever reported, which also
// serves as the list of that device's services (Svc) and properties
// (Pn), since there is no separate schema-only endpoint wrapped here.
// Confirmed against a live server; an offline/never-reported sub-device
// returns an empty slice rather than an error (the server's own "data"
// field is a bare JSON null in that case).
func (c *Client) SubDeviceTwin(deviceID string) ([]TwinProperty, error) {
	path := "/v1/ithings/twin/" + url.PathEscape(deviceID)
	var twin subDeviceTwin
	if err := c.do(path, &twin); err != nil {
		return nil, err
	}
	if twin.Reported == nil {
		return []TwinProperty{}, nil
	}
	return twin.Reported, nil
}

// QueryTwinHistory returns historical samples of one specific
// service+property on a sub-device. beginTs/endTs are Unix-millisecond
// bounds (0 leaves that bound unset - sent as an empty string, confirmed
// against a live server to be accepted); count is an optional cap on how
// many samples come back (0 omits it).
//
// CAVEAT: confirmed against a live server that beginTs/endTs/count are
// currently accepted but have NO observable filtering effect - every
// combination tested (including a year-2000 and a year-2100 range)
// returned the identical most-recent ~20 samples. They're still sent
// (matching AppHub-v2.3.0.md's documented request shape and a real
// packet capture) in case a different server version honors them, but
// don't rely on them actually narrowing the result set today.
func (c *Client) QueryTwinHistory(deviceID, serviceName, propertyName string, beginTs, endTs int64, count int) ([]TwinProperty, error) {
	q := url.Values{}
	q.Set("type", "property")
	q.Set("deviceId", deviceID)
	q.Set("serviceName", serviceName)
	q.Set("name", propertyName)
	q.Set("beginTs", optionalMillis(beginTs))
	q.Set("endTs", optionalMillis(endTs))
	if count > 0 {
		q.Set("count", strconv.Itoa(count))
	} else {
		q.Set("count", "")
	}

	var props []TwinProperty
	if err := c.do("/v1/ithings/queryTwin/condition?"+q.Encode(), &props); err != nil {
		return nil, err
	}
	return props, nil
}

func optionalMillis(ms int64) string {
	if ms == 0 {
		return ""
	}
	return strconv.FormatInt(ms, 10)
}

// deviceEventPage is GET /v1/deviceEvent/byPage's data shape - not
// documented in AppHub-v2.3.0.md at all, confirmed against a live server
// via a real packet capture. List's element shape is kept as raw JSON:
// no test sub-device in the environment this was verified against had
// any recorded events, so its populated shape is unconfirmed.
type deviceEventPage struct {
	List  []json.RawMessage `json:"list"`
	Total int64             `json:"total"`
}

// ListDeviceEvents returns one page of a sub-device's recorded events
// (edgeID and/or deviceID scope the query - pass "" to leave either
// unset), plus the total count across all pages. beginTs/endTs are
// Unix-millisecond bounds (0 leaves that bound unset). Not documented in
// AppHub-v2.3.0.md; confirmed against a live server to respond
// successfully, but every sub-device available to test against had zero
// recorded events, so a populated response's exact shape (beyond the
// {"list":[...],"total":N} wrapper) is unconfirmed - list entries are
// therefore returned as raw JSON rather than a typed struct.
func (c *Client) ListDeviceEvents(edgeID, deviceID string, page, limit int, beginTs, endTs int64) ([]json.RawMessage, int64, error) {
	q := url.Values{}
	q.Set("currentPage", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("edgeId", edgeID)
	q.Set("deviceId", deviceID)
	q.Set("beginTs", optionalMillis(beginTs))
	q.Set("endTs", optionalMillis(endTs))

	var result deviceEventPage
	if err := c.do("/v1/deviceEvent/byPage?"+q.Encode(), &result); err != nil {
		return nil, 0, err
	}
	return result.List, result.Total, nil
}
