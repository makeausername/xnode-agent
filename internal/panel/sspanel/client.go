package sspanel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/makeausername/xnode-agent/internal/panel"
	"github.com/makeausername/xnode-agent/pkg/nodeapi"
)

const (
	defaultHTTPTimeout = 15 * time.Second

	enrollPath      = "/node/api/v1/enroll"
	configPath      = "/node/api/v1/config"
	usersPath       = "/node/api/v1/users"
	detectRulesPath = "/node/api/v1/detect-rules"
	runtimePath     = "/node/api/v1/runtime"
	trafficPath     = "/node/api/v1/traffic"
	onlinePath      = "/node/api/v1/online"
	heartbeatPath   = "/node/api/v1/heartbeat"
)

type Client struct {
	PanelURL   string
	Token      string
	HTTPClient *http.Client
}

var _ panel.Client = (*Client)(nil)

func NewClient(panelURL string, token string) *Client {
	return NewClientWithHTTPClient(panelURL, token, &http.Client{
		Timeout: defaultHTTPTimeout,
	})
}

func NewClientWithHTTPClient(panelURL string, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultHTTPTimeout,
		}
	}

	return &Client{
		PanelURL:   strings.TrimRight(panelURL, "/"),
		Token:      token,
		HTTPClient: httpClient,
	}
}

func (c *Client) Enroll(ctx context.Context, req nodeapi.EnrollRequest) (nodeapi.EnrollResponse, error) {
	return postJSON[nodeapi.EnrollResponse](ctx, c, enrollPath, req)
}

func (c *Client) GetConfig(ctx context.Context) (nodeapi.NodeConfig, error) {
	return getJSON[nodeapi.NodeConfig](ctx, c, configPath)
}

func (c *Client) GetUsers(ctx context.Context, etag string) ([]nodeapi.UserInfo, string, error) {
	return getJSONWithETag[[]nodeapi.UserInfo](ctx, c, usersPath, etag)
}

func (c *Client) GetDetectRules(ctx context.Context, etag string) ([]nodeapi.DetectRule, string, error) {
	return getJSONWithETag[[]nodeapi.DetectRule](ctx, c, detectRulesPath, etag)
}

func (c *Client) ReportRuntime(ctx context.Context, report nodeapi.RuntimeReport) error {
	return postJSONNoData(ctx, c, runtimePath, report)
}

func (c *Client) ReportTraffic(ctx context.Context, report nodeapi.TrafficReport) error {
	return postJSONNoData(ctx, c, trafficPath, report)
}

func (c *Client) ReportOnline(ctx context.Context, report nodeapi.OnlineReport) error {
	return postJSONNoData(ctx, c, onlinePath, report)
}

func (c *Client) ReportHeartbeat(ctx context.Context, report nodeapi.HeartbeatReport) error {
	return postJSONNoData(ctx, c, heartbeatPath, report)
}

func getJSON[T any](ctx context.Context, c *Client, path string) (T, error) {
	var zero T

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return zero, err
	}

	resp, err := c.do(req, http.MethodGet, path)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	if err := ensureOK(resp, http.MethodGet, path); err != nil {
		return zero, err
	}

	return decodeAPIResponse[T](resp.Body, http.MethodGet, path)
}

func getJSONWithETag[T any](ctx context.Context, c *Client, path string, etag string) (T, string, error) {
	var zero T

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return zero, "", err
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.do(req, http.MethodGet, path)
	if err != nil {
		return zero, "", err
	}
	defer resp.Body.Close()

	responseETag := resp.Header.Get("ETag")
	if resp.StatusCode == http.StatusNotModified {
		return zero, responseETag, nil
	}
	if err := ensureOK(resp, http.MethodGet, path); err != nil {
		return zero, responseETag, err
	}

	data, err := decodeAPIResponse[T](resp.Body, http.MethodGet, path)
	if err != nil {
		return zero, responseETag, err
	}
	return data, responseETag, nil
}

func postJSON[T any](ctx context.Context, c *Client, path string, payload any) (T, error) {
	var zero T

	req, err := c.newRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return zero, err
	}

	resp, err := c.do(req, http.MethodPost, path)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	if err := ensureOK(resp, http.MethodPost, path); err != nil {
		return zero, err
	}

	return decodeAPIResponse[T](resp.Body, http.MethodPost, path)
}

func postJSONNoData(ctx context.Context, c *Client, path string, payload any) error {
	_, err := postJSON[json.RawMessage](ctx, c, path, payload)
	return err
}

func (c *Client) newRequest(ctx context.Context, method string, path string, payload any) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return nil, fmt.Errorf("encode sspanel %s %s request body: %w", method, path, err)
		}
		body = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, c.PanelURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create sspanel %s %s request: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	return req, nil
}

func (c *Client) do(req *http.Request, method string, path string) (*http.Response, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultHTTPTimeout,
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send sspanel %s %s request: %w", method, path, err)
	}
	return resp, nil
}

func ensureOK(resp *http.Response, method string, path string) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	detail := statusDetail(resp.StatusCode)
	errMsg := fmt.Sprintf("sspanel %s %s failed: HTTP %d %s", method, path, resp.StatusCode, detail)

	envelope, err := decodeErrorEnvelope(resp.Body)
	if err == nil && hasErrorDetail(envelope) {
		errMsg = fmt.Sprintf("%s: code=%q msg=%q request_id=%q", errMsg, envelope.Code, envelope.Msg, envelope.RequestID)
	}

	return fmt.Errorf("%s", errMsg)
}

func decodeAPIResponse[T any](body io.Reader, method string, path string) (T, error) {
	var zero T
	var envelope nodeapi.APIResponse[T]

	if err := json.NewDecoder(body).Decode(&envelope); err != nil {
		return zero, fmt.Errorf("decode sspanel %s %s response: invalid JSON: %w", method, path, err)
	}
	if envelope.Ret != 1 {
		return zero, fmt.Errorf(
			"sspanel %s %s returned ret=%d: code=%q msg=%q request_id=%q",
			method,
			path,
			envelope.Ret,
			envelope.Code,
			envelope.Msg,
			envelope.RequestID,
		)
	}

	return envelope.Data, nil
}

func decodeErrorEnvelope(body io.Reader) (nodeapi.APIResponse[json.RawMessage], error) {
	var envelope nodeapi.APIResponse[json.RawMessage]
	if err := json.NewDecoder(body).Decode(&envelope); err != nil {
		return envelope, err
	}
	return envelope, nil
}

func hasErrorDetail(envelope nodeapi.APIResponse[json.RawMessage]) bool {
	return envelope.Code != "" || envelope.Msg != "" || envelope.RequestID != ""
}

func statusDetail(statusCode int) string {
	switch statusCode {
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusConflict:
		return "Conflict"
	case http.StatusTooManyRequests:
		return "Too Many Requests"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	default:
		if text := http.StatusText(statusCode); text != "" {
			return text
		}
		return "unexpected status"
	}
}
