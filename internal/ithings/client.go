// Package ithings is an HTTP client for the "ithings" Go microservice that
// backs DeviceLink/AppHub's sub-device (Modbus/OPC UA/IEC104/S7/SNMP
// gateway) and data-forward (AMQP/MQTT/Kafka) features. It runs alongside
// the main AppHub server (by convention on port 8082) and, per its own
// auth middleware (webserver/middlewares/middlewares.go Cors(), which
// forwards the accesstoken to AppHub's verifyToken message), accepts the
// same accesstoken issued by AppHub's /login endpoint.
package ithings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// TokenSource supplies the accesstoken used to authenticate against the
// ithings server, refreshing it on demand. *apphub.Client satisfies this.
type TokenSource interface {
	Token() (string, error)
	Relogin() error
}

// Client talks to the ithings server over HTTP.
type Client struct {
	baseURL    string
	tokens     TokenSource
	httpClient *http.Client
}

// NewClient creates a Client for the ithings server at baseURL (e.g.
// "http://127.0.0.1:8082"), authenticating with tokens.
func NewClient(baseURL string, tokens TokenSource) *Client {
	return &Client{
		baseURL: baseURL,
		tokens:  tokens,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError is returned when the ithings server reports a failure, either
// via HTTP status or its {"code":<>, "msg":...} response envelope.
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ithings: %s (code %d)", e.Message, e.Code)
}

// envelope is ithings' response wrapper: {"code":200,"data":...,"msg":"success"}.
// The HTTP status code always equals Code (webserver/types/http_response.go Result()).
type envelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

// do issues a request against path, optionally JSON-encoding body, and
// decodes the envelope's data field into out. It retries once after a
// fresh AppHub login on 401.
func (c *Client) do(method, path string, body interface{}, out interface{}) error {
	token, err := c.tokens.Token()
	if err != nil {
		return err
	}

	doOnce := func(tok string) (*http.Response, []byte, error) {
		var reader io.Reader
		if body != nil {
			data, err := json.Marshal(body)
			if err != nil {
				return nil, nil, err
			}
			reader = bytes.NewReader(data)
		}

		req, err := http.NewRequest(method, c.baseURL+path, reader)
		if err != nil {
			return nil, nil, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("accesstoken", tok)

		return c.send(req)
	}

	resp, data, err := doOnce(token)
	if err != nil {
		return fmt.Errorf("ithings: request %s %s failed: %w", method, path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.tokens.Relogin(); err != nil {
			return err
		}
		token, err = c.tokens.Token()
		if err != nil {
			return err
		}
		resp, data, err = doOnce(token)
		if err != nil {
			return fmt.Errorf("ithings: request %s %s failed: %w", method, path, err)
		}
	}

	return decodeEnvelope(resp, data, out)
}

// doMultipart uploads a file (plus optional extra form fields) to path via
// POST multipart/form-data. fieldName is the multipart field the server
// expects the file under (e.g. "file").
func (c *Client) doMultipart(path, fieldName, filePath string, extraFields map[string]string, out interface{}) error {
	token, err := c.tokens.Token()
	if err != nil {
		return err
	}

	buildBody := func() (*bytes.Buffer, string, error) {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, "", err
		}
		defer file.Close()

		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		part, err := w.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, file); err != nil {
			return nil, "", err
		}
		for k, v := range extraFields {
			if err := w.WriteField(k, v); err != nil {
				return nil, "", err
			}
		}
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, w.FormDataContentType(), nil
	}

	doOnce := func(tok string) (*http.Response, []byte, error) {
		buf, contentType, err := buildBody()
		if err != nil {
			return nil, nil, err
		}
		req, err := http.NewRequest(http.MethodPost, c.baseURL+path, buf)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("accesstoken", tok)
		return c.send(req)
	}

	resp, data, err := doOnce(token)
	if err != nil {
		return fmt.Errorf("ithings: upload to %s failed: %w", path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.tokens.Relogin(); err != nil {
			return err
		}
		token, err = c.tokens.Token()
		if err != nil {
			return err
		}
		resp, data, err = doOnce(token)
		if err != nil {
			return fmt.Errorf("ithings: upload to %s failed: %w", path, err)
		}
	}

	return decodeEnvelope(resp, data, out)
}

func (c *Client) send(req *http.Request) (*http.Response, []byte, error) {
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

// getRaw issues a GET and returns the envelope's data field verbatim, for
// endpoints where we deliberately don't model the exact response shape.
func (c *Client) getRaw(path string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.do(http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// --- device models -------------------------------------------------------

// ListDeviceModels returns every sub-device model (raw server JSON).
func (c *Client) ListDeviceModels() (json.RawMessage, error) {
	return c.getRaw("/v1/deviceModel")
}

// GetDeviceModel returns full detail (modules, properties, events) for the
// named model (raw server JSON).
func (c *Client) GetDeviceModel(name string) (json.RawMessage, error) {
	return c.getRaw("/v1/deviceModel/allInfo/" + url.PathEscape(name))
}

// ImportDeviceModelExcel bulk-creates a device model (plus its modules,
// properties and events) from an Excel workbook with sheets "deviceModel"
// (name, description, manufacturer, industry, dataFormat), "property"
// (module, name, description, dataType, maxValue, minValue, writeAble,
// unit), and "event" (module, name, description).
func (c *Client) ImportDeviceModelExcel(filePath string) error {
	return c.doMultipart("/v1/deviceModel/importExcel", "file", filePath, nil, nil)
}

// --- sub-devices -----------------------------------------------------------

// ListSubDevices returns every sub-device (raw server JSON).
func (c *Client) ListSubDevices() (json.RawMessage, error) {
	return c.getRaw("/v1/ithings")
}

// GetSubDevice returns full detail (services, properties, events) for a
// sub-device (raw server JSON).
func (c *Client) GetSubDevice(deviceID string) (json.RawMessage, error) {
	return c.getRaw("/v1/ithings/allInfo/" + url.PathEscape(deviceID))
}

// ListProtocolTypes returns the protocol types the server currently
// supports for sub-devices (raw server JSON) - observed values include
// "modbus" and "snmp"; "opcua"/"iec104"/"s7" were not found registered in
// this deployment (see README).
func (c *Client) ListProtocolTypes() (json.RawMessage, error) {
	return c.getRaw("/v1/protocolType")
}

// ImportSubDeviceExcel bulk-creates sub-devices (plus their property/event
// access config) from an Excel workbook with sheets "subDevice"
// (deviceId, name, edgeId, description, deviceType,
// liftTimeOfDesiredValue, deviceModel, protocolType, protocol, tags,
// reliable, reportOnChange), "propertyConfig", "eventConfig", and
// "commandConfig". One device is created per resolved edge/gateway: pass
// either edgeIDs (explicit endpoint list) or groupName (all devices in
// that group), not both.
func (c *Client) ImportSubDeviceExcel(filePath string, edgeIDs []string, groupName string) error {
	fields := map[string]string{}
	if len(edgeIDs) > 0 {
		data, err := json.Marshal(edgeIDs)
		if err != nil {
			return err
		}
		fields["edgeIds"] = string(data)
	}
	if groupName != "" {
		fields["groupName"] = groupName
	}
	return c.doMultipart("/v1/ithings/importExcel", "file", filePath, fields, nil)
}

// SetCollectionEnabled starts or stops a sub-device's data collection.
func (c *Client) SetCollectionEnabled(deviceID string, enabled bool) error {
	body := setCollectionStatusRequest{Status: enabled}
	return c.do(http.MethodPut, "/v1/ithings/status/"+url.PathEscape(deviceID), body, nil)
}

// GetTwin returns the sub-device's real-time collected data ("device
// shadow": each property's latest reported value) as raw server JSON.
func (c *Client) GetTwin(deviceID string) (json.RawMessage, error) {
	return c.getRaw("/v1/ithings/twin/" + url.PathEscape(deviceID))
}

// QueryTwinHistory returns a sub-device property's collected data samples
// between beginTs and endTs (Unix milliseconds) as raw server JSON.
func (c *Client) QueryTwinHistory(deviceID, serviceName, propertyName string, beginTs, endTs int64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("deviceId", deviceID)
	q.Set("serviceName", serviceName)
	q.Set("name", propertyName)
	q.Set("type", "reported")
	q.Set("beginTs", strconv.FormatInt(beginTs, 10))
	q.Set("endTs", strconv.FormatInt(endTs, 10))
	return c.getRaw("/v1/ithings/queryTwin/condition?" + q.Encode())
}

// --- data forward ---------------------------------------------------------

// ListDataForwards returns every configured data-forward rule (raw server JSON).
func (c *Client) ListDataForwards() (json.RawMessage, error) {
	return c.getRaw("/v1/dataForward")
}

// GetDataForward returns one data-forward rule by name (raw server JSON).
func (c *Client) GetDataForward(name string) (json.RawMessage, error) {
	return c.getRaw("/v1/dataForward/name/" + url.PathEscape(name))
}

// CreateDataForward creates an empty (disabled) data-forward rule; use
// SetDataForward to configure its source/destination afterward.
func (c *Client) CreateDataForward(name, description string) error {
	body := createDataForwardRequest{Name: name, Description: description}
	return c.do(http.MethodPost, "/v1/dataForward", body, nil)
}

// SetDataForward configures an existing data-forward rule's sources and
// destination (AMQP/Kafka/MQTT broker).
func (c *Client) SetDataForward(name, description string, sources []DataForwardSource, destination DataForwardDestination) error {
	destJSON, err := json.Marshal(destination)
	if err != nil {
		return err
	}
	body := updateDataForwardRequest{
		Name:        name,
		Description: description,
		Source:      sources,
		Destination: string(destJSON),
	}
	return c.do(http.MethodPut, "/v1/dataForward/name/"+url.PathEscape(name), body, nil)
}

// SetDataForwardEnabled enables or disables a data-forward rule.
func (c *Client) SetDataForwardEnabled(name string, enabled bool) error {
	status := "disable"
	if enabled {
		status = "enable"
	}
	body := updateDataForwardStatusRequest{Status: status}
	return c.do(http.MethodPut, "/v1/dataForward/status/name/"+url.PathEscape(name), body, nil)
}

// DeleteDataForward removes a data-forward rule by id (see ListDataForwards
// for each rule's id).
func (c *Client) DeleteDataForward(id string) error {
	return c.do(http.MethodDelete, "/v1/dataForward/"+url.PathEscape(id), nil, nil)
}

// ListDataForwardLogs returns the data-forward delivery log (raw server JSON).
func (c *Client) ListDataForwardLogs() (json.RawMessage, error) {
	return c.getRaw("/v1/dataForwardLog")
}

func decodeEnvelope(resp *http.Response, data []byte, out interface{}) error {
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("ithings: decode response (http %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || (env.Code != 0 && env.Code != 200) {
		msg := env.Msg
		if msg == "" {
			msg = string(data)
		}
		return &APIError{Code: env.Code, Message: msg}
	}

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("ithings: decode data: %w", err)
		}
	}
	return nil
}
