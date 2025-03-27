package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Client provides access to the Tailfed API
type Client struct {
	logger logrus.FieldLogger
	inner  *http.Client
}

// NewClient creates a new Tailfed API client
func NewClient() *Client {
	return &Client{
		logger: logrus.WithField("component", "api"),
		// TODO: allow customizing client
		inner: http.DefaultClient,
	}
}

// Start begins the ID token issuance process
func (c *Client) Start(ctx context.Context, node string, port int) (*StartResponse, error) {
	return doRequest[StartResponse](c, ctx, "start", "/start", &StartRequest{node, port})
}

// Finalize attempts to finish the request flow and issue a token
func (c *Client) Finalize(ctx context.Context, id string) (string, error) {
	res, err := doRequest[FinalizeResponse](c, ctx, "finalize", "/finalize", &FinalizeRequest{id})
	if err != nil {
		return "", err
	}

	return res.IdentityToken, nil
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

	req, err := http.NewRequestWithContext(ctx, "POST", path, bytes.NewReader(encoded))
	if err != nil {
		logger.WithError(err).Error("failed to build request")
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

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
