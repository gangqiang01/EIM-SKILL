// Package repo is an HTTP client for the m2m-management-edge "software
// repository" server that DeviceLink/AppHub uses to store OTA-able
// packages. It runs alongside the main AppHub server and, per the
// DeviceLink web console (assets/js/baseApi.js repoApiGet/repoApiPost),
// accepts the same accesstoken issued by AppHub's /login endpoint - there
// is no separate repo-server login.
package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TokenSource supplies the accesstoken used to authenticate against the
// repo server, refreshing it on demand. *apphub.Client satisfies this.
type TokenSource interface {
	Token() (string, error)
	Relogin() error
}

// Client talks to the m2m-management-edge repo server over HTTP.
type Client struct {
	baseURL    string
	tokens     TokenSource
	httpClient *http.Client
}

// NewClient creates a Client for the repo server at baseURL (e.g.
// "http://127.0.0.1:30001"), authenticating with tokens.
func NewClient(baseURL string, tokens TokenSource) *Client {
	return &Client{
		baseURL: baseURL,
		tokens:  tokens,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError is returned when the repo server reports a failure, either via
// HTTP status or its {"status":"error", ...} response envelope.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("repo: %s (http %d)", e.Message, e.StatusCode)
}

// envelope is the repo server's response wrapper: {"status":"success"|"error","data":...,"count":...}
type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Count  int64           `json:"count"`
}

// do issues a request against path, optionally JSON-encoding body, and
// decodes the envelope's data field into out. It retries once after a
// fresh AppHub login on 401/403.
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
		return fmt.Errorf("repo: request %s %s failed: %w", method, path, err)
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
			return fmt.Errorf("repo: request %s %s failed: %w", method, path, err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Message: string(data)}
	}

	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("repo: decode response for %s %s: %w", method, path, err)
	}

	if env.Status != "success" {
		var msg string
		if json.Unmarshal(env.Data, &msg) != nil || msg == "" {
			msg = string(env.Data)
		}
		return &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("repo: decode data for %s %s: %w", method, path, err)
		}
	}
	return nil
}

// ListLinuxPackages returns every Linux software package available in the
// repo - i.e. the packages that can be deployed/OTA-updated to a device.
func (c *Client) ListLinuxPackages() ([]LinuxPackage, error) {
	var pkgs []LinuxPackage
	if err := c.do(http.MethodGet, "/repoLinuxPkg", nil, &pkgs); err != nil {
		return nil, err
	}
	return pkgs, nil
}

// ListStorages returns every storage backend configured on the repo server.
// Note: the server never returns the secret key on this listing endpoint.
func (c *Client) ListStorages() ([]Storage, error) {
	var storages []Storage
	if err := c.do(http.MethodGet, "/storage", nil, &storages); err != nil {
		return nil, err
	}
	return storages, nil
}

// AddS3CompatibleStorage registers a MinIO/AWS S3/Alibaba OSS storage
// backend. platform is stored verbatim as the backend's "type" - the server
// enforces no fixed enum here, so use a consistent convention such as
// "minio", "s3", or "oss".
func (c *Client) AddS3CompatibleStorage(platform, endpointURL, bucket, accessKey, secretKey string) error {
	body := addStorageRequest{
		Platform:  platform,
		BlobURL:   endpointURL,
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
	return c.do(http.MethodPost, "/storage", body, nil)
}

// AddAzureBlobStorage registers an Azure Blob storage backend.
func (c *Client) AddAzureBlobStorage(container, connectionString string) error {
	body := addStorageRequest{
		Platform:         "azure",
		Container:        container,
		ConnectionString: connectionString,
	}
	return c.do(http.MethodPost, "/storage", body, nil)
}

// SetActiveStorage re-validates and activates the storage backend
// identified by sid - it becomes the one used for all subsequent uploads
// (OTA packages, app icons, etc).
func (c *Client) SetActiveStorage(sid int64) error {
	return c.do(http.MethodPost, fmt.Sprintf("/storage/%d", sid), nil, nil)
}

// DeleteStorage removes a storage backend and every repo object stored in it.
func (c *Client) DeleteStorage(sid int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/storage/%d", sid), nil, nil)
}
