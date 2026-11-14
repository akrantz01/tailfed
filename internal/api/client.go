package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/akrantz01/tailfed/internal/http/types"
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

	if client == nil {
		client = &http.Client{}
	}
	if client.Timeout == 0 {
		client.Timeout = 10 * time.Second
	}

	return &Client{
		logger: logrus.WithField("component", "api"),
		inner:  client,
		base:   base,
	}, nil
}

func (c *Client) GetToken(ctx context.Context, profile string) (*types.TokenResponse, error) {
	logger := c.logger.WithField("request.path", "/token").WithField("request.method", "GET")

	url := c.base.JoinPath("/token")
	query := url.Query()
	query.Add("profile", profile)
	url.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "tailfed-client/")
	req.Header.Set("Sec-Tailfed", "1")

	logger.Debug("sending request...")
	res, err := c.inner.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			logger.WithError(err).Warn("failed to close response body")
		}
	}()
	logger.WithField("status", res.StatusCode).Debug("got response")

	if res.StatusCode == http.StatusOK {
		var data types.TokenResponse
		if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return &data, nil
	} else {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		return nil, &ApiError{
			Status: res.StatusCode,
			Reason: string(body),
		}
	}
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

type ApiError struct {
	Status int
	Reason string
}

var _ error = (*ApiError)(nil)

func (a *ApiError) Error() string {
	return fmt.Sprintf("invalid response: %s (status: %d)", a.Reason, a.Status)
}
