package historian

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Azure/go-ntlmssp"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

func New(baseURL, username, password string, logger *slog.Logger) *Client {
	transport := ntlmssp.Negotiator{RoundTripper: &http.Transport{}}
	httpClient := &http.Client{
		Transport: &ntlmsspWrapper{
			inner:    transport,
			username: username,
			password: password,
		},
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		logger:     logger,
	}
}

func (c *Client) Get(ctx context.Context, path, query string, dest any) error {
	full := fmt.Sprintf("%s/Historian/v2/%s?%s", c.baseURL, path, query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return fmt.Errorf("historian API %d: %s", resp.StatusCode, snippet)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) BaseURL() string { return c.baseURL }

type ntlmsspWrapper struct {
	inner    ntlmssp.Negotiator
	username string
	password string
}

func (w *ntlmsspWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(w.username, w.password)
	return w.inner.RoundTrip(req)
}
