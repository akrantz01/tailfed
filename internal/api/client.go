package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/akrantz01/tailfed/internal/types"
	"github.com/sirupsen/logrus"
)

// Client provides access to the Tailfed API
type Client struct {
	logger logrus.FieldLogger

	inner *http.Client
	base  *url.URL
}

// NewClient creates a new Tailfed API client
func NewClient(baseUrl string) (*Client, error) {
	base, err := parseBaseUrl(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid API base url: %w", err)
	}

	return &Client{
		logger: logrus.WithField("component", "api"),
		// TODO: allow customizing client
		inner: http.DefaultClient,
		base:  base,
	}, nil
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

	return doRequest[types.StartResponse](c, ctx, "start", "/start", &types.StartRequest{Node: node, Ports: ports})
}

// Finalize attempts to finish the request flow and issue a token
func (c *Client) Finalize(ctx context.Context, id string) (string, error) {
	res, err := doRequest[types.FinalizeResponse](c, ctx, "finalize", "/finalize", &types.FinalizeRequest{ID: id})
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

// doRequest makes a request to the Tailfed server. This should be a method, but Go does not support generics
// in methods yet so we make do
func doRequest[R any](c *Client, ctx context.Context, name, path string, body any) (*R, error) {
	logger := c.logger.WithFields(map[string]any{
		"request": name,
		"path":    path,
		"method":  "POST",
	})

	encoded, err := json.Marshal(body)
	if err != nil {
		logger.WithError(err).Panic("failed to encode body (this should never happen)")
	}
	logger.WithField("body", body).Trace("encoded body")

	req, err := http.NewRequestWithContext(ctx, "POST", c.base.JoinPath(path).String(), bytes.NewReader(encoded))
	if err != nil {
		logger.WithError(err).Error("failed to build request")
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	logger.Debug("sending request...")
	res, err := c.inner.Do(req)
	if err != nil {
		logger.WithError(err).Error("failed to send request")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer res.Body.Close()
	logger.WithField("status", res.StatusCode).Debug("got response")

	var resBody types.Response[R]
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		logger.WithError(err).Error("failed to deserialize response")
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}
	logger.WithField("body", resBody).Trace("decoded response")

	if resBody.Success {
		return resBody.Data, nil
	}

	return nil, fmt.Errorf("http error: %s (code: %d)", resBody.Error, res.StatusCode)
}
