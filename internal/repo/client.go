// Package repo is an HTTP client for the m2m-management-edge "software
// repository" server that DeviceOn-EIM uses to store OTA-able
// packages (Android APKs, Windows exe/zip installers, Docker compose
// files, generic files, Linux packages). It runs alongside the main
// AppHub server and, per the DeviceLink web console
// (assets/js/baseApi.js repoApiGet), accepts the same accesstoken issued
// by AppHub's /login endpoint - there is no separate repo-server login.
package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
// "http://127.0.0.1:30001"), authenticating with tokens and issuing
// requests over httpClient.
func NewClient(baseURL string, tokens TokenSource, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		tokens:     tokens,
		httpClient: httpClient,
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

// do issues a GET request against path and decodes the envelope's data
// field into out. It retries once after a fresh AppHub login on 401/403.
func (c *Client) do(method, path string, out interface{}) error {
	token, err := c.tokens.Token()
	if err != nil {
		return err
	}

	doOnce := func(tok string) (*http.Response, []byte, error) {
		req, err := http.NewRequest(method, c.baseURL+path, nil)
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
	if err := c.do(http.MethodGet, "/repoLinuxPkg", &pkgs); err != nil {
		return nil, err
	}
	return pkgs, nil
}

// ListAndroidApps returns every Android APK available in the repo.
func (c *Client) ListAndroidApps() ([]AndroidApp, error) {
	var apps []AndroidApp
	if err := c.do(http.MethodGet, "/repoapps", &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// ListWindowsPackages returns Windows installer packages available in the
// repo. pkgType filters to "exe" or "zip" (the compressed-package
// discriminator observed in RepoExe.type); pass "" for both.
func (c *Client) ListWindowsPackages(pkgType string) ([]WindowsPackage, error) {
	path := "/repoexe"
	if pkgType != "" {
		q := url.Values{}
		q.Set("type", pkgType)
		path = "/exeByType?" + q.Encode()
	}
	var pkgs []WindowsPackage
	if err := c.do(http.MethodGet, path, &pkgs); err != nil {
		return nil, err
	}
	return pkgs, nil
}

// ListDockerComposePackages returns every Docker compose/swarm package
// available in the repo.
func (c *Client) ListDockerComposePackages() ([]DockerComposePackage, error) {
	var pkgs []DockerComposePackage
	if err := c.do(http.MethodGet, "/repoDockerCompose", &pkgs); err != nil {
		return nil, err
	}
	return pkgs, nil
}

// ListFiles returns generic files available in the repo for file-deploy,
// filtered by fileType (required by the server - there is no "list all"
// endpoint for this package type).
func (c *Client) ListFiles(fileType string) ([]GeneralFile, error) {
	q := url.Values{}
	q.Set("type", fileType)
	var files []GeneralFile
	if err := c.do(http.MethodGet, "/repofiles/bytype?"+q.Encode(), &files); err != nil {
		return nil, err
	}
	return files, nil
}
