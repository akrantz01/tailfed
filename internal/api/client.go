package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/akrantz01/tailfed/internal/types"
	"github.com/akrantz01/tailfed/internal/version"
	"github.com/sirupsen/logrus"
)

// Client provides access to the Tailfed API
type Client struct {
	logger logrus.FieldLogger

	inner *http.Client
	base  *url.URL
}

// NewClient creates a new Tailfed API client
func NewClient(client *http.Client, baseUrl string) (*Client, error) {
	base, err := parseBaseUrl(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid API base url: %w", err)
	}

	return &Client{
		logger: logrus.WithField("component", "api"),
		inner:  client,
		base:   base,
	}, nil
}

// GetVersion retrieves version information for the remove API
func (c *Client) GetVersion(ctx context.Context) (*version.Info, error) {
	info, _, err := doRequest[version.Info](c, ctx, "get-version", "GET", "/version.json", nil)
	return info, err
}

// GetConfig retrieves the daemon config
func (c *Client) GetConfig(ctx context.Context) (*types.ConfigResponse, error) {
	info, _, err := doRequest[types.ConfigResponse](c, ctx, "get-config", "GET", "/config.json", nil)
	return info, err
}

// Start begins the ID token issuance process
func (c *Client) Start(ctx context.Context, node string, addresses []string) (*types.StartResponse, error) {
	ports := types.Ports{}
	for _, address := range addresses {
		addr := netip.MustParseAddrPort(address)
		switch {
		case addr.Addr().Is4():
			ports.IPv4 = addr.Port()
		case addr.Addr().Is6():
			ports.IPv6 = addr.Port()
		default:
			c.logger.WithField("address", address).Warn("found address that is neither ipv4 nor ipv6")
		}
	}

	return doApiRequest[types.StartResponse](c, ctx, "start", "POST", "/start", &types.StartRequest{Node: node, Ports: ports})
}

// Finalize attempts to finish the request flow and issue a token
func (c *Client) Finalize(ctx context.Context, id string) (string, error) {
	res, err := doApiRequest[types.FinalizeResponse](c, ctx, "finalize", "POST", "/finalize", &types.FinalizeRequest{ID: id})
	if err != nil {
		return "", err
	}

	return res.IdentityToken, nil
}

func parseBaseUrl(baseUrl string) (*url.URL, error) {
	baseUrl = strings.TrimSpace(baseUrl)
	if len(baseUrl) == 0 {
		return nil, errors.New("not configured")
	}

	base, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	if base.Scheme != "https" && base.Scheme != "http" {
		return nil, errors.New("scheme must be 'http' or 'https'")
	}

	return base, nil
}

// doApiRequest makes a request to the Tailfed server. It expects a response wrapped in a [types.Response] to determine
// whether the action was a success or failure.
func doApiRequest[R any](c *Client, ctx context.Context, name, method, path string, body any) (*R, error) {
	res, status, err := doRequest[types.Response[R]](c, ctx, name, method, path, body)
	if err != nil {
		return nil, err
	}

	if res.Success {
		return res.Data, nil
	}

	return nil, &Error{
		message: res.Error,
		status:  status,
	}
}

// doRequest makes a request to the Tailfed server. This should be a method, but Go does not support generics
// in methods yet so we make do
func doRequest[R any](c *Client, ctx context.Context, name, method, path string, body any) (*R, int, error) {
	logger := c.logger.WithFields(map[string]any{
		"request": name,
		"path":    path,
		"method":  method,
	})

	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			logger.WithError(err).Panic("failed to encode body (this should never happen)")
		}
		logger.WithField("body", body).Trace("encoded body")

		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base.JoinPath(path).String(), reqBody)
	if err != nil {
		logger.WithError(err).Error("failed to build request")
		return nil, 0, fmt.Errorf("failed to build request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	logger.Debug("sending request...")
	res, err := c.inner.Do(req)
	if err != nil {
		logger.WithError(err).Error("failed to send request")
		return nil, 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer res.Body.Close()
	logger.WithField("status", res.StatusCode).Debug("got response")

	var data R
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		logger.WithError(err).Error("failed to deserialize response")
		return nil, 0, fmt.Errorf("failed to deserialize response: %w", err)
	}
	logger.WithField("body", data).Trace("decoded response")

	return &data, res.StatusCode, nil
}
