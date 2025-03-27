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
func (c *Client) Start(ctx context.Context, node string, addresses []string) (*StartResponse, error) {
	bindings := make([]PortBinding, 0, len(addresses))
	for _, address := range addresses {
		addr := netip.MustParseAddrPort(address)
		bindings = append(bindings, PortBinding{
			Port:    addr.Port(),
			Network: NetworkFromAddrPort(addr),
		})
	}

	return doRequest[StartResponse](c, ctx, "start", "/start", &StartRequest{node, bindings})
}

// Finalize attempts to finish the request flow and issue a token
func (c *Client) Finalize(ctx context.Context, id string) (string, error) {
	res, err := doRequest[FinalizeResponse](c, ctx, "finalize", "/finalize", &FinalizeRequest{id})
	if err != nil {
		return "", err
	}

	return res.IdentityToken, nil
}

func parseBaseUrl(baseUrl string) (*url.URL, error) {
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

	var resBody Response[R]
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
